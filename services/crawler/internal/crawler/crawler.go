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

// Crawler represents the main web crawler instance. It manages the lifecycle
// of crawling operations, including Redis queue interaction, Kafka publishing,
// and Colly collector integration.
type Crawler struct {
	// Core configuration and external dependencies
	redisClient *redis.Client        // Client for interacting with Redis for URL queue management.
	producer    sarama.AsyncProducer // Kafka producer for publishing crawled page data. NOW ASYNCHRONOUS.
	stats       *metrics.Metrics     // Metrics collector for tracking crawling statistics.

	// Concurrency control:
	inFlightPages int64 // Track pages currently being processed (consider using a semaphore or channel for more robust control if this becomes complex).

	// URL management and filtering:
	whitelist []string // List of URL patterns allowed for crawling.
	blacklist []string // List of URL patterns disallowed for crawling.
	visited   sync.Map // A concurrent map to keep track of URLs that have been visited or are currently in flight
	requeued  sync.Map // A concurrent map to keep track of URLs that have been re-queued due to transient errors.

	// Channels for managing Kafka async producer feedback
	// These are not directly in the struct as they are managed by the producer itself
	// but are conceptually part of its async operation.
}

// Global variables for managing the crawler's state and lifecycle.
// These are initialized once and used across different goroutines for coordination.
var (
	crawler      *Crawler           // Singleton instance of the Crawler.
	wg           sync.WaitGroup     // WaitGroup to track active goroutines for graceful shutdown.
	ctx          context.Context    // Base context for the crawler, used for cancellation.
	cancel       context.CancelFunc // Function to trigger context cancellation.
	shutdown     chan struct{}      // Channel to signal a global shutdown to long-running goroutines.
	shutdownOnce sync.Once          // Ensures the shutdown process is initiated only once.
)

// NewCrawler creates and initializes a new Crawler instance.
// It sets up the application context, configures Redis and Kafka clients,
// and initializes the URL validator based on application configuration.
//
// Returns an error if any initialization step fails.
func New() error {
	log := logger.GetLogger() // Obtain the global logger instance.

	// Create a new context with cancellation for graceful shutdown.
	// This context will be passed to long-running operations.
	newCtx, newCancel := context.WithCancel(context.Background())
	newShutdown := make(chan struct{})

	ctx = newCtx
	cancel = newCancel
	shutdown = newShutdown

	// Initialize the Crawler instance.
	newCrawler := &Crawler{
		stats: metrics.NewMetrics(), // Create a new metrics collector.
	}

	// Parse URL whitelist and blacklist from configuration.
	// These are used by the URL validator to filter URLs.
	if cfgURLWhitelist := config.GetConfig().URLWhitelist; cfgURLWhitelist != "" {
		newCrawler.whitelist = strings.Split(cfgURLWhitelist, ",")
		log.Debugf("URL Whitelist configured: %s", cfgURLWhitelist)
	}
	if cfgURLBlacklist := config.GetConfig().URLBlacklist; cfgURLBlacklist != "" {
		newCrawler.blacklist = strings.Split(cfgURLBlacklist, ",")
		log.Debugf("URL Blacklist configured: %s", cfgURLBlacklist)
	}

	// Initialize Redis client. This is crucial for managing the pending URL queue.
	if err := newCrawler.initializeRedis(); err != nil {
		cancel() // Trigger context cancellation on failure.
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}
	log.Info("Redis client initialized successfully.")

	// Initialize Kafka producer. This is used to send crawled page data to Kafka.
	// This will now initialize an AsyncProducer and start its handlers.
	if err := newCrawler.initializeKafka(); err != nil {
		cancel() // Trigger context cancellation on failure.
		return fmt.Errorf("failed to initialize Kafka producer: %w", err)
	}
	log.Info("Kafka AsyncProducer initialized successfully.")

	// Initialize and configure the URL Validator.
	// This component is responsible for checking the validity and safety of URLs.
	validator.NewURLValidator(newCrawler.whitelist, newCrawler.blacklist, log)
	urlValidator := validator.GetURLValidator()

	// Configure URL Validator settings. These settings impact performance and security.
	urlValidator.SetDNSCacheTimeout(10 * time.Minute)
	// SetSkipDNSCheck to 'false' means DNS resolution will be performed for each URL.
	// Setting it to 'true' would skip DNS checks, which is faster but less safe
	// as it could allow connections to unresolvable or malicious IPs.
	urlValidator.SetSkipDNSCheck(false)
	urlValidator.SetAllowPrivateIPs(false) // Disallow crawling of private IP addresses.
	urlValidator.SetAllowLoopback(false)   // Disallow crawling of loopback addresses (e.g., 127.0.0.1).

	crawler = newCrawler // Assign the newly created crawler to the global instance.
	return nil
}

// GetCrawler returns the singleton instance of the Crawler.
// It's expected that New() has been called prior to this.
func GetCrawler() *Crawler {
	return crawler
}

