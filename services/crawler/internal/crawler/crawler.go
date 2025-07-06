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
	"github.com/sneakyhydra/sneakdex/crawler/internal/metrics"
	"github.com/sneakyhydra/sneakdex/crawler/internal/validator"
)

// Crawler represents the main web crawler instance. It manages the lifecycle
// of crawling operations, including Redis queue interaction, Kafka publishing,
// and Colly collector integration.
type Crawler struct {
	Cfg *config.Config // Application configuration, loaded from config package.
	Log *logrus.Logger // Logger instance for structured logging throughout the crawler's operations.

	RedisClient   *redis.Client           // Client for interacting with Redis for URL queue management.
	AsyncProducer sarama.AsyncProducer    // Kafka async producer for publishing crawled page data.
	Stats         *metrics.Metrics        // Metrics collector for tracking crawling statistics.
	UrlValidator *validator.URLValidator // URL validator for checking and normalizing URLs.

	Whitelist     []string // List of URL patterns allowed for crawling.
	Blacklist     []string // List of URL patterns disallowed for crawling.
	Visited       sync.Map // A concurrent map to keep track of URLs that have been visited or are currently in flight
	Requeued      sync.Map // A concurrent map to keep track of URLs that have been re-queued due to transient errors.
	Pending       sync.Map // Local cache for pending URLs to avoid Redis checks
	SeenLocal     sync.Map // Local cache for any URL we've seen (visited, pending, or rejected)
	InFlightPages int64    // Track pages currently being processed (consider using a semaphore or channel for more robust control if this becomes complex).

	Ctx           context.Context
	CtxCancel     context.CancelFunc // Function to cancel the context, used for graceful shutdown.
	Wg            sync.WaitGroup     // WaitGroup to track active goroutines for graceful shutdown.
	CShutdown     chan struct{}      // Channel to signal shutdown to long-running goroutines.
	CShutdownOnce sync.Once          // Ensures shutdown is only initiated once.
}

// NewCrawler creates and initializes a new Crawler instance.
// It sets up the application context, configures Redis and Kafka clients,
// and initializes the URL validator based on application configuration.
// Returns an error if any initialization step fails.
func New(cfg *config.Config, log *logrus.Logger) (*Crawler, error) {
	// Create a new context with cancellation for graceful shutdown.
	// This context will be passed to long-running operations.
	ctx, ctxCancel := context.WithCancel(context.Background())
	shutdown := make(chan struct{})

	// Initialize the Crawler instance.
	crawler := &Crawler{
		Cfg:       cfg,
		Log:       log,
		Stats:     metrics.NewMetrics(), // Create a new metrics collector.
		Ctx:       ctx,
		CtxCancel: ctxCancel,
		CShutdown: shutdown,
	}

	// Parse URL whitelist and blacklist from configuration.
	// These are used by the URL validator to filter URLs.
	if cfgURLWhitelist := cfg.URLWhitelist; cfgURLWhitelist != "" {
		crawler.Whitelist = strings.Split(cfgURLWhitelist, ",")
		log.Debugf("URL Whitelist configured: %s", cfgURLWhitelist)
	}
	if cfgURLBlacklist := cfg.URLBlacklist; cfgURLBlacklist != "" {
		crawler.Blacklist = strings.Split(cfgURLBlacklist, ",")
		log.Debugf("URL Blacklist configured: %s", cfgURLBlacklist)
	}

	// Initialize Redis client.
	if err := crawler.initializeRedis(); err != nil {
		ctxCancel() // Trigger context cancellation on failure.
		return nil, fmt.Errorf("failed to initialize Redis: %w", err)
	}
	log.Info("Redis client initialized successfully.")

	// Initialize Kafka producer. This is used to send crawled page data to Kafka.
	// This will now initialize an AsyncProducer and start its handlers.
	if err := crawler.initializeKafka(); err != nil {
		ctxCancel() // Trigger context cancellation on failure.
		return nil, fmt.Errorf("failed to initialize Kafka producer: %w", err)
	}
	log.Info("Kafka AsyncProducer initialized successfully.")

	// Initialize and configure the URL Validator.
	// This component is responsible for checking the validity and safety of URLs.
	crawler.UrlValidator = validator.NewURLValidator(crawler.Whitelist, crawler.Blacklist, log)

	// Configure URL Validator settings. These settings impact performance and security.
	crawler.UrlValidator.SetDNSCacheTimeout(10 * time.Minute)
	// SetSkipDNSCheck to 'false' means DNS resolution will be performed for each URL.
	// Setting it to 'true' would skip DNS checks, which is faster but less safe
	// as it could allow connections to unresolvable or malicious IPs.
	crawler.UrlValidator.SetSkipDNSCheck(true)
	crawler.UrlValidator.SetAllowPrivateIPs(false) // Disallow crawling of private IP addresses.
	crawler.UrlValidator.SetAllowLoopback(false)   // Disallow crawling of loopback addresses (e.g., 127.0.0.1).

	return crawler, nil
}

