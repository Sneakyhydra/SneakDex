package main

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/kelseyhightower/envconfig"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// ------------------------------------------------------------------
// Crawler Configuration and Metrics
// ------------------------------------------------------------------

// Config holds the crawler configuration from environment variables
type Config struct {
	// Kafka Configuration
	KafkaBrokers  string `envconfig:"KAFKA_BROKERS" default:"kafka:9092"`
	KafkaTopic    string `envconfig:"KAFKA_TOPIC_HTML" default:"raw-html"`
	KafkaRetryMax int    `envconfig:"KAFKA_RETRY_MAX" default:"3"`

	// Redis Configuration
	RedisHost     string        `envconfig:"REDIS_HOST" default:"redis"`
	RedisPort     int           `envconfig:"REDIS_PORT" default:"6379"`
	RedisPassword string        `envconfig:"REDIS_PASSWORD" default:""`
	RedisDB       int           `envconfig:"REDIS_DB" default:"0"`
	RedisTimeout  time.Duration `envconfig:"REDIS_TIMEOUT" default:"60s"`
	RedisRetryMax int           `envconfig:"REDIS_RETRY_MAX" default:"3"`

	// Crawling Behavior
	StartURLs    string `envconfig:"START_URLS" default:"https://en.wikipedia.org/wiki/Special:Random,https://simple.wikipedia.org/wiki/Special:Random,https://news.ycombinator.com,https://www.reuters.com/news/archive/worldNews,https://www.bbc.com/news,https://github.com/trending,https://stackoverflow.com/questions,https://dev.to,https://developer.mozilla.org/en-US/docs/Web,https://arxiv.org/list/cs/new,https://eng.uber.com,https://netflixtechblog.com,https://blog.cloudflare.com"`
	CrawlDepth   int    `envconfig:"CRAWL_DEPTH" default:"3"`
	MaxPages     int64  `envconfig:"MAX_PAGES" default:"10000"`
	URLWhitelist string `envconfig:"URL_WHITELIST" default:""`
	URLBlacklist string `envconfig:"URL_BLACKLIST" default:""`

	// Performance & Limits
	MaxConcurrency int           `envconfig:"MAX_CONCURRENCY" default:"50"`
	RequestTimeout time.Duration `envconfig:"REQUEST_TIMEOUT" default:"30s"`
	RequestDelay   time.Duration `envconfig:"REQUEST_DELAY" default:"100ms"`
	MaxContentSize int64         `envconfig:"MAX_CONTENT_SIZE" default:"2621440"` // 2.5MB

	// Application Settings
	LogLevel        string `envconfig:"LOG_LEVEL" default:"info"`
	UserAgent       string `envconfig:"USER_AGENT" default:"SneakdexCrawler/1.0"`
	EnableDebug     bool   `envconfig:"ENABLE_DEBUG" default:"false"`
	HealthCheckPort int    `envconfig:"HEALTH_CHECK_PORT" default:"8080"`
}

// Validate checks the configuration for required fields and valid values
func (config *Config) Validate() error {
	// Kafka Configuration
	if config.KafkaBrokers == "" {
		return fmt.Errorf("kafka_brokers must be set")
	}
	if config.KafkaTopic == "" {
		return fmt.Errorf("kafka_topic must be set")
	}
	if config.KafkaRetryMax <= 0 {
		return fmt.Errorf("kafka_retry_max must be positive")
	}

	// Redis Configuration
	if config.RedisHost == "" {
		return fmt.Errorf("redis_host must be set")
	}
	if config.RedisPort <= 0 || config.RedisPort > 65535 {
		return fmt.Errorf("redis_port must be between 1 and 65535")
	}
	if config.RedisDB < 0 {
		return fmt.Errorf("redis_db must be non-negative")
	}
	if config.RedisTimeout <= 0 {
		return fmt.Errorf("redis_timeout must be positive")
	}
	if config.RedisRetryMax <= 0 {
		return fmt.Errorf("redis_retry_max must be positive")
	}

	// Crawling Behavior
	if config.StartURLs == "" {
		return fmt.Errorf("start_urls must be set")
	}
	if config.CrawlDepth < 1 {
		return fmt.Errorf("crawl_depth must be at least 1")
	}
	if config.MaxPages <= 0 {
		return fmt.Errorf("max_pages must be positive")
	}
	if config.MaxPages > 1000000 {
		return fmt.Errorf("max_pages must not exceed 1,000,000")
	}

	// Performance & Limits
	if config.MaxConcurrency < 1 || config.MaxConcurrency > 1000 {
		return fmt.Errorf("max_concurrency must be between 1 and 1000")
	}
	if config.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be positive")
	}
	if config.RequestDelay < 0 {
		return fmt.Errorf("request_delay must be non-negative")
	}
	if config.MaxContentSize <= 0 {
		return fmt.Errorf("max_content_size must be positive")
	}

	// Application Settings
	if config.LogLevel == "" {
		return fmt.Errorf("log_level must be set")
	}
	if config.UserAgent == "" {
		return fmt.Errorf("user_agent must be set")
	}
	if config.HealthCheckPort <= 0 || config.HealthCheckPort > 65535 {
		return fmt.Errorf("health_check_port must be between 1 and 65535")
	}

	return nil
}

