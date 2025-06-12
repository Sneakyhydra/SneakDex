package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
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
	MaxContentSize int           `envconfig:"MAX_CONTENT_SIZE" default:"2621440"` // 2.5MB/10MB

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

// IncrementMetrics provides atomic increment methods for crawler metrics
func (metric *Metrics) IncrementPagesProcessed()  { atomic.AddInt64(&metric.PagesProcessed, 1) }
func (metric *Metrics) IncrementPagesSuccessful() { atomic.AddInt64(&metric.PagesSuccessful, 1) }
func (metric *Metrics) IncrementPagesFailed()     { atomic.AddInt64(&metric.PagesFailed, 1) }
func (metric *Metrics) IncrementKafkaSuccessful() { atomic.AddInt64(&metric.KafkaSuccessful, 1) }
func (metric *Metrics) IncrementKafkaFailed()     { atomic.AddInt64(&metric.KafkaFailed, 1) }
func (metric *Metrics) IncrementRedisSuccessful() { atomic.AddInt64(&metric.RedisSuccessful, 1) }
func (metric *Metrics) IncrementRedisFailed()     { atomic.AddInt64(&metric.RedisFailed, 1) }

// Uptime returns the duration since the crawler started.
func (metric *Metrics) Uptime() time.Duration { return time.Since(metric.StartTime) }

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

// URLValidator handles URL validation with caching and improved performance
type URLValidator struct {
	whitelist []string
	blacklist []string
	log       *logrus.Logger

	// Caching for DNS lookups
	dnsCache        sync.Map // map[string]DNSResult
	dnsCacheTimeout time.Duration

	// Caching for domain checks
	domainCache sync.Map // map[string]bool

	// Configuration
	allowPrivateIPs bool
	allowLoopback   bool
	skipDNSCheck    bool
	maxURLLength    int
}

type DNSResult struct {
	IPs       []net.IP
	Timestamp time.Time
	Valid     bool
}

// Crawler represents the main crawler instance
type Crawler struct {
	// Core configuration and dependencies
	config      Config
	log         *logrus.Logger
	redisClient *redis.Client
	producer    sarama.SyncProducer
	metrics     *Metrics

	// Context and cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Concurrency control
	wg            sync.WaitGroup
	shutdown      chan struct{}
	shutdownOnce  sync.Once
	inFlightPages int64

	// URL management and filtering
	whitelist    []string
	blacklist    []string
	visited      sync.Map
	requeued     sync.Map
	urlValidator *URLValidator
}

// IncrementInFlightPages increments the count of in-flight pages being processed
func (crawler *Crawler) IncrementInFlightPages() { atomic.AddInt64(&crawler.inFlightPages, 1) }

// DecrementInFlightPages decrements the count of in-flight pages being processed
func (crawler *Crawler) DecrementInFlightPages() { atomic.AddInt64(&crawler.inFlightPages, -1) }

// GetInFlightPages returns the current number of in-flight pages being processed
func (crawler *Crawler) GetInFlightPages() int64 { return atomic.LoadInt64(&crawler.inFlightPages) }

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
			backoff := time.Duration(1<<uint(attempt)) * 100 * time.Millisecond // 2^attempt * 100ms
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
	kafkaConfig.Producer.MaxMessageBytes = int(crawler.config.MaxContentSize)

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
			backoff := time.Duration(1<<uint(attempt)) * 100 * time.Millisecond // 2^attempt * 100ms
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed to create kafka producer after %d attempts. please ensure kafka is running on %s", crawler.config.KafkaRetryMax, strings.Join(brokers, ","))
}

// NewURLValidator creates a new URL validator with default settings
func NewURLValidator(whitelist, blacklist []string, log *logrus.Logger) *URLValidator {
	return &URLValidator{
		whitelist:       whitelist,
		blacklist:       blacklist,
		log:             log,
		dnsCacheTimeout: 5 * time.Minute, // Cache DNS results for 5 minutes
		allowPrivateIPs: false,
		allowLoopback:   false,
		skipDNSCheck:    false,
		maxURLLength:    2048, // Reasonable URL length limit
	}
}

