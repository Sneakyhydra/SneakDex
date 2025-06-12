package crawler

import (
	// StdLib
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	// Third-party
	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	// Internal modules
	"github.com/sneakyhydra/sneakdex/crawler/internal/config"
	"github.com/sneakyhydra/sneakdex/crawler/internal/health"
	"github.com/sneakyhydra/sneakdex/crawler/internal/logger"
	"github.com/sneakyhydra/sneakdex/crawler/internal/metrics"
	"github.com/sneakyhydra/sneakdex/crawler/internal/validator"
)

// Crawler represents the main crawler instance
type Crawler struct {
	// Core configuration and dependencies
	redisClient *redis.Client
	producer    sarama.SyncProducer
	stats       *metrics.Metrics

	// Concurrency control
	inFlightPages int64

	// URL management and filtering
	whitelist []string
	blacklist []string
	visited   sync.Map
	requeued  sync.Map
}

var crawler *Crawler
var wg sync.WaitGroup
var ctx context.Context
var cancel context.CancelFunc
var shutdown chan struct{}
var shutdownOnce sync.Once

// NewCrawler creates a new crawler instance
func New() error {
	log := logger.GetLogger()
	// Create a new context with cancellation for graceful shutdown
	newCtx, newCancel := context.WithCancel(context.Background())
	newshutdown := make(chan struct{})

	ctx = newCtx
	cancel = newCancel
	shutdown = newshutdown

	// Create the Crawler instance
	newCrawler := &Crawler{
		stats: &metrics.Metrics{StartTime: time.Now()},
	}

	// Parse whitelist and blacklist
	if config.GetConfig().URLWhitelist != "" {
		newCrawler.whitelist = strings.Split(config.GetConfig().URLWhitelist, ",")
	}
	if config.GetConfig().URLBlacklist != "" {
		newCrawler.blacklist = strings.Split(config.GetConfig().URLBlacklist, ",")
	}

	// Initialize Redis
	if err := newCrawler.initializeRedis(); err != nil {
		cancel()
		return fmt.Errorf("redis initialization failed: %w", err)
	}
	log.Info("Redis initialized successfully")

	// Initialize Kafka
	if err := newCrawler.initializeKafka(); err != nil {
		cancel()
		return fmt.Errorf("kafka initialization failed: %w", err)
	}
	log.Info("Kafka initialized successfully")

	// Initialize URL Validator
	validator.NewURLValidator(newCrawler.whitelist, newCrawler.blacklist, log)
	urlValidator := validator.GetURLValidator()

	// Configure URL Validator
	urlValidator.SetDNSCacheTimeout(10 * time.Minute)
	urlValidator.SetSkipDNSCheck(false) // Set to false for high-performance crawling. Not Safe
	urlValidator.SetAllowPrivateIPs(false)
	urlValidator.SetAllowLoopback(false)

	crawler = newCrawler
	return nil
}

func GetCrawler() *Crawler {
	return crawler
}