// CrawlError represents a structured error type for crawling operations
type CrawlError struct {
	URL       string
	Operation string
	Err       error
	Retry     bool
	Timestamp time.Time
}

// Error implements the error interface for CrawlError
func (err *CrawlError) Error() string {
	return fmt.Sprintf("CrawlError: %s operation failed for URL %s at %s: %v (Retry: %v)",
		err.Operation, err.URL, err.Timestamp.Format(time.RFC3339), err.Err, err.Retry)
}

// Metrics holds crawler metrics
type Metrics struct {
	PagesProcessed  int64
	PagesSuccessful int64
	PagesFailed     int64
	KafkaSuccessful int64
	KafkaFailed     int64
	RedisSuccessful int64
	RedisFailed     int64
	StartTime       time.Time
}

func (metric *Metrics) IncrementPagesProcessed()  { atomic.AddInt64(&metric.PagesProcessed, 1) }
func (metric *Metrics) IncrementPagesSuccessful() { atomic.AddInt64(&metric.PagesSuccessful, 1) }
func (metric *Metrics) IncrementPagesFailed()     { atomic.AddInt64(&metric.PagesFailed, 1) }
func (metric *Metrics) IncrementKafkaSuccessful() { atomic.AddInt64(&metric.KafkaSuccessful, 1) }
func (metric *Metrics) IncrementKafkaFailed()     { atomic.AddInt64(&metric.KafkaFailed, 1) }
func (metric *Metrics) IncrementRedisSuccessful() { atomic.AddInt64(&metric.RedisSuccessful, 1) }
func (metric *Metrics) IncrementRedisFailed()     { atomic.AddInt64(&metric.RedisFailed, 1) }

// Uptime returns the duration since the crawler started.
func (metric *Metrics) Uptime() time.Duration {
	return time.Since(metric.StartTime)
}

// GetStats returns a map of crawler statistics
func (metric *Metrics) GetStats() map[string]any {
	return map[string]any{
		"pages_processed":  atomic.LoadInt64(&metric.PagesProcessed),
		"pages_successful": atomic.LoadInt64(&metric.PagesSuccessful),
		"pages_failed":     atomic.LoadInt64(&metric.PagesFailed),
		"kafka_successful": atomic.LoadInt64(&metric.KafkaSuccessful),
		"kafka_failed":     atomic.LoadInt64(&metric.KafkaFailed),
		"redis_successful": atomic.LoadInt64(&metric.RedisSuccessful),
		"redis_failed":     atomic.LoadInt64(&metric.RedisFailed),
		"uptime_seconds":   metric.Uptime().Seconds(),
	}
}

// ------------------------------------------------------------------
// Crawler initialization and setup
// ------------------------------------------------------------------

// Crawler represents the main crawler instance
type Crawler struct {
	config       Config
	log          *logrus.Logger
	redisClient  *redis.Client
	producer     sarama.SyncProducer
	metrics      *Metrics
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	whitelist    []string
	blacklist    []string
	shutdown     chan struct{}
	shutdownOnce sync.Once
	visited      sync.Map
}

