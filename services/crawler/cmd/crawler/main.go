package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/gocolly/colly/v2"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

// Config holds the crawler configuration from environment variables
type Config struct {
	KafkaBrokers string `envconfig:"KAFKA_BROKERS" default:"kafka:9092"`
	KafkaTopic   string `envconfig:"KAFKA_TOPIC_HTML" default:"raw-html"`
	StartURLs    string `envconfig:"START_URLS" default:"https://en.wikipedia.org,https://news.ycombinator.com"`
	CrawlDepth   int    `envconfig:"CRAWL_DEPTH" default:"3"`
	MaxPages     int    `envconfig:"MAX_PAGES" default:"1000"`
	LogLevel     string `envconfig:"LOG_LEVEL" default:"info"`
}

var (
	config Config
	log    = logrus.New()
)

func init() {
	// Parse environment variables
	if err := envconfig.Process("", &config); err != nil {
		log.Fatalf("Failed to process environment variables: %v", err)
	}

	// Set up logging
	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		log.Fatalf("Invalid log level: %v", err)
	}
	log.SetLevel(level)
	log.SetFormatter(&logrus.JSONFormatter{})
}

func main() {
	log.Infof("Kafka brokers: %s", config.KafkaBrokers)
	log.Infof("Start URLs: %s", config.StartURLs)

	log.Info("Starting web crawler")

	// Configure Kafka producer
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	kafkaConfig.Producer.Retry.Max = 10
	kafkaConfig.Producer.Return.Successes = true

	// Create Kafka producer
	brokers := strings.Split(config.KafkaBrokers, ",")
	producer, err := sarama.NewSyncProducer(brokers, kafkaConfig)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer func() {
		if err := producer.Close(); err != nil {
			log.Errorf("Failed to close Kafka producer: %v", err)
		}
	}()

	// Initialize web crawler
	c := colly.NewCollector(
		colly.MaxDepth(config.CrawlDepth),
		colly.Async(true),
	)

	// Set up a limit for concurrent requests
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 5,
		Delay:       1 * time.Second,
	})

	// Count pages crawled
	pageCount := 0

	// Handle found HTML pages
	c.OnHTML("html", func(e *colly.HTMLElement) {
		if pageCount >= config.MaxPages {
			return
		}

		pageCount++
		url := e.Request.URL.String()
		html, err := e.DOM.Html()
		if err != nil {
			log.Errorf("Failed to extract HTML from %s: %v", url, err)
			return
		}

		// Send to Kafka
		msg := &sarama.ProducerMessage{
			Topic: config.KafkaTopic,
			Key:   sarama.StringEncoder(url),
			Value: sarama.StringEncoder(html),
		}

		_, _, err = producer.SendMessage(msg)
		if err != nil {
			log.Errorf("Failed to send message to Kafka: %v", err)
		} else {
			log.Infof("Page sent to Kafka: %s", url)
		}
	})

	// Handle links found on the page
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		if pageCount >= config.MaxPages {
			return
		}
		link := e.Attr("href")
		e.Request.Visit(link)
	})

	// Handle errors
	c.OnError(func(r *colly.Response, err error) {
		log.Errorf("Error visiting %s: %v", r.Request.URL, err)
	})

	// Start crawling from the configured start URLs
	startURLs := strings.Split(config.StartURLs, ",")
	for _, url := range startURLs {
		err := c.Visit(url)
		if err != nil {
			log.Errorf("Failed to visit start URL %s: %v", url, err)
		}
	}

	// Wait for all crawling operations to finish
	c.Wait()

	log.Infof("Crawler finished. Processed %d pages.", pageCount)
	// Keep the process alive for hot reload
	if os.Getenv("GO_ENV") == "development" {
		log.Println("Development mode: keeping process alive for hot reload")

		// Wait for interrupt signal
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		// Or run in a loop
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Println("Crawler still running...")
				// Optionally re-run crawler here
				// runCrawler()
			case <-c:
				log.Println("Shutting down...")
				return
			}
		}
	}
	os.Exit(0)
}