func (crawler *Crawler) Start() error {
	log := logger.GetLogger()
	cfg := config.GetConfig()
	urlValidator := validator.GetURLValidator()
	log.Info("Starting web crawler with persistent Redis queue. Waiting indefinitely until completion.")
	log.WithFields(logrus.Fields{
		"kafka": map[string]any{
			"kafka_brokers":   cfg.KafkaBrokers,
			"kafka_topic":     cfg.KafkaTopic,
			"kafka_retry_max": cfg.KafkaRetryMax,
		},
		"redis": map[string]any{
			"redis_host":      cfg.RedisHost,
			"redis_port":      cfg.RedisPort,
			"redis_password":  cfg.RedisPassword,
			"redis_db":        cfg.RedisDB,
			"redis_timeout":   cfg.RedisTimeout,
			"redis_retry_max": cfg.RedisRetryMax,
		},
		"crawling_behavior": map[string]any{
			"start_urls":    cfg.StartURLs,
			"max_pages":     cfg.MaxPages,
			"crawl_depth":   cfg.CrawlDepth,
			"url_whitelist": cfg.URLWhitelist,
			"url_blacklist": cfg.URLBlacklist,
		},
		"performance_and_limits": map[string]any{
			"max_concurrency":  cfg.MaxConcurrency,
			"request_timeout":  cfg.RequestTimeout,
			"request_delay":    cfg.RequestDelay,
			"max_content_size": cfg.MaxContentSize,
		},
		"application_settings": map[string]any{
			"log_level":         cfg.LogLevel,
			"user_agent":        cfg.UserAgent,
			"enable_debug":      cfg.EnableDebug,
			"health_check_port": cfg.HealthCheckPort,
		},
	}).Info("Crawler configuration")

	health.StartHealthCheck(&wg, shutdown, crawler.redisClient, &crawler.producer, crawler.stats)
	crawler.logMetricsPeriodically()

	collector := crawler.setupCollyCollector(ctx)

	// Seed Redis 'pending_urls' with configured start URLs
	startURLs := strings.SplitSeq(cfg.StartURLs, ",")
	for rawURL := range startURLs {
		url := strings.TrimSpace(rawURL)
		if !urlValidator.IsValidURL(url) {
			log.Warnf("Skipping invalid start URL: %s", url)
			continue
		}

		visitedOrPending, err := crawler.isURLVisited(url)
		if err != nil {
			log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to check start URL status in Redis, skipping")
			continue
		}
		if visitedOrPending {
			log.WithFields(logrus.Fields{"url": url}).Info("Start URL already visited or pending, skipping initial queueing")
			continue
		}

		// Add to Redis 'pending_urls' set
		_, err = crawler.redisClient.SAdd(ctx, "crawler:pending_urls", url).Result()
		if err != nil {
			log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to add start URL to Redis pending queue")
		} else {
			log.WithFields(logrus.Fields{"url": url}).Info("Added start URL to Redis pending queue")
		}
	}

	// --- Core Waiting Logic ---

	// Increment WaitGroup for the feedCollyFromRedisQueue goroutine
	wg.Add(1)
	go crawler.feedCollyFromRedisQueue(collector)

	log.Info("Crawler started. Blocking until all crawling activities are complete...")

	// Use a separate goroutine to orchestrate the waiting for Colly and the feeder.
	// This allows the main `Start` function to simply block until all work is signaled as done.
	doneChan := make(chan struct{})
	go func() {
		// First, wait for Colly to finish processing all URLs it has been given.
		// This will only complete once `feedCollyFromRedisQueue` stops feeding it URLs
		// (either due to context cancellation, max pages, or empty Redis queue).
		collector.Wait()
		log.Info("Colly collector finished processing its internal queue.")

		// Then, wait for the feedCollyFromRedisQueue goroutine to exit.
		// It will exit when its internal conditions (context, max pages + empty queue) are met.
		wg.Wait()
		log.Info("Redis queue feeder goroutine has stopped.")

		close(doneChan) // Signal that all crawling activities are complete
	}()

	// Block the main `Start()` goroutine until all crawling activities are done.
	<-doneChan
	log.Info("All crawling activities (Colly queue and Redis feeder) have completed.")

	// --- End Core Waiting Logic ---

	stats := crawler.stats.GetStats()
	log.WithFields(logrus.Fields{
		"pages_processed":  stats["pages_processed"],
		"pages_successful": stats["pages_successful"],
		"pages_failed":     stats["pages_failed"],
		"duration_seconds": stats["uptime_seconds"],
	}).Info("Crawling process completed")

	return nil
}

// Shutdown gracefully shuts down the crawler by stopping background goroutines
// and closing connections.
func (crawler *Crawler) Shutdown() {
	log := logger.GetLogger()
	shutdownOnce.Do(func() {
		log.Info("Initiating crawler shutdown...")
		close(shutdown) // Signal background goroutines to stop
		cancel()        // Signal context cancellation to active operations

		// Wait for all goroutines (health check, metrics logger) to finish.
		// Colly's Wait() should already be done or will be canceled by ctx.
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			log.Info("All background goroutines finished.")
		case <-time.After(10 * time.Second):
			log.Warn("Timeout waiting for background goroutines to finish. Some might still be running.")
		}

		// Close external connections
		if crawler.producer != nil {
			if err := crawler.producer.Close(); err != nil {
				log.Errorf("Failed to close Kafka producer: %v", err)
			} else {
				log.Info("Kafka producer closed.")
			}
		}

		if crawler.redisClient != nil {
			if err := crawler.redisClient.Close(); err != nil {
				log.Errorf("Failed to close Redis client: %v", err)
			} else {
				log.Info("Redis client closed.")
			}
		}

		log.Info("Crawler shutdown complete.")
	})
}

// logMetricsPeriodically logs metrics every 30 seconds
func (crawler *Crawler) logMetricsPeriodically() {
	log := logger.GetLogger()
	ticker := time.NewTicker(10 * time.Second)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stats := crawler.stats.GetStats()
				pagesPerSecond := float64(0)
				if stats["uptime_seconds"].(float64) > 0 {
					pagesPerSecond = float64(stats["pages_processed"].(int64)) / stats["uptime_seconds"].(float64)
				}

				log.WithFields(logrus.Fields{
					"pages_processed":  stats["pages_processed"],
					"pages_successful": stats["pages_successful"],
					"pages_failed":     stats["pages_failed"],
					"kafka_successful": stats["kafka_successful"],
					"kafka_failed":     stats["kafka_failed"],
					"redis_successful": stats["redis_successful"],
					"redis_failed":     stats["redis_failed"],
					"uptime_seconds":   fmt.Sprintf("%.2f", stats["uptime_seconds"].(float64)),
					"pages_per_second": fmt.Sprintf("%.2f", pagesPerSecond),
				}).Info("Crawler metrics")
			case <-shutdown:
				log.Info("Stopping periodic metrics logging")
				return
			}
		}
	}()
}