// initializeRedis sets up Redis client with proper configuration
func (crawler *Crawler) initializeRedis() error {
	// Construct the Redis address from host and port
	redisAddr := fmt.Sprintf("%s:%d", crawler.config.RedisHost, crawler.config.RedisPort)

	// Create a new Redis client with the provided configuration
	crawler.redisClient = redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     crawler.config.RedisPassword,
		DB:           crawler.config.RedisDB,
		DialTimeout:  crawler.config.RedisTimeout,
		ReadTimeout:  crawler.config.RedisTimeout,
		WriteTimeout: crawler.config.RedisTimeout,
		MaxRetries:   crawler.config.RedisRetryMax,
	})

	// Test connection with retries
	for attempt := 1; attempt <= crawler.config.RedisRetryMax; attempt++ {
		ctx, cancel := context.WithTimeout(crawler.ctx, crawler.config.RedisTimeout)
		err := crawler.redisClient.Ping(ctx).Err()
		cancel()

		if err == nil {
			crawler.log.Info("Redis connection established")
			return nil
		}

		crawler.log.Warnf("Redis connection attempt %d/%d failed: %v", attempt, crawler.config.RedisRetryMax, err)
		if attempt < crawler.config.RedisRetryMax {
			// Exponential backoff for retries
			backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed to connect to redis after %d attempts. please ensure redis is running on %s", crawler.config.RedisRetryMax, redisAddr)
}

// initializeKafka sets up Kafka producer with proper configuration
func (crawler *Crawler) initializeKafka() error {
	//  Create a new Sarama configuration
	kafkaConfig := sarama.NewConfig()

	// Set Kafka producer configurations
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	kafkaConfig.Producer.Retry.Max = crawler.config.KafkaRetryMax
	kafkaConfig.Producer.Retry.Backoff = 100 * time.Millisecond
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Return.Errors = true
	kafkaConfig.Producer.Timeout = crawler.config.RequestTimeout
	kafkaConfig.Net.DialTimeout = crawler.config.RequestTimeout
	kafkaConfig.Metadata.RefreshFrequency = 10 * time.Minute

	// Create a new Kafka producer
	brokers := strings.Split(crawler.config.KafkaBrokers, ",")

	for attempt := 1; attempt <= crawler.config.KafkaRetryMax; attempt++ {
		producer, err := sarama.NewSyncProducer(brokers, kafkaConfig)
		if err == nil {
			crawler.producer = producer
			crawler.log.Info("Kafka producer initialized")
			return nil
		}
		crawler.log.Warnf("Kafka producer initialization attempt %d/%d failed: %v", attempt, crawler.config.KafkaRetryMax, err)
		if attempt < crawler.config.KafkaRetryMax {
			// Exponential backoff for retries
			backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed to create kafka producer after %d attempts. please ensure kafka is running on %s", crawler.config.KafkaRetryMax, strings.Join(brokers, ","))
}

// NewCrawler creates a new crawler instance
func NewCrawler() (*Crawler, error) {
	// Load and validate configuration from environment variables
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize logger
	log := logrus.New()
	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	log.SetLevel(level)
	log.SetFormatter(&logrus.JSONFormatter{})

	// Create a new context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Create the Crawler instance
	crawler := &Crawler{
		config:   config,
		log:      log,
		ctx:      ctx,
		cancel:   cancel,
		metrics:  &Metrics{StartTime: time.Now()},
		shutdown: make(chan struct{}),
	}

	// Parse whitelist and blacklist
	if config.URLWhitelist != "" {
		crawler.whitelist = strings.Split(config.URLWhitelist, ",")
	}
	if config.URLBlacklist != "" {
		crawler.blacklist = strings.Split(config.URLBlacklist, ",")
	}

	// Initialize Redis
	if err := crawler.initializeRedis(); err != nil {
		cancel()
		return nil, fmt.Errorf("redis initialization failed: %w", err)
	}
	crawler.log.Info("Redis initialized successfully")

	// Initialize Kafka
	if err := crawler.initializeKafka(); err != nil {
		cancel()
		return nil, fmt.Errorf("kafka initialization failed: %w", err)
	}
	crawler.log.Info("Kafka initialized successfully")

	return crawler, nil
}

// ------------------------------------------------------------------
// Crawler URL Validation and Redis Interaction
// ------------------------------------------------------------------

// isValidURL checks if a URL is valid and not blocked by blacklist or whitelist rules.
func (crawler *Crawler) isValidURL(rawURL string) bool {
	// Check if the URL is empty
	if rawURL == "" {
		return false
	}

	// Parse the URL to ensure it is well-formed
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Check if the URL has a valid scheme and host
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}
	if parsedURL.Host == "" {
		return false
	}

	// Normalize the host to lowercase for consistent checks
	host := strings.ToLower(parsedURL.Hostname())

	// Resolve DNS for domain names
	ips, err := net.LookupIP(host)
	if err == nil {
		for _, ip := range ips {
			// Check if IP is a loopback or private address
			// This prevents crawling internal networks or localhost.
			if ip.IsLoopback() || ip.IsPrivate() {
				crawler.log.WithFields(logrus.Fields{"url": rawURL, "ip": ip.String()}).Debug("Skipping URL with loopback or private IP")
				return false
			}
		}
	} else {
		// Log DNS resolution errors but don't necessarily block if it's not crucial
		crawler.log.WithFields(logrus.Fields{"url": rawURL, "error": err}).Debug("Failed to resolve IP for host")
	}

	// Check blacklist domains
	for _, blocked := range crawler.blacklist {
		blocked = strings.ToLower(blocked)
		// Check if the host ends with the blocked domain or is exactly the blocked domain
		if strings.HasSuffix(host, "."+blocked) || host == blocked {
			crawler.log.WithFields(logrus.Fields{"url": rawURL, "blocked_by": blocked}).Debug("Skipping URL due to blacklist")
			return false
		}
	}

	// Whitelist domains if set
	if len(crawler.whitelist) > 0 {
		allowed := false
		for _, allowedDomain := range crawler.whitelist {
			allowedDomain = strings.ToLower(allowedDomain)
			// Check if the host ends with the allowed domain or is exactly the allowed domain
			if strings.HasSuffix(host, "."+allowedDomain) || host == allowedDomain {
				allowed = true
				break
			}
		}
		// If no allowed domain matched, skip this URL
		if !allowed {
			crawler.log.WithFields(logrus.Fields{"url": rawURL}).Debug("Skipping URL not in whitelist")
			return false
		}
	}

	return true
}

// normalizeURL removes fragments and query parameters for deduplication purposes,
// and converts scheme/host to lowercase.
func normalizeURL(rawURL string) (string, error) {
	// Parse the URL to ensure it is well-formed
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Remove fragments and query parameters
	parsed.Fragment = "" // remove fragments
	parsed.RawQuery = "" // remove query parameters

	// Normalize scheme and host to lowercase
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)

	// Remove trailing slash except root path
	if parsed.Path != "/" {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}

	return parsed.String(), nil
}

// checkRedisSet checks if URL exists in the specified Redis set with retry logic
func (crawler *Crawler) checkRedisSet(setKey, url string) (bool, error) {
	var lastErr error

	for attempt := 1; attempt <= crawler.config.RedisRetryMax; attempt++ {
		ctx, cancel := context.WithTimeout(crawler.ctx, crawler.config.RedisTimeout)

		exists, err := crawler.redisClient.SIsMember(ctx, setKey, url).Result()
		cancel()

		if err == nil {
			if exists {
				crawler.metrics.IncrementRedisSuccessful()
			}
			return exists, nil
		}

		lastErr = err
		crawler.log.Warnf("Redis SIsMember '%s' attempt %d/%d failed for URL %s: %v",
			setKey, attempt, crawler.config.RedisRetryMax, url, err)

		// Don't sleep on the last attempt
		if attempt < crawler.config.RedisRetryMax {
			backoff := time.Duration(1<<uint(attempt)) * 100 * time.Millisecond // 2^attempt * 100ms
			time.Sleep(backoff)
		}
	}

	crawler.metrics.IncrementRedisFailed()
	return false, fmt.Errorf("failed after %d attempts: %w", crawler.config.RedisRetryMax, lastErr)
}

// cacheAndLogFound caches the URL locally and logs the discovery
func (crawler *Crawler) cacheAndLogFound(url, setType string) {
	crawler.visited.Store(url, struct{}{})
	crawler.metrics.IncrementRedisSuccessful()
	crawler.log.WithFields(logrus.Fields{
		"url":      url,
		"set_type": setType,
	}).Trace("URL found in redis")
}

// isURLVisited checks if a URL has already been visited or is pending,
// using both local memory cache and Redis sets for quick lookups.
func (crawler *Crawler) isURLVisited(url string) (bool, error) {
	// Check local memory cache first for quick lookups
	if _, ok := crawler.visited.Load(url); ok {
		crawler.log.WithFields(logrus.Fields{"url": url}).Trace("URL found in local visited/pending cache")
		return true, nil
	}

	// Check both Redis sets with retry logic
	visited, err := crawler.checkRedisSet("crawler:visited_urls", url)
	if err != nil {
		return false, fmt.Errorf("failed to check visited URLs in redis: %w", err)
	}
	if visited {
		crawler.cacheAndLogFound(url, "visited")
		return true, nil
	}

	pending, err := crawler.checkRedisSet("crawler:pending_urls", url)
	if err != nil {
		return false, fmt.Errorf("failed to check pending URLs in redis: %w", err)
	}
	if pending {
		crawler.cacheAndLogFound(url, "pending")
		return true, nil
	}

	crawler.log.WithFields(logrus.Fields{"url": url}).Trace("URL not found in redis, considered new")
	return false, nil
}

// ------------------------------------------------------------------
// Crawler Main Logic
// ------------------------------------------------------------------

// feedCollyFromRedisQueue feeds URLs from the Redis queue to the Colly collector.
func (c *Crawler) feedCollyFromRedisQueue(collector *colly.Collector) {
	defer c.wg.Done() // Decrement WaitGroup counter when this goroutine exits
	c.log.Info("Starting goroutine to feed URLs from Redis queue to Colly")

	ticker := time.NewTicker(200 * time.Millisecond) // Poll Redis every 200ms
	defer ticker.Stop()

	emptyQueueChecks := 0    // Counter for consecutive times the Redis queue was found empty
	const maxEmptyChecks = 5 // How many consecutive empty checks before considering the queue truly empty

	for {
		select {
		case <-c.ctx.Done(): // Primary shutdown signal: context cancellation
			c.log.Info("Stopping Redis queue feeder goroutine due to context cancellation.")
			return // Exit goroutine immediately
		case <-ticker.C:
			// Check if max pages limit is reached. If so, we're approaching termination.
			if atomic.LoadInt64(&c.metrics.PagesProcessed) >= c.config.MaxPages {
				c.log.Debug("Max pages limit reached. Feeder will stop if queue is empty.")
				// We don't return immediately, allow draining existing items if any.
			}

			// Attempt to pull a URL from the 'pending' set
			url, err := c.redisClient.SPop(c.ctx, "crawler:pending_urls").Result()
			if err == redis.Nil {
				// Redis queue is empty
				emptyQueueChecks++
				c.log.WithFields(logrus.Fields{"empty_checks": emptyQueueChecks}).Trace("Redis pending queue empty.")

				// Check for termination conditions if queue is consistently empty
				if emptyQueueChecks >= maxEmptyChecks {
					// If queue is consistently empty AND (max pages reached OR context is cancelled)
					if atomic.LoadInt64(&c.metrics.PagesProcessed) >= c.config.MaxPages || c.ctx.Err() != nil {
						c.log.Info("Redis pending queue consistently empty and termination condition met. Stopping feeder.")
						return // Exit goroutine
					}
				}
				continue // Queue empty, wait for next tick
			} else if err != nil {
				// An actual error occurred with Redis (not just empty queue)
				c.log.WithFields(logrus.Fields{"error": err}).Error("Failed to pop URL from Redis pending queue, retrying...")
				// Consider adding more robust retry logic for transient Redis errors if needed.
				// For now, it simply continues to the next ticker cycle.
				continue
			}

			// If a URL was successfully popped, reset the empty queue counter
			emptyQueueChecks = 0

			c.log.WithFields(logrus.Fields{"url": url}).Debug("Pulled URL from Redis pending queue for visit")

			// Feed the URL to Colly. Colly's internal Limit rules will manage concurrency.
			if err := collector.Visit(url); err != nil {
				// This error means Colly failed to even initiate the visit (e.g., malformed URL given to Colly).
				// It's crucial to mark this URL as visited (failed) in Redis,
				// otherwise it might be re-added to pending if discovered again, leading to an infinite loop.
				c.log.WithFields(logrus.Fields{"url": url, "error": err}).Warn("Failed to initiate Colly visit for URL from Redis queue (e.g., invalid format). Marking as failed visited.")
				_, redisErr := c.redisClient.SAdd(c.ctx, "crawler:visited_urls", url).Result()
				if redisErr != nil {
					c.log.WithFields(logrus.Fields{"url": url, "error": redisErr}).Error("Failed to add URL to visited_urls Redis set after Colly visit initiation failure")
				}
			}
		}
	}
}

// sendToKafka sends the scraped HTML content to Kafka.
func (c *Crawler) sendToKafka(url, html string) error {
	if int64(len(html)) > c.config.MaxContentSize {
		return &CrawlError{
			URL:       url,
			Operation: "kafka_send",
			Err:       fmt.Errorf("content size %d exceeds limit %d", len(html), c.config.MaxContentSize),
			Retry:     false, // No point in retrying for size limit
		}
	}

	msg := &sarama.ProducerMessage{
		Topic:     c.config.KafkaTopic,
		Key:       sarama.StringEncoder(url),
		Value:     sarama.StringEncoder(html),
		Timestamp: time.Now(),
	}

	for attempt := 0; attempt < c.config.KafkaRetryMax; attempt++ {
		_, _, err := c.producer.SendMessage(msg)
		if err == nil {
			c.metrics.IncrementKafkaSuccessful()
			return nil
		}

		c.log.Warnf("Kafka SendMessage attempt %d failed for URL %s: %v", attempt+1, url, err)
		if attempt < c.config.KafkaRetryMax-1 {
			// Exponential backoff for retries
			backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	c.metrics.IncrementKafkaFailed()
	return &CrawlError{
		URL:       url,
		Operation: "kafka_send",
		Err:       fmt.Errorf("failed to send to Kafka after %d attempts", c.config.KafkaRetryMax),
		Retry:     true, // Mark as retriable for external handling if needed
	}
}

// setupCollyCollector configures the colly collector with callbacks for crawling logic.
func (c *Crawler) setupCollyCollector() *colly.Collector {
	options := []colly.CollectorOption{
		colly.MaxDepth(c.config.CrawlDepth),
		colly.Async(true), // Enable asynchronous requests, managing concurrency internally
		colly.UserAgent(c.config.UserAgent),
	}

	if c.config.EnableDebug {
		options = append(options, colly.Debugger(&debug.LogDebugger{}))
	}

	collector := colly.NewCollector(options...)

	// Apply rate limits for all domains
	collector.Limit(&colly.LimitRule{
		DomainGlob:  "*", // Apply to all domains
		Parallelism: c.config.MaxConcurrency,
		Delay:       c.config.RequestDelay,
		RandomDelay: c.config.RequestDelay / 2, // Add some random delay
	})

	collector.SetRequestTimeout(c.config.RequestTimeout)

	// Request filtering and header setting
	collector.OnRequest(func(r *colly.Request) {
		// Check for context cancellation before proceeding with the request
		select {
		case <-c.ctx.Done():
			c.log.WithFields(logrus.Fields{"url": r.URL.String()}).Debug("Request aborted due to shutdown")
			r.Abort()
			return
		default:
			// Continue
		}

		urlStr := r.URL.String()
		// Skip common file extensions
		skipExts := []string{".pdf", ".jpg", ".jpeg", ".png", ".gif", ".css", ".js", ".ico", ".svg", ".woff", ".ttf", ".mp4", ".mp3", ".zip", ".exe"}
		for _, ext := range skipExts {
			if strings.HasSuffix(strings.ToLower(urlStr), ext) {
				c.log.WithFields(logrus.Fields{"url": urlStr, "ext": ext}).Debug("Skipping URL due to file extension")
				r.Abort()
				return
			}
		}

		// Set standard request headers
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Accept-Encoding", "gzip, deflate")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")

		c.log.WithFields(logrus.Fields{"url": urlStr}).Debug("Visiting URL")
	})

	// Handle HTML pages
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		// Check for context cancellation
		select {
		case <-c.ctx.Done():
			c.log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("HTML processing skipped due to shutdown")
			return
		default:
			// Continue
		}

		// Stop processing if max pages limit is reached
		if atomic.LoadInt64(&c.metrics.PagesProcessed) >= c.config.MaxPages {
			c.log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("Max pages limit reached, stopping HTML processing")
			return
		}

		c.metrics.IncrementPagesProcessed()
		url := e.Request.URL.String()

		html, err := e.DOM.Html()
		if err != nil {
			c.log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to extract HTML")
			c.metrics.IncrementPagesFailed()
			// IMPORTANT: Mark as visited even if HTML extraction fails
			_, redisErr := c.redisClient.SAdd(c.ctx, "crawler:visited_urls", url).Result()
			if redisErr != nil {
				c.log.WithFields(logrus.Fields{"url": url, "error": redisErr}).Error("Failed to add URL to visited_urls Redis set after HTML extraction failure")
			}
			return
		}

		if err := c.sendToKafka(url, html); err != nil {
			c.log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to send to Kafka")
			c.metrics.IncrementPagesFailed()
			// IMPORTANT: Mark as visited even if sending to Kafka fails
			_, redisErr := c.redisClient.SAdd(c.ctx, "crawler:visited_urls", url).Result()
			if redisErr != nil {
				c.log.WithFields(logrus.Fields{"url": url, "error": redisErr}).Error("Failed to add URL to visited_urls Redis set after Kafka failure")
			}
			return
		}

		c.metrics.IncrementPagesSuccessful()
		c.log.WithFields(logrus.Fields{"url": url, "content_size": len(html)}).Debug("Page processed successfully")

		// IMPORTANT: Mark as visited in Redis after successful processing
		_, err = c.redisClient.SAdd(c.ctx, "crawler:visited_urls", url).Result()
		if err != nil {
			c.log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to add URL to visited_urls Redis set")
		}
	})

	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		// Check for context cancellation
		select {
		case <-c.ctx.Done():
			c.log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("Link extraction skipped due to shutdown")
			return
		default:
			// Continue
		}

		// Stop processing if max pages limit is reached
		if atomic.LoadInt64(&c.metrics.PagesProcessed) >= c.config.MaxPages {
			c.log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("Max pages limit reached, stopping link extraction")
			return
		}

		link := e.Attr("href")
		if link == "" {
			return
		}

		// Skip common non-HTTP/HTTPS links or internal page anchors
		if strings.HasPrefix(link, "javascript:") ||
			strings.HasPrefix(link, "mailto:") ||
			strings.HasPrefix(link, "tel:") ||
			strings.Contains(link, "#") { // Links with fragments are typically internal page anchors
			return
		}

		absoluteURL := e.Request.AbsoluteURL(link)
		if !c.isValidURL(absoluteURL) {
			return
		}

		normalized, err := normalizeURL(absoluteURL)
		if err != nil {
			c.log.WithFields(logrus.Fields{"url": absoluteURL, "error": err}).Debug("Failed to normalize URL")
			return
		}

		// Use the enhanced isURLVisited to check both 'visited' and 'pending' sets
		visitedOrPending, err := c.isURLVisited(normalized)
		if err != nil {
			c.log.WithFields(logrus.Fields{"url": normalized, "error": err}).Error("Failed to check URL visited/pending status")
			c.metrics.IncrementRedisFailed() // Increment failed metrics if Redis check fails
			return
		}

		if !visitedOrPending {
			c.log.WithFields(logrus.Fields{"url": normalized}).Debug("Adding new URL to Redis pending queue")
			// Add to Redis 'pending_urls' set instead of calling Colly's Visit directly
			_, err := c.redisClient.SAdd(c.ctx, "crawler:pending_urls", normalized).Result()
			if err != nil {
				c.log.WithFields(logrus.Fields{"url": normalized, "error": err}).Error("Failed to add URL to Redis pending queue")
				c.metrics.IncrementRedisFailed()
			}
		}
	})

	// Error handler for HTTP requests
	collector.OnError(func(r *colly.Response, err error) {
		// Suppress logging common timeout/connection errors if debug is not enabled,
		// as these can be noisy but are expected in network operations.
		isNetworkError := strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "no such host")

		if !isNetworkError || c.config.EnableDebug {
			c.log.WithFields(logrus.Fields{
				"url":         r.Request.URL.String(),
				"status_code": r.StatusCode,
				"error":       err,
			}).Warn("Request failed")
		} else {
			c.log.WithFields(logrus.Fields{
				"url":         r.Request.URL.String(),
				"status_code": r.StatusCode,
				"error":       err,
			}).Debug("Suppressed network error") // Log as debug if suppressed
		}
		c.metrics.IncrementPagesFailed()

		// IMPORTANT: Mark as visited in Redis, even if the request failed, to prevent re-attempts for this URL.
		url := r.Request.URL.String()
		_, redisErr := c.redisClient.SAdd(c.ctx, "crawler:visited_urls", url).Result()
		if redisErr != nil {
			c.log.WithFields(logrus.Fields{"url": url, "error": redisErr}).Error("Failed to add URL to visited_urls Redis set on error")
		}
	})

	// Debug logging on successful response
	if c.config.EnableDebug {
		collector.OnResponse(func(r *colly.Response) {
			c.log.WithFields(logrus.Fields{
				"url":          r.Request.URL.String(),
				"status_code":  r.StatusCode,
				"content_type": r.Headers.Get("Content-Type"),
				"size":         len(r.Body),
			}).Debug("Response received")
		})
	}

	return collector
}