// Start initiates the web crawling process.
// It sets up the Colly collector, seeds the Redis queue with start URLs,
// and orchestrates the goroutines responsible for feeding URLs to Colly
// and monitoring the crawling progress.
// This function blocks until all crawling activities are complete or a
// shutdown signal is received.
func (c *Crawler) Start() error {
	c.Log.Info("Starting web crawler with persistent Redis queue. Waiting indefinitely until completion.")

	// Log the full crawler configuration for debugging and operational visibility.
	c.Log.WithFields(logrus.Fields{
		"kafka_config": logrus.Fields{
			"brokers":       c.Cfg.KafkaBrokers,
			"topic":         c.Cfg.KafkaTopic,
			"retry_max":     c.Cfg.KafkaRetryMax,
			"producer_type": "Async", // Explicitly state producer type
		},
		"redis_config": logrus.Fields{
			"host":      c.Cfg.RedisHost,
			"port":      c.Cfg.RedisPort,
			"db":        c.Cfg.RedisDB,
			"timeout":   c.Cfg.RedisTimeout,
			"retry_max": c.Cfg.RedisRetryMax,
		},
		"crawling_behavior": logrus.Fields{
			"start_urls":    c.Cfg.StartURLs,
			"max_pages":     c.Cfg.MaxPages,
			"crawl_depth":   c.Cfg.CrawlDepth,
			"url_whitelist": c.Cfg.URLWhitelist,
			"url_blacklist": c.Cfg.URLBlacklist,
		},
		"performance_and_limits": logrus.Fields{
			"max_concurrency":  c.Cfg.MaxConcurrency,
			"request_timeout":  c.Cfg.RequestTimeout,
			"request_delay":    c.Cfg.RequestDelay,
			"max_content_size": c.Cfg.MaxContentSize,
		},
		"application_settings": logrus.Fields{
			"log_level":    c.Cfg.LogLevel,
			"user_agent":   c.Cfg.UserAgent,
			"enable_debug": c.Cfg.EnableDebug,
			"monitor_port": c.Cfg.MonitorPort,
		},
	}).Info("Crawler configuration loaded and applied")

	// Start periodic metrics logging in a new goroutine.
	// This provides continuous visibility into the crawler's performance.
	c.logMetricsPeriodically()

	// Setup the Colly collector instance. Colly is the core crawling engine.
	collector := c.setupCollyCollector()

	// Seed Redis 'pending_urls' with configured start URLs.
	// These URLs will be the initial points for the crawl.
	startURLs := strings.Split(c.Cfg.StartURLs, ",")
	if len(startURLs) == 0 || (len(startURLs) == 1 && strings.TrimSpace(startURLs[0]) == "") {
		c.Log.Warn("No start URLs provided in configuration. Crawler will not initiate any crawls.")
	} else {
		c.Log.Infof("Attempting to seed Redis with %d start URLs.", len(startURLs))
		
		// Pre-populate local cache by loading existing Redis data
		c.preloadLocalCaches()
		
		for _, rawURL := range startURLs {
			url := strings.TrimSpace(rawURL)
			if url == "" {
				continue // Skip empty strings resulting from split.
			}

			// Validate the start URL before adding to the queue.
			normalizedURL, valid := c.UrlValidator.IsValidURL(url)
			if !valid {
				c.Log.WithField("url", normalizedURL).Warn("Skipping invalid start URL.")
				continue
			}

			c.AddToPending(normalizedURL)
		}
	}

	// --- Core Waiting Logic ---
	// This section manages the lifetime of the main crawling operations.

	// Increment WaitGroup for the feedCollyFromRedisQueue goroutine.
	// This ensures the main goroutine waits for it to complete.
	c.Wg.Add(1)
	go c.feedCollyFromRedisQueue(collector)

	c.Log.Info("Crawler started. Blocking until all crawling activities are complete...")

	// Use a separate goroutine to orchestrate the waiting for Colly and the feeder.
	// This allows the main `Start` function to simply block until all work is signaled as done.
	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan) // Ensure doneChan is closed when this goroutine exits.

		// First, wait for Colly to finish processing all URLs it has been given.
		// This will only complete once `feedCollyFromRedisQueue` stops feeding it URLs
		// (either due to context cancellation, max pages, or empty Redis queue).
		c.Log.Info("Waiting for Colly collector to finish processing its internal queue...")
		collector.Wait()
		c.Log.Info("Colly collector finished processing its internal queue.")

		// Then, wait for the feedCollyFromRedisQueue goroutine to exit.
		// It will exit when its internal conditions (context, max pages + empty queue) are met.
		c.Log.Info("Waiting for Redis queue feeder goroutine to stop...")
		c.Wg.Wait() // Wait for all goroutines added to `wg` (primarily feedCollyFromRedisQueue)
		c.Log.Info("Redis queue feeder goroutine has stopped.")

	}()

	// Block the main `Start()` goroutine until all crawling activities are done.
	<-doneChan
	c.Log.Info("All crawling activities (Colly queue processing and Redis feeder) have completed.")

	// --- End Core Waiting Logic ---

	// Log final statistics before exiting.
	finalStats := c.Stats.GetStats()
	c.Log.WithFields(logrus.Fields{
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
func (c *Crawler) Shutdown() {
	c.CShutdownOnce.Do(func() { // Ensures this block runs only once.
		c.Log.Info("Initiating crawler shutdown sequence...")

		// Signal background goroutines to stop via the 'shutdown' channel.
		close(c.CShutdown)
		// Signal context cancellation to active operations.
		c.CtxCancel()

		// Wait for all tracked goroutines (e.g., monitor, metrics logger) to finish.
		// A timeout is implemented to prevent indefinite blocking during shutdown.
		doneWaitingForGoroutines := make(chan struct{})
		go func() {
			c.Wg.Wait() // Wait for all goroutines that called wg.Add() to call wg.Done().
			close(doneWaitingForGoroutines)
		}()

		select {
		case <-doneWaitingForGoroutines:
			c.Log.Info("All background goroutines have gracefully stopped.")
		case <-time.After(10 * time.Second): // A reasonable timeout for goroutines to clean up.
			c.Log.Warn("Timeout waiting for background goroutines to finish. Some might still be running or blocked.")
		}

		// Close Kafka producer connection.
		if c.AsyncProducer != nil {
			c.Log.Info("Attempting to close Kafka AsyncProducer.")
			if err := c.AsyncProducer.Close(); err != nil {
				c.Log.WithError(err).Error("Failed to close Kafka AsyncProducer.")
			} else {
				c.Log.Info("Kafka AsyncProducer closed successfully.")
			}
		}

		// Close Redis client connection.
		if c.RedisClient != nil {
			c.Log.Info("Attempting to close Redis client...")
			if err := c.RedisClient.Close(); err != nil {
				c.Log.WithError(err).Error("Failed to close Redis client.")
			} else {
				c.Log.Info("Redis client closed successfully.")
			}
		}

		c.Log.Info("Crawler shutdown complete.")
	})
}

// logMetricsPeriodically starts a goroutine that logs crawler metrics
// to the configured logger every 10 seconds. It stops when the 'shutdown'
// channel is closed.
func (c *Crawler) logMetricsPeriodically() {
	ticker := time.NewTicker(10 * time.Second) // Configure metrics logging interval.
	c.Wg.Add(1)                                // Increment WaitGroup as this is a long-running goroutine.

	go func() {
		defer c.Wg.Done()   // Ensure WaitGroup is decremented when this goroutine exits.
		defer ticker.Stop() // Release ticker resources when done.

		c.Log.Info("Starting periodic metrics logging.")
		for {
			select {
			case <-ticker.C: // On each tick, retrieve and log current metrics.
				stats := c.Stats.GetStats()
				pagesPerSecond := float64(0)
				if uptime := stats["uptime_seconds"].(float64); uptime > 0 {
					pagesPerSecond = float64(stats["pages_processed"].(int64)) / uptime
				}

				c.Log.WithFields(logrus.Fields{
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
			case <-c.CShutdown: // Exit loop if shutdown signal is received.
				c.Log.Info("Stopping periodic metrics logging goroutine.")
				return
			}
		}
	}()
}