// SetDNSCacheTimeout sets the DNS cache timeout duration
func (urlValidator *URLValidator) SetDNSCacheTimeout(timeout time.Duration) {
	urlValidator.dnsCacheTimeout = timeout
}

// SetAllowPrivateIPs configures whether private IPs are allowed
func (urlValidator *URLValidator) SetAllowPrivateIPs(allow bool) {
	urlValidator.allowPrivateIPs = allow
}

// SetAllowLoopback configures whether loopback are allowed
func (urlValidator *URLValidator) SetAllowLoopback(allow bool) {
	urlValidator.allowPrivateIPs = allow
}

// SetSkipDNSCheck configures whether to skip DNS validation
func (urlValidator *URLValidator) SetSkipDNSCheck(skip bool) {
	urlValidator.skipDNSCheck = skip
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

	// Initialize URL Validator
	crawler.urlValidator = NewURLValidator(crawler.whitelist, crawler.blacklist, crawler.log)

	// Configure URL Validator
	crawler.urlValidator.SetDNSCacheTimeout(10 * time.Minute)
	crawler.urlValidator.SetSkipDNSCheck(false) // Set to false for high-performance crawling. Not Safe
	crawler.urlValidator.SetAllowPrivateIPs(false)
	crawler.urlValidator.SetAllowLoopback(false)

	return crawler, nil
}

// ------------------------------------------------------------------
// Crawler URL Validation and Redis Interaction
// ------------------------------------------------------------------

// IsValidURL checks if a URL is valid and not blocked by blacklist or whitelist rules
func (urlValidator *URLValidator) IsValidURL(rawURL string) bool {
	// Quick checks first (cheapest operations)
	if rawURL == "" || len(rawURL) > urlValidator.maxURLLength {
		return false
	}

	// Parse the URL to ensure it is well-formed
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		urlValidator.log.WithFields(logrus.Fields{"url": rawURL, "error": err}).Debug("Invalid URL format")
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

	// Check domain-based filtering first (before expensive DNS lookup)
	if !urlValidator.isDomainAllowed(host) {
		return false
	}

	// DNS validation (most expensive operation, do last)
	if !urlValidator.skipDNSCheck && !urlValidator.isIPValid(host) {
		return false
	}

	return true
}

// isDomainAllowed checks whitelist and blacklist rules with caching
func (urlValidator *URLValidator) isDomainAllowed(host string) bool {
	// Check cache first
	if cached, exists := urlValidator.domainCache.Load(host); exists {
		if valid, ok := cached.(bool); ok {
			return valid
		}
	}

	allowed := urlValidator.checkDomainRules(host)

	// Cache the result
	urlValidator.domainCache.Store(host, allowed)

	return allowed
}

// checkDomainRules performs the actual domain filtering logic
func (urlValidator *URLValidator) checkDomainRules(host string) bool {
	// Check blacklist first (fail fast)
	for _, blocked := range urlValidator.blacklist {
		blocked = strings.ToLower(blocked)
		if urlValidator.matchesDomain(host, blocked) {
			urlValidator.log.WithFields(logrus.Fields{"host": host, "blocked_by": blocked}).Debug("Host blocked by blacklist")
			return false
		}
	}

	// Check whitelist if configured
	if len(urlValidator.whitelist) > 0 {
		for _, allowedDomain := range urlValidator.whitelist {
			allowedDomain = strings.ToLower(allowedDomain)
			if urlValidator.matchesDomain(host, allowedDomain) {
				return true
			}
		}
		// If whitelist is configured but no match found, deny
		urlValidator.log.WithFields(logrus.Fields{"host": host}).Debug("Host not in whitelist")
		return false
	}

	return true
}

// matchesDomain checks if host matches domain pattern
func (urlValidator *URLValidator) matchesDomain(host, domain string) bool {
	// Exact match
	if host == domain {
		return true
	}

	// Subdomain match
	if strings.HasSuffix(host, "."+domain) {
		return true
	}

	return false
}