// startHealthCheck starts HTTP health check and metrics server.
func (c *Crawler) startHealthCheck() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Also check Redis and Kafka health here for a more robust health check
		redisStatus := "ok"
		kafkaStatus := "ok"

		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second) // Use request context, short timeout
		defer cancel()

		if err := c.redisClient.Ping(ctx).Err(); err != nil {
			redisStatus = fmt.Sprintf("error: %v", err)
		}

		// Kafka health check:
		// A non-invasive check is just to ensure the producer is initialized (not nil).
		// Sarama's SyncProducer handles internal connection management and retries.
		if c.producer == nil {
			kafkaStatus = "error: Kafka producer uninitialized"
		}
		// For more advanced Kafka health, you might consider:
		// 1. Trying to send a trivial message to a dedicated health-check topic.
		// 2. Accessing the underlying Sarama Client's metadata (if exposed or managed separately).
		// For this setup, a non-nil producer indicates it's attempting to work.

		status := "healthy"
		if strings.HasPrefix(redisStatus, "error") || strings.HasPrefix(kafkaStatus, "error") {
			status = "unhealthy"
		}

		fmt.Fprintf(w, `{"status":"%s","timestamp":"%s","dependencies":{"redis":"%s","kafka":"%s"}}`,
			status, time.Now().Format(time.RFC3339), redisStatus, kafkaStatus)
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats := c.metrics.GetStats()
		fmt.Fprintf(w, `{
            "pages_processed": %d,
            "pages_successful": %d,
            "pages_failed": %d,
            "kafka_successful": %d,
            "kafka_failed": %d,
            "redis_successful": %d,
            "redis_failed": %d,
            "uptime_seconds": %.2f
        }`,
			stats["pages_processed"],
			stats["pages_successful"],
			stats["pages_failed"],
			stats["kafka_successful"],
			stats["kafka_failed"],
			stats["redis_successful"],
			stats["redis_failed"],
			stats["uptime_seconds"].(float64), // Ensure type assertion for float
		)
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", c.config.HealthCheckPort),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.log.Infof("Health check server starting on port %d", c.config.HealthCheckPort)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			c.log.Errorf("Health check server error: %v", err)
		}
	}()

	// Graceful shutdown for health check server
	go func() {
		<-c.shutdown // Wait for the main shutdown signal
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			c.log.Errorf("Health check server shutdown error: %v", err)
		} else {
			c.log.Info("Health check server shut down gracefully")
		}
	}()
}