// Start initiates the web crawling process.
// It sets up the Colly collector, seeds the Redis queue with start URLs,
// and orchestrates the goroutines responsible for feeding URLs to Colly
// and monitoring the crawling progress.
//
// This function blocks until all crawling activities are complete or a
// shutdown signal is received.
func (crawler *Crawler) Start() error {
	log := logger.GetLogger()
	cfg := config.GetConfig() // Retrieve application configuration.
	urlValidator := validator.GetURLValidator()

	log.Info("Starting web crawler with persistent Redis queue. Waiting indefinitely until completion.")

	// Log the full crawler configuration for debugging and operational visibility.
	log.WithFields(logrus.Fields{
		"kafka_config": logrus.Fields{
			"brokers":       cfg.KafkaBrokers,
			"topic":         cfg.KafkaTopic,
			"retry_max":     cfg.KafkaRetryMax,
			"producer_type": "Async", // Explicitly state producer type
		},
		"redis_config": logrus.Fields{
			"host":      cfg.RedisHost,
			"port":      cfg.RedisPort,
			"db":        cfg.RedisDB,
			"timeout":   cfg.RedisTimeout,
			"retry_max": cfg.RedisRetryMax,
		},
		"crawling_behavior": logrus.Fields{
			"start_urls":    cfg.StartURLs,
			"max_pages":     cfg.MaxPages,
			"crawl_depth":   cfg.CrawlDepth,
			"url_whitelist": cfg.URLWhitelist,
			"url_blacklist": cfg.URLBlacklist,
		},
		"performance_and_limits": logrus.Fields{
			"max_concurrency":  cfg.MaxConcurrency,
			"request_timeout":  cfg.RequestTimeout,
			"request_delay":    cfg.RequestDelay,
			"max_content_size": cfg.MaxContentSize,
		},
		"application_settings": logrus.Fields{
			"log_level":         cfg.LogLevel,
			"user_agent":        cfg.UserAgent,
			"enable_debug":      cfg.EnableDebug,
			"health_check_port": cfg.HealthCheckPort,
		},
	}).Info("Crawler configuration loaded and applied")

	// Start the health check server in a new goroutine.
	// This server provides insights into the crawler's operational status.
	health.Start(&wg, shutdown, crawler.redisClient, crawler.producer, crawler.stats)

	// Start periodic metrics logging in a new goroutine.
	// This provides continuous visibility into the crawler's performance.
	crawler.logMetricsPeriodically()

	// Setup the Colly collector instance. Colly is the core crawling engine.
	collector := crawler.setupCollyCollector(ctx)

	// Seed Redis 'pending_urls' with configured start URLs.
	// These URLs will be the initial points for the crawl.
	startURLs := strings.Split(cfg.StartURLs, ",")
	if len(startURLs) == 0 || (len(startURLs) == 1 && strings.TrimSpace(startURLs[0]) == "") {
		log.Warn("No start URLs provided in configuration. Crawler will not initiate any crawls.")
	} else {
		log.Infof("Attempting to seed Redis with %d start URLs.", len(startURLs))
		for _, rawURL := range startURLs {
			url := strings.TrimSpace(rawURL)
			if url == "" {
				continue // Skip empty strings resulting from split.
			}

			// Validate the start URL before adding to the queue.
			if !urlValidator.IsValidURL(url) {
				log.WithField("url", url).Warn("Skipping invalid start URL.")
				continue
			}

			// Check if the URL has already been visited or is pending to avoid duplicates.
			visitedOrPending, err := crawler.isURLVisited(url)
			if err != nil {
				log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to check start URL status in Redis, skipping.")
				continue
			}
			if visitedOrPending {
				log.WithFields(logrus.Fields{"url": url}).Info("Start URL already visited or pending, skipping initial queueing.")
				continue
			}

			// Add the valid and unvisited URL to the Redis 'pending_urls' set.
			_, err = crawler.redisClient.SAdd(ctx, "crawler:pending_urls", url).Result()
			if err != nil {
				log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to add start URL to Redis pending queue.")
			} else {
				log.WithFields(logrus.Fields{"url": url}).Info("Added start URL to Redis pending queue.")
			}
		}
	}

	// --- Core Waiting Logic ---
	// This section manages the lifetime of the main crawling operations.

	// Increment WaitGroup for the feedCollyFromRedisQueue goroutine.
	// This ensures the main goroutine waits for it to complete.
	wg.Add(1)
	go crawler.feedCollyFromRedisQueue(collector)

	log.Info("Crawler started. Blocking until all crawling activities are complete...")

	// Use a separate goroutine to orchestrate the waiting for Colly and the feeder.
	// This allows the main `Start` function to simply block until all work is signaled as done.
	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan) // Ensure doneChan is closed when this goroutine exits.

		// First, wait for Colly to finish processing all URLs it has been given.
		// This will only complete once `feedCollyFromRedisQueue` stops feeding it URLs
		// (either due to context cancellation, max pages, or empty Redis queue).
		log.Info("Waiting for Colly collector to finish processing its internal queue...")
		collector.Wait()
		log.Info("Colly collector finished processing its internal queue.")

		// Then, wait for the feedCollyFromRedisQueue goroutine to exit.
		// It will exit when its internal conditions (context, max pages + empty queue) are met.
		log.Info("Waiting for Redis queue feeder goroutine to stop...")
		wg.Wait() // Wait for all goroutines added to `wg` (primarily feedCollyFromRedisQueue and Kafka handlers)
		log.Info("Redis queue feeder goroutine has stopped.")

	}()

	// Block the main `Start()` goroutine until all crawling activities are done.
	<-doneChan
	log.Info("All crawling activities (Colly queue processing and Redis feeder) have completed.")

	// --- End Core Waiting Logic ---

	// Log final statistics before exiting.
	finalStats := crawler.stats.GetStats()
	log.WithFields(logrus.Fields{
		"pages_processed":  finalStats["pages_processed"],
		"pages_successful": finalStats["pages_successful"],
		"pages_failed":     finalStats["pages_failed"],
		"duration_seconds": fmt.Sprintf("%.2f", finalStats["uptime_seconds"].(float64)),
	}).Info("Crawling process summary")

	return nil
}