// isIPValid checks if the host resolves to valid IPs with caching
func (urlValidator *URLValidator) isIPValid(host string) bool {
	// Check if it's already an IP address
	if ip := net.ParseIP(host); ip != nil {
		return urlValidator.isIPAllowed(ip)
	}

	// Check DNS cache first
	if cached, exists := urlValidator.dnsCache.Load(host); exists {
		if result, ok := cached.(DNSResult); ok {
			// Check if cache is still valid
			if time.Since(result.Timestamp) < urlValidator.dnsCacheTimeout {
				if !result.Valid {
					return false
				}
				// Check cached IPs
				return urlValidator.areIPsAllowed(result.IPs)
			}
		}
	}

	// Perform DNS lookup
	ips, err := net.LookupIP(host)
	dnsResult := DNSResult{
		IPs:       ips,
		Timestamp: time.Now(),
		Valid:     err == nil,
	}

	// Cache the result
	urlValidator.dnsCache.Store(host, dnsResult)

	if err != nil {
		urlValidator.log.WithFields(logrus.Fields{"host": host, "error": err}).Debug("DNS lookup failed")
		return false
	}

	return urlValidator.areIPsAllowed(ips)
}

// areIPsAllowed checks if any of the IPs are allowed
func (urlValidator *URLValidator) areIPsAllowed(ips []net.IP) bool {
	for _, ip := range ips {
		if urlValidator.isIPAllowed(ip) {
			return true
		}
	}
	return false
}

// isIPAllowed checks if a single IP is allowed based on configuration
func (urlValidator *URLValidator) isIPAllowed(ip net.IP) bool {
	if !urlValidator.allowLoopback && ip.IsLoopback() {
		urlValidator.log.WithFields(logrus.Fields{"ip": ip.String()}).Debug("Blocked loopback IP")
		return false
	}

	if !urlValidator.allowPrivateIPs && ip.IsPrivate() {
		urlValidator.log.WithFields(logrus.Fields{"ip": ip.String()}).Debug("Blocked private IP")
		return false
	}

	return true
}

// ClearCache clears all cached results
func (urlValidator *URLValidator) ClearCache() {
	urlValidator.dnsCache = sync.Map{}
	urlValidator.domainCache = sync.Map{}
}

// UpdateWhitelist updates the whitelist and clears domain cache
func (urlValidator *URLValidator) UpdateWhitelist(whitelist []string) {
	urlValidator.whitelist = whitelist
	urlValidator.domainCache = sync.Map{} // Clear cache since rules changed
}

// UpdateBlacklist updates the blacklist and clears domain cache
func (urlValidator *URLValidator) UpdateBlacklist(blacklist []string) {
	urlValidator.blacklist = blacklist
	urlValidator.domainCache = sync.Map{} // Clear cache since rules changed
}