// logMetricsPeriodically logs metrics every 30 seconds
func (c *Crawler) logMetricsPeriodically() {
	ticker := time.NewTicker(30 * time.Second)
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stats := c.metrics.GetStats()
				pagesPerSecond := float64(0)
				if stats["uptime_seconds"].(float64) > 0 {
					pagesPerSecond = float64(stats["pages_processed"].(int64)) / stats["uptime_seconds"].(float64)
				}

				c.log.WithFields(logrus.Fields{
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
			case <-c.shutdown:
				c.log.Info("Stopping periodic metrics logging")
				return
			}
		}
	}()
}

func (c *Crawler) Start() error {
	c.log.Info("Starting web crawler with persistent Redis queue. Waiting indefinitely until completion.")
	c.log.WithFields(logrus.Fields{
		"kafka_brokers": c.config.KafkaBrokers,
		"start_urls":    c.config.StartURLs,
		"max_pages":     c.config.MaxPages,
		"max_depth":     c.config.CrawlDepth,
		"concurrency":   c.config.MaxConcurrency,
	}).Info("Crawler configuration")

	c.startHealthCheck()
	c.logMetricsPeriodically()

	collector := c.setupCollyCollector()

	// Seed Redis 'pending_urls' with configured start URLs
	startURLs := strings.Split(c.config.StartURLs, ",")
	for _, rawURL := range startURLs {
		url := strings.TrimSpace(rawURL)
		if !c.isValidURL(url) {
			c.log.Warnf("Skipping invalid start URL: %s", url)
			continue
		}

		visitedOrPending, err := c.isURLVisited(url)
		if err != nil {
			c.log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to check start URL status in Redis, skipping")
			continue
		}
		if visitedOrPending {
			c.log.WithFields(logrus.Fields{"url": url}).Info("Start URL already visited or pending, skipping initial queueing")
			continue
		}

		// Add to Redis 'pending_urls' set
		_, err = c.redisClient.SAdd(c.ctx, "crawler:pending_urls", url).Result()
		if err != nil {
			c.log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to add start URL to Redis pending queue")
		} else {
			c.log.WithFields(logrus.Fields{"url": url}).Info("Added start URL to Redis pending queue")
		}
	}

	// --- Core Waiting Logic ---

	// Increment WaitGroup for the feedCollyFromRedisQueue goroutine
	c.wg.Add(1)
	go c.feedCollyFromRedisQueue(collector)

	c.log.Info("Crawler started. Blocking until all crawling activities are complete...")

	// Use a separate goroutine to orchestrate the waiting for Colly and the feeder.
	// This allows the main `Start` function to simply block until all work is signaled as done.
	doneChan := make(chan struct{})
	go func() {
		// First, wait for Colly to finish processing all URLs it has been given.
		// This will only complete once `feedCollyFromRedisQueue` stops feeding it URLs
		// (either due to context cancellation, max pages, or empty Redis queue).
		collector.Wait()
		c.log.Info("Colly collector finished processing its internal queue.")

		// Then, wait for the feedCollyFromRedisQueue goroutine to exit.
		// It will exit when its internal conditions (context, max pages + empty queue) are met.
		c.wg.Wait()
		c.log.Info("Redis queue feeder goroutine has stopped.")

		close(doneChan) // Signal that all crawling activities are complete
	}()

	// Block the main `Start()` goroutine until all crawling activities are done.
	<-doneChan
	c.log.Info("All crawling activities (Colly queue and Redis feeder) have completed.")

	// --- End Core Waiting Logic ---

	stats := c.metrics.GetStats()
	c.log.WithFields(logrus.Fields{
		"pages_processed":  stats["pages_processed"],
		"pages_successful": stats["pages_successful"],
		"pages_failed":     stats["pages_failed"],
		"duration_seconds": stats["uptime_seconds"],
	}).Info("Crawling process completed")

	return nil
}

