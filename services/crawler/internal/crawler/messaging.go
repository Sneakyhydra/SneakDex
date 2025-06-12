package crawler

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/IBM/sarama"

	"github.com/sneakyhydra/sneakdex/crawler/internal/config"
	"github.com/sneakyhydra/sneakdex/crawler/internal/crawlerrors"
	"github.com/sneakyhydra/sneakdex/crawler/internal/logger"
)

// initializeKafka sets up Kafka producer with proper configuration
func (crawler *Crawler) initializeKafka() error {
	log := logger.GetLogger()
	cfg := config.GetConfig()

	//  Create a new Sarama configuration
	kafkaConfig := sarama.NewConfig()

	// Set Kafka producer configurations
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	kafkaConfig.Producer.Retry.Max = cfg.KafkaRetryMax
	kafkaConfig.Producer.Retry.Backoff = 100 * time.Millisecond
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Return.Errors = true
	kafkaConfig.Producer.Timeout = cfg.RequestTimeout
	kafkaConfig.Net.DialTimeout = cfg.RequestTimeout
	kafkaConfig.Metadata.RefreshFrequency = 10 * time.Minute
	kafkaConfig.Producer.MaxMessageBytes = cfg.MaxContentSize

	// Create a new Kafka producer
	brokers := strings.Split(cfg.KafkaBrokers, ",")

	for attempt := 1; attempt <= cfg.KafkaRetryMax; attempt++ {
		producer, err := sarama.NewSyncProducer(brokers, kafkaConfig)
		if err == nil {
			crawler.producer = producer
			log.Info("Kafka producer initialized")
			return nil
		}
		log.Warnf("Kafka producer initialization attempt %d/%d failed: %v", attempt, cfg.KafkaRetryMax, err)
		if attempt < cfg.KafkaRetryMax {
			// Exponential backoff for retries
			backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed to create kafka producer after %d attempts. please ensure kafka is running on %s", cfg.KafkaRetryMax, strings.Join(brokers, ","))
}

// sendToKafka sends the scraped HTML content to Kafka.
func (crawler *Crawler) sendToKafka(url, html string) error {
	log := logger.GetLogger()
	cfg := config.GetConfig()

	if len(html) > cfg.MaxContentSize {
		return &crawlerrors.CrawlError{
			URL:       url,
			Operation: "kafka_send",
			Err:       fmt.Errorf("content size %d exceeds limit %d", len(html), cfg.MaxContentSize),
			Retry:     false, // No point in retrying for size limit
		}
	}

	// Create the message
	msg := &sarama.ProducerMessage{
		Topic:     cfg.KafkaTopic,
		Key:       sarama.StringEncoder(url),
		Value:     sarama.StringEncoder(html),
		Timestamp: time.Now(),
	}

	var lastErr error
	for attempt := 1; attempt <= cfg.KafkaRetryMax; attempt++ {
		_, _, err := crawler.producer.SendMessage(msg)
		if err == nil {
			crawler.stats.IncrementKafkaSuccessful()
			return nil
		}

		lastErr = err
		log.Warnf("Kafka SendMessage attempt %d/%d failed for URL %s: %v",
			attempt, cfg.KafkaRetryMax, url, err)

		// Don't sleep after the last attempt
		if attempt < cfg.KafkaRetryMax {
			// Exponential backoff for retries
			backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	crawler.stats.IncrementKafkaFailed()
	return &crawlerrors.CrawlError{
		URL:       url,
		Operation: "kafka_send",
		Err:       fmt.Errorf("failed to send to kafka after %d attempts: %w", cfg.KafkaRetryMax, lastErr),
		Retry:     true, // Mark as retriable
	}
}