// isValidURL checks if a URL is valid and not blocked by blacklist or whitelist rules
func (crawler *Crawler) isValidURL(rawURL string) bool {
	return crawler.urlValidator.IsValidURL(rawURL)
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
		return false, fmt.Errorf("failed to check visited urls in redis: %w", err)
	}
	if visited {
		crawler.cacheAndLogFound(url, "visited")
		return true, nil
	}

	pending, err := crawler.checkRedisSet("crawler:pending_urls", url)
	if err != nil {
		return false, fmt.Errorf("failed to check pending urls in redis: %w", err)
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
func (crawler *Crawler) feedCollyFromRedisQueue(collector *colly.Collector) {
	defer crawler.wg.Done() // Decrement WaitGroup counter when this goroutine exits
	crawler.log.Info("Starting goroutine to feed URLs from Redis queue to Colly")

	// Ticker to periodically check the Redis queue
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	// Counter to track consecutive empty queue checks
	emptyQueueChecks := 0
	const maxEmptyChecks = 5

	for {
		select {
		case <-crawler.ctx.Done(): // Primary shutdown signal: context cancellation
			crawler.log.Info("Stopping Redis queue feeder goroutine due to context cancellation.")
			return
		case <-ticker.C:
			// Check if max pages limit is reached. If so, terminate.
			if atomic.LoadInt64(&crawler.metrics.PagesProcessed) >= crawler.config.MaxPages {
				crawler.log.Debug("Max pages limit reached. Feeder will stop.")
				// Return immediately so we can continue our progress when we restart the crawler
				return
			}

			// Attempt to pull a URL from the 'pending' set
			url, err := crawler.redisClient.SPop(crawler.ctx, "crawler:pending_urls").Result()
			if err == redis.Nil {
				// Redis queue is empty
				emptyQueueChecks++
				crawler.log.WithFields(logrus.Fields{"empty_checks": emptyQueueChecks}).Trace("Redis pending queue empty.")

				// Check for termination conditions if queue is consistently empty
				if emptyQueueChecks >= maxEmptyChecks {
					// If queue is consistently empty AND (max pages reached OR context is cancelled)
					if atomic.LoadInt64(&crawler.metrics.PagesProcessed) >= crawler.config.MaxPages || crawler.ctx.Err() != nil || crawler.GetInFlightPages() == 0 { // ctx.Done() always returns a channel so !=nill is always true
						crawler.log.Info("Redis pending queue consistently empty and termination condition met. Stopping feeder.")
						return
					}
					// If queue is empty but we haven't reached max pages or in-flight pages or context is not cancelled, just continue checking
					crawler.log.Debug("Redis pending queue empty, but max pages not reached. Continuing to check...")
				}
				continue
			} else if err != nil {
				// An actual error occurred with Redis (not just empty queue)
				crawler.log.WithFields(logrus.Fields{"error": err}).Error("Failed to pop URL from Redis pending queue, retrying...")
				crawler.metrics.IncrementRedisFailed()
				continue
			}

			// If a URL was successfully popped, reset the empty queue counter
			emptyQueueChecks = 0

			crawler.log.WithFields(logrus.Fields{"url": url}).Debug("Pulled URL from Redis pending queue for visit")

			// Feed the URL to Colly. Colly's internal Limit rules will manage concurrency.
			if err := collector.Visit(url); err != nil {
				// This error means Colly failed to even initiate the visit so collector.onError won't be triggered (e.g., malformed URL given to Colly).
				// It's crucial to mark this URL as visited (failed) in Redis,
				// otherwise it might be re-added to pending if discovered again, leading to an infinite loop.
				crawler.log.WithFields(logrus.Fields{"url": url, "error": err}).Warn("Failed to initiate Colly visit for URL from Redis queue (e.g., invalid format). Marking as failed visited.")
				_, redisErr := crawler.redisClient.SAdd(crawler.ctx, "crawler:visited_urls", url).Result()
				if redisErr != nil {
					crawler.log.WithFields(logrus.Fields{"url": url, "error": redisErr}).Error("Failed to add URL to visited_urls Redis set after Colly visit initiation failure")
					crawler.metrics.IncrementRedisFailed()
				}
				crawler.metrics.IncrementPagesFailed()
			}
		}
	}
}

// sendToKafka sends the scraped HTML content to Kafka.
func (crawler *Crawler) sendToKafka(url, html string) error {
	if int(len(html)) > crawler.config.MaxContentSize {
		return &CrawlError{
			URL:       url,
			Operation: "kafka_send",
			Err:       fmt.Errorf("content size %d exceeds limit %d", len(html), crawler.config.MaxContentSize),
			Retry:     false, // No point in retrying for size limit
		}
	}

	// Create the message
	msg := &sarama.ProducerMessage{
		Topic:     crawler.config.KafkaTopic,
		Key:       sarama.StringEncoder(url),
		Value:     sarama.StringEncoder(html),
		Timestamp: time.Now(),
	}

	var lastErr error
	for attempt := 1; attempt <= crawler.config.KafkaRetryMax; attempt++ {
		_, _, err := crawler.producer.SendMessage(msg)
		if err == nil {
			crawler.metrics.IncrementKafkaSuccessful()
			return nil
		}

		lastErr = err
		crawler.log.Warnf("Kafka SendMessage attempt %d/%d failed for URL %s: %v",
			attempt, crawler.config.KafkaRetryMax, url, err)

		// Don't sleep after the last attempt
		if attempt < crawler.config.KafkaRetryMax {
			// Exponential backoff for retries
			backoff := time.Duration(1<<uint(attempt)) * 100 * time.Millisecond // 2^attempt * 100ms
			time.Sleep(backoff)
		}
	}

	crawler.metrics.IncrementKafkaFailed()
	return &CrawlError{
		URL:       url,
		Operation: "kafka_send",
		Err:       fmt.Errorf("failed to send to kafka after %d attempts: %w", crawler.config.KafkaRetryMax, lastErr),
		Retry:     true, // Mark as retriable
	}
}

// setupCollyCollector configures the colly collector with callbacks for crawling logic.
func (crawler *Crawler) setupCollyCollector() *colly.Collector {
	// Setup Colly collector options
	options := []colly.CollectorOption{
		colly.MaxDepth(crawler.config.CrawlDepth),
		colly.Async(true), // Enable asynchronous requests, managing concurrency internally
		colly.UserAgent(crawler.config.UserAgent),
		colly.ParseHTTPErrorResponse(), // Parse 4xx/5xx responses for better error handling
		colly.DetectCharset(),          // Auto-detect and convert character encoding
	}

	if len(crawler.blacklist) > 0 {
		options = append(options, colly.DisallowedDomains(crawler.blacklist...))
	}
	if len(crawler.whitelist) > 0 {
		options = append(options, colly.AllowedDomains(crawler.whitelist...))
	}
	if crawler.config.EnableDebug {
		options = append(options, colly.Debugger(&debug.LogDebugger{}))
	}

	// Create the collector
	collector := colly.NewCollector(options...)

	// Apply rate limits for all domains
	err := collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: crawler.config.MaxConcurrency,
		Delay:       crawler.config.RequestDelay,
		RandomDelay: crawler.config.RequestDelay / 2,
	})
	if err != nil {
		crawler.log.WithFields(logrus.Fields{"error": err}).Error("Failed to set rate limit")
		// Continue
	}

	collector.SetRequestTimeout(crawler.config.RequestTimeout)

	var skipExts = map[string]struct{}{
		".pdf": {}, ".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".css": {}, ".js": {}, ".ico": {},
		".svg": {}, ".woff": {}, ".ttf": {}, ".mp4": {}, ".mp3": {}, ".zip": {}, ".exe": {},
	}

	// Request filtering and header setting
	collector.OnRequest(func(r *colly.Request) {
		// Check for context cancellation before proceeding with the request
		select {
		case <-crawler.ctx.Done():
			crawler.log.WithFields(logrus.Fields{"url": r.URL.String()}).Debug("Request aborted due to shutdown")
			r.Abort()
			return
		default:
			// Continue
		}

		urlStr := r.URL.String()
		// Skip common file extensions
		ext := strings.ToLower(path.Ext(r.URL.Path))
		if _, skip := skipExts[ext]; skip {
			crawler.log.WithFields(logrus.Fields{"url": urlStr, "ext": ext}).Debug("Skipping URL due to file extension")
			r.Abort()
			return
		}

		// Add common headers to appear more like a real browser
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Accept-Encoding", "gzip, deflate")
		r.Headers.Set("DNT", "1")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")

		crawler.IncrementInFlightPages()
		crawler.log.WithFields(logrus.Fields{"url": urlStr}).Debug("Visiting URL")
	})

	// Handle HTML pages
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		// Check for context cancellation
		select {
		case <-crawler.ctx.Done():
			crawler.log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("HTML processing skipped due to shutdown")
			return
		default:
			// Continue
		}

		// Stop processing if max pages limit is reached
		if atomic.LoadInt64(&crawler.metrics.PagesProcessed) >= crawler.config.MaxPages {
			crawler.log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("Max pages limit reached, stopping HTML processing")
			return
		}

		crawler.metrics.IncrementPagesProcessed()
		defer crawler.DecrementInFlightPages()
		url := e.Request.URL.String()

		html, err := e.DOM.Html()
		if err != nil {
			crawler.log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to extract HTML")
			crawler.metrics.IncrementPagesFailed()
			// IMPORTANT: Mark as visited even if HTML extraction fails
			_, redisErr := crawler.redisClient.SAdd(crawler.ctx, "crawler:visited_urls", url).Result()
			if redisErr != nil {
				crawler.log.WithFields(logrus.Fields{"url": url, "error": redisErr}).Error("Failed to add URL to visited_urls Redis set after HTML extraction failure")
				crawler.metrics.IncrementRedisFailed()
			}
			return
		}

		if err := crawler.sendToKafka(url, html); err != nil {
			var crawlErr *CrawlError
			if errors.As(err, &crawlErr) && crawlErr.Retry {
				if _, ok := crawler.requeued.Load(url); ok {
					crawler.log.WithFields(logrus.Fields{"url": url}).Trace("URL already requeued once. Will be marked as visited")
					crawler.requeued.Delete(url)
				} else {
					// Re-queue URL instead of marking as visited
					crawler.log.WithFields(logrus.Fields{"url": url, "error": err}).Warn("Retriable error occurred, requeuing URL")

					_, redisErr := crawler.redisClient.SAdd(crawler.ctx, "crawler:pending_urls", url).Result()
					if redisErr != nil {
						crawler.log.WithFields(logrus.Fields{"url": url, "error": redisErr}).Error("Failed to requeue URL after Kafka retryable error")
						crawler.metrics.IncrementRedisFailed()
					}
					crawler.requeued.Store(url, struct{}{})
					return
				}
			}

			crawler.log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to send to Kafka")
			crawler.metrics.IncrementPagesFailed()
			// IMPORTANT: Mark as visited even if sending to Kafka fails
			_, redisErr := crawler.redisClient.SAdd(crawler.ctx, "crawler:visited_urls", url).Result()
			if redisErr != nil {
				crawler.log.WithFields(logrus.Fields{"url": url, "error": redisErr}).Error("Failed to add URL to visited_urls Redis set after Kafka failure")
				crawler.metrics.IncrementRedisFailed()
			}
			return
		}

		crawler.metrics.IncrementPagesSuccessful()
		crawler.log.WithFields(logrus.Fields{"url": url, "content_size": len(html)}).Debug("Page processed successfully")

		// IMPORTANT: Mark as visited in Redis after successful processing
		_, err = crawler.redisClient.SAdd(crawler.ctx, "crawler:visited_urls", url).Result()
		if err != nil {
			crawler.log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to add URL to visited_urls Redis set")
			crawler.metrics.IncrementRedisFailed()
		}
	})

	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		// Check for context cancellation
		select {
		case <-crawler.ctx.Done():
			crawler.log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("Link extraction skipped due to shutdown")
			return
		default:
			// Continue
		}

		// Stop processing if max pages limit is reached
		if atomic.LoadInt64(&crawler.metrics.PagesProcessed) >= crawler.config.MaxPages {
			crawler.log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("Max pages limit reached, stopping link extraction")
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
		if !crawler.isValidURL(absoluteURL) {
			return
		}

		normalized, err := normalizeURL(absoluteURL)
		if err != nil {
			crawler.log.WithFields(logrus.Fields{"url": absoluteURL, "error": err}).Debug("Failed to normalize URL")
			return
		}

		// Use the enhanced isURLVisited to check both 'visited' and 'pending' sets
		visitedOrPending, err := crawler.isURLVisited(normalized)
		if err != nil {
			crawler.log.WithFields(logrus.Fields{"url": normalized, "error": err}).Error("Failed to check URL visited/pending status")
			crawler.metrics.IncrementRedisFailed() // Increment failed metrics if Redis check fails
			return
		}

		if !visitedOrPending {
			crawler.log.WithFields(logrus.Fields{"url": normalized}).Debug("Adding new URL to Redis pending queue")
			// Add to Redis 'pending_urls' set instead of calling Colly's Visit directly
			_, err := crawler.redisClient.SAdd(crawler.ctx, "crawler:pending_urls", normalized).Result()
			if err != nil {
				crawler.log.WithFields(logrus.Fields{"url": normalized, "error": err}).Error("Failed to add URL to Redis pending queue")
				crawler.metrics.IncrementRedisFailed()
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

		if !isNetworkError || crawler.config.EnableDebug {
			crawler.log.WithFields(logrus.Fields{
				"url":         r.Request.URL.String(),
				"status_code": r.StatusCode,
				"error":       err,
			}).Warn("Request failed")
		} else {
			crawler.log.WithFields(logrus.Fields{
				"url":         r.Request.URL.String(),
				"status_code": r.StatusCode,
				"error":       err,
			}).Debug("Suppressed network error") // Log as debug if suppressed
		}
		crawler.metrics.IncrementPagesFailed()

		// IMPORTANT: Mark as visited in Redis, even if the request failed, to prevent re-attempts for this URL.
		url := r.Request.URL.String()
		_, redisErr := crawler.redisClient.SAdd(crawler.ctx, "crawler:visited_urls", url).Result()
		if redisErr != nil {
			crawler.log.WithFields(logrus.Fields{"url": url, "error": redisErr}).Error("Failed to add URL to visited_urls Redis set on error")
			crawler.metrics.IncrementRedisFailed()
		}
		crawler.DecrementInFlightPages()
	})

	// Debug logging on successful response
	if crawler.config.EnableDebug {
		collector.OnResponse(func(r *colly.Response) {
			crawler.log.WithFields(logrus.Fields{
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
func (crawler *Crawler) startHealthCheck() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Also check Redis and Kafka health here for a more robust health check
		redisStatus := "ok"
		kafkaStatus := "ok"

		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second) // Use request context, short timeout
		defer cancel()

		if err := crawler.redisClient.Ping(ctx).Err(); err != nil {
			redisStatus = fmt.Sprintf("error: %v", err)
		}

		// Kafka health check:
		// A non-invasive check is just to ensure the producer is initialized (not nil).
		// Sarama's SyncProducer handles internal connection management and retries.
		if crawler.producer == nil {
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
		stats := crawler.metrics.GetStats()
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
		Addr:         fmt.Sprintf(":%d", crawler.config.HealthCheckPort),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	crawler.wg.Add(1)
	go func() {
		defer crawler.wg.Done()
		crawler.log.Infof("Health check server starting on port %d", crawler.config.HealthCheckPort)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			crawler.log.Errorf("Health check server error: %v", err)
		}
	}()

	// Graceful shutdown for health check server
	go func() {
		<-crawler.shutdown // Wait for the main shutdown signal
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			crawler.log.Errorf("Health check server shutdown error: %v", err)
		} else {
			crawler.log.Info("Health check server shut down gracefully")
		}
	}()
}

// logMetricsPeriodically logs metrics every 30 seconds
func (crawler *Crawler) logMetricsPeriodically() {
	ticker := time.NewTicker(10 * time.Second)
	crawler.wg.Add(1)
	go func() {
		defer crawler.wg.Done()
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stats := crawler.metrics.GetStats()
				pagesPerSecond := float64(0)
				if stats["uptime_seconds"].(float64) > 0 {
					pagesPerSecond = float64(stats["pages_processed"].(int64)) / stats["uptime_seconds"].(float64)
				}

				crawler.log.WithFields(logrus.Fields{
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
			case <-crawler.shutdown:
				crawler.log.Info("Stopping periodic metrics logging")
				return
			}
		}
	}()
}

func (crawler *Crawler) Start() error {
	crawler.log.Info("Starting web crawler with persistent Redis queue. Waiting indefinitely until completion.")
	crawler.log.WithFields(logrus.Fields{
		"kafka": map[string]any{
			"kafka_brokers":   crawler.config.KafkaBrokers,
			"kafka_topic":     crawler.config.KafkaTopic,
			"kafka_retry_max": crawler.config.KafkaRetryMax,
		},
		"redis": map[string]any{
			"redis_host":      crawler.config.RedisHost,
			"redis_port":      crawler.config.RedisPort,
			"redis_password":  crawler.config.RedisPassword,
			"redis_db":        crawler.config.RedisDB,
			"redis_timeout":   crawler.config.RedisTimeout,
			"redis_retry_max": crawler.config.RedisRetryMax,
		},
		"crawling_behavior": map[string]any{
			"start_urls":    crawler.config.StartURLs,
			"max_pages":     crawler.config.MaxPages,
			"crawl_depth":   crawler.config.CrawlDepth,
			"url_whitelist": crawler.config.URLWhitelist,
			"url_blacklist": crawler.config.URLBlacklist,
		},
		"performance_and_limits": map[string]any{
			"max_concurrency":  crawler.config.MaxConcurrency,
			"request_timeout":  crawler.config.RequestTimeout,
			"request_delay":    crawler.config.RequestDelay,
			"max_content_size": crawler.config.MaxContentSize,
		},
		"application_settings": map[string]any{
			"log_level":         crawler.config.LogLevel,
			"user_agent":        crawler.config.UserAgent,
			"enable_debug":      crawler.config.EnableDebug,
			"health_check_port": crawler.config.HealthCheckPort,
		},
	}).Info("Crawler configuration")

	crawler.startHealthCheck()
	crawler.logMetricsPeriodically()

	collector := crawler.setupCollyCollector()

	// Seed Redis 'pending_urls' with configured start URLs
	startURLs := strings.SplitSeq(crawler.config.StartURLs, ",")
	for rawURL := range startURLs {
		url := strings.TrimSpace(rawURL)
		if !crawler.isValidURL(url) {
			crawler.log.Warnf("Skipping invalid start URL: %s", url)
			continue
		}

		visitedOrPending, err := crawler.isURLVisited(url)
		if err != nil {
			crawler.log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to check start URL status in Redis, skipping")
			continue
		}
		if visitedOrPending {
			crawler.log.WithFields(logrus.Fields{"url": url}).Info("Start URL already visited or pending, skipping initial queueing")
			continue
		}

		// Add to Redis 'pending_urls' set
		_, err = crawler.redisClient.SAdd(crawler.ctx, "crawler:pending_urls", url).Result()
		if err != nil {
			crawler.log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to add start URL to Redis pending queue")
		} else {
			crawler.log.WithFields(logrus.Fields{"url": url}).Info("Added start URL to Redis pending queue")
		}
	}

	// --- Core Waiting Logic ---

	// Increment WaitGroup for the feedCollyFromRedisQueue goroutine
	crawler.wg.Add(1)
	go crawler.feedCollyFromRedisQueue(collector)

	crawler.log.Info("Crawler started. Blocking until all crawling activities are complete...")

	// Use a separate goroutine to orchestrate the waiting for Colly and the feeder.
	// This allows the main `Start` function to simply block until all work is signaled as done.
	doneChan := make(chan struct{})
	go func() {
		// First, wait for Colly to finish processing all URLs it has been given.
		// This will only complete once `feedCollyFromRedisQueue` stops feeding it URLs
		// (either due to context cancellation, max pages, or empty Redis queue).
		collector.Wait()
		crawler.log.Info("Colly collector finished processing its internal queue.")

		// Then, wait for the feedCollyFromRedisQueue goroutine to exit.
		// It will exit when its internal conditions (context, max pages + empty queue) are met.
		crawler.wg.Wait()
		crawler.log.Info("Redis queue feeder goroutine has stopped.")

		close(doneChan) // Signal that all crawling activities are complete
	}()

	// Block the main `Start()` goroutine until all crawling activities are done.
	<-doneChan
	crawler.log.Info("All crawling activities (Colly queue and Redis feeder) have completed.")

	// --- End Core Waiting Logic ---

	stats := crawler.metrics.GetStats()
	crawler.log.WithFields(logrus.Fields{
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
	crawler.shutdownOnce.Do(func() {
		crawler.log.Info("Initiating crawler shutdown...")
		close(crawler.shutdown) // Signal background goroutines to stop
		crawler.cancel()        // Signal context cancellation to active operations

		// Wait for all goroutines (health check, metrics logger) to finish.
		// Colly's Wait() should already be done or will be canceled by ctx.
		done := make(chan struct{})
		go func() {
			crawler.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			crawler.log.Info("All background goroutines finished.")
		case <-time.After(10 * time.Second):
			crawler.log.Warn("Timeout waiting for background goroutines to finish. Some might still be running.")
		}

		// Close external connections
		if crawler.producer != nil {
			if err := crawler.producer.Close(); err != nil {
				crawler.log.Errorf("Failed to close Kafka producer: %v", err)
			} else {
				crawler.log.Info("Kafka producer closed.")
			}
		}

		if crawler.redisClient != nil {
			if err := crawler.redisClient.Close(); err != nil {
				crawler.log.Errorf("Failed to close Redis client: %v", err)
			} else {
				crawler.log.Info("Redis client closed.")
			}
		}

		crawler.log.Info("Crawler shutdown complete.")
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