// Shutdown gracefully shuts down the crawler by stopping background goroutines
// and closing connections.
func (c *Crawler) Shutdown() {
	c.shutdownOnce.Do(func() {
		c.log.Info("Initiating crawler shutdown...")
		close(c.shutdown) // Signal background goroutines to stop
		c.cancel()        // Signal context cancellation to active operations

		// Wait for all goroutines (health check, metrics logger) to finish.
		// Colly's Wait() should already be done or will be canceled by ctx.
		done := make(chan struct{})
		go func() {
			c.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			c.log.Info("All background goroutines finished.")
		case <-time.After(10 * time.Second):
			c.log.Warn("Timeout waiting for background goroutines to finish. Some might still be running.")
		}

		// Close external connections
		if c.producer != nil {
			if err := c.producer.Close(); err != nil {
				c.log.Errorf("Failed to close Kafka producer: %v", err)
			} else {
				c.log.Info("Kafka producer closed.")
			}
		}

		if c.redisClient != nil {
			if err := c.redisClient.Close(); err != nil {
				c.log.Errorf("Failed to close Redis client: %v", err)
			} else {
				c.log.Info("Redis client closed.")
			}
		}

		c.log.Info("Crawler shutdown complete.")
	})
}

func main() {
	crawler, err := NewCrawler()
	if err != nil {
		logrus.Fatalf("Failed to create crawler: %v", err)
	}

	// Set up OS signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Channel to signal when the main crawler Start() routine completes
	done := make(chan struct{})

	// Start the main crawling process in a goroutine
	go func() {
		if err := crawler.Start(); err != nil {
			logrus.Errorf("Crawler encountered a critical error: %v", err)
		}
		close(done) // Signal that the crawling process has finished
	}()

	// Wait for either an OS signal or the crawler to complete naturally
	select {
	case <-sigChan:
		logrus.Info("Received OS shutdown signal. Initiating graceful shutdown...")
	case <-done:
		logrus.Info("Crawler completed all tasks naturally. Initiating graceful shutdown...")
	}

	// Always ensure shutdown is called
	crawler.Shutdown()
}