// Shutdown gracefully shuts down the crawler by stopping background goroutines
// and closing external connections (Kafka producer, Redis client).
// This function is designed to be called only once.
func (crawler *Crawler) Shutdown() {
	log := logger.GetLogger()
	shutdownOnce.Do(func() { // Ensures this block runs only once.
		log.Info("Initiating crawler shutdown sequence...")

		// Signal background goroutines to stop via the 'shutdown' channel.
		close(shutdown)
		// Signal context cancellation to active operations.
		cancel()

		// Wait for all tracked goroutines (e.g., health check, metrics logger, Kafka handlers) to finish.
		// A timeout is implemented to prevent indefinite blocking during shutdown.
		doneWaitingForGoroutines := make(chan struct{})
		go func() {
			wg.Wait() // Wait for all goroutines that called wg.Add() to call wg.Done().
			close(doneWaitingForGoroutines)
		}()

		select {
		case <-doneWaitingForGoroutines:
			log.Info("All background goroutines have gracefully stopped.")
		case <-time.After(10 * time.Second): // A reasonable timeout for goroutines to clean up.
			log.Warn("Timeout waiting for background goroutines to finish. Some might still be running or blocked.")
		}

		// Close Kafka producer connection.
		if crawler.producer != nil {
			log.Info("Attempting to close Kafka producer (Async)... This will flush remaining messages.")
			// AsyncClose() will close the input channel and wait for all buffered messages
			// to be sent, then close successes and errors channels.
			// The handleKafkaSuccesses and handleKafkaErrors goroutines should exit
			// gracefully as their channels will be closed.
			if err := crawler.producer.Close(); err != nil { // Using Close() which blocks until all pending messages are sent.
				log.WithError(err).Error("Failed to close Kafka AsyncProducer.")
			} else {
				log.Info("Kafka AsyncProducer closed successfully.")
			}
		}

		// Close Redis client connection.
		if crawler.redisClient != nil {
			log.Info("Attempting to close Redis client...")
			if err := crawler.redisClient.Close(); err != nil {
				log.WithError(err).Error("Failed to close Redis client.")
			} else {
				log.Info("Redis client closed successfully.")
			}
		}

		log.Info("Crawler shutdown complete.")
	})
}

// logMetricsPeriodically starts a goroutine that logs crawler metrics
// to the configured logger every 10 seconds. It stops when the 'shutdown'
// channel is closed.
func (crawler *Crawler) logMetricsPeriodically() {
	log := logger.GetLogger()
	ticker := time.NewTicker(10 * time.Second) // Configure metrics logging interval.
	wg.Add(1)                                  // Increment WaitGroup as this is a long-running goroutine.

	go func() {
		defer wg.Done()     // Ensure WaitGroup is decremented when this goroutine exits.
		defer ticker.Stop() // Release ticker resources when done.

		log.Info("Starting periodic metrics logging.")
		for {
			select {
			case <-ticker.C: // On each tick, retrieve and log current metrics.
				stats := crawler.stats.GetStats()
				pagesPerSecond := float64(0)
				if uptime := stats["uptime_seconds"].(float64); uptime > 0 {
					pagesPerSecond = float64(stats["pages_processed"].(int64)) / uptime
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
				}).Info("Current crawler metrics")
			case <-shutdown: // Exit loop if shutdown signal is received.
				log.Info("Stopping periodic metrics logging goroutine.")
				return
			}
		}
	}()
}
