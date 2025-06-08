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

// Config holds the crawler configuration from environment variables
type Config struct {
	KafkaBrokers    string        `envconfig:"KAFKA_BROKERS" default:"kafka:9092"`
	KafkaTopic      string        `envconfig:"KAFKA_TOPIC_HTML" default:"raw-html"`
	StartURLs       string        `envconfig:"START_URLS" default:"https://en.wikipedia.org,https://news.ycombinator.com"`
	CrawlDepth      int           `envconfig:"CRAWL_DEPTH" default:"3"`
	MaxPages        int64         `envconfig:"MAX_PAGES" default:"1000"`
	LogLevel        string        `envconfig:"LOG_LEVEL" default:"info"`
	RedisAddr       string        `envconfig:"REDIS_ADDR" default:"redis:6379"`
	RedisPassword   string        `envconfig:"REDIS_PASSWORD" default:""`
	RedisDB         int           `envconfig:"REDIS_DB" default:"0"`
	RequestTimeout  time.Duration `envconfig:"REQUEST_TIMEOUT" default:"30s"`
	RedisTimeout    time.Duration `envconfig:"REDIS_TIMEOUT" default:"5s"`
	MaxConcurrency  int           `envconfig:"MAX_CONCURRENCY" default:"50"`
	RequestDelay    time.Duration `envconfig:"REQUEST_DELAY" default:"100ms"`
	URLWhitelist    string        `envconfig:"URL_WHITELIST" default:""`
	URLBlacklist    string        `envconfig:"URL_BLACKLIST" default:""`
	MaxContentSize  int64         `envconfig:"MAX_CONTENT_SIZE" default:"5242880"` // 5MB
	UserAgent       string        `envconfig:"USER_AGENT" default:"WebCrawler/1.0"`
	EnableDebug     bool          `envconfig:"ENABLE_DEBUG" default:"false"`
	HealthCheckPort int           `envconfig:"HEALTH_CHECK_PORT" default:"8080"`
	KafkaRetryMax   int           `envconfig:"KAFKA_RETRY_MAX" default:"3"`
	RedisRetryMax   int           `envconfig:"REDIS_RETRY_MAX" default:"3"`
}

// CrawlError represents a structured error type for crawling operations
type CrawlError struct {
	URL       string
	Operation string
	Err       error
	Retry     bool
	Timestamp time.Time
}

func (e *CrawlError) Error() string {
	return fmt.Sprintf("CrawlError[%s]: %s - %v (retry: %t)", e.Operation, e.URL, e.Err, e.Retry)
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

func (m *Metrics) IncrementPagesProcessed()  { atomic.AddInt64(&m.PagesProcessed, 1) }
func (m *Metrics) IncrementPagesSuccessful() { atomic.AddInt64(&m.PagesSuccessful, 1) }
func (m *Metrics) IncrementPagesFailed()     { atomic.AddInt64(&m.PagesFailed, 1) }
func (m *Metrics) IncrementKafkaSuccessful() { atomic.AddInt64(&m.KafkaSuccessful, 1) }
func (m *Metrics) IncrementKafkaFailed()     { atomic.AddInt64(&m.KafkaFailed, 1) }
func (m *Metrics) IncrementRedisSuccessful() { atomic.AddInt64(&m.RedisSuccessful, 1) }
func (m *Metrics) IncrementRedisFailed()     { atomic.AddInt64(&m.RedisFailed, 1) }

func (m *Metrics) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"pages_processed":  atomic.LoadInt64(&m.PagesProcessed),
		"pages_successful": atomic.LoadInt64(&m.PagesSuccessful),
		"pages_failed":     atomic.LoadInt64(&m.PagesFailed),
		"kafka_successful": atomic.LoadInt64(&m.KafkaSuccessful),
		"kafka_failed":     atomic.LoadInt64(&m.KafkaFailed),
		"redis_successful": atomic.LoadInt64(&m.RedisSuccessful),
		"redis_failed":     atomic.LoadInt64(&m.RedisFailed),
		"uptime_seconds":   time.Since(m.StartTime).Seconds(),
	}
}

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

// NewCrawler creates a new crawler instance
func NewCrawler() (*Crawler, error) {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}

	log := logrus.New()
	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	log.SetLevel(level)
	log.SetFormatter(&logrus.JSONFormatter{})

	ctx, cancel := context.WithCancel(context.Background())

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

	if err := crawler.initializeRedis(); err != nil {
		cancel()
		return nil, err
	}

	if err := crawler.initializeKafka(); err != nil {
		cancel()
		return nil, err
	}

	return crawler, nil
}

// initializeRedis sets up Redis client with proper configuration
func (c *Crawler) initializeRedis() error {
	c.redisClient = redis.NewClient(&redis.Options{
		Addr:         c.config.RedisAddr,
		Password:     c.config.RedisPassword,
		DB:           c.config.RedisDB,
		DialTimeout:  c.config.RedisTimeout,
		ReadTimeout:  c.config.RedisTimeout,
		WriteTimeout: c.config.RedisTimeout,
		MaxRetries:   c.config.RedisRetryMax,
	})

	// Test connection with retries
	for attempt := 1; attempt <= 5; attempt++ {
		ctx, cancel := context.WithTimeout(c.ctx, c.config.RedisTimeout)
		err := c.redisClient.Ping(ctx).Err()
		cancel()

		if err == nil {
			c.log.Info("Redis connection established")
			return nil
		}

		c.log.Warnf("Redis connection attempt %d/5 failed: %v", attempt, err)
		if attempt < 5 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	return fmt.Errorf("failed to connect to Redis after 5 attempts. Please ensure Redis is running on %s", c.config.RedisAddr)
}

// initializeKafka sets up Kafka producer with proper configuration
func (c *Crawler) initializeKafka() error {
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	kafkaConfig.Producer.Retry.Max = c.config.KafkaRetryMax
	kafkaConfig.Producer.Retry.Backoff = 100 * time.Millisecond
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Return.Errors = true
	kafkaConfig.Producer.Timeout = c.config.RequestTimeout
	kafkaConfig.Net.DialTimeout = c.config.RequestTimeout
	kafkaConfig.Metadata.RefreshFrequency = 10 * time.Minute

	brokers := strings.Split(c.config.KafkaBrokers, ",")
	producer, err := sarama.NewSyncProducer(brokers, kafkaConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	c.producer = producer
	c.log.Info("Kafka producer initialized")
	return nil
}

func (c *Crawler) isValidURL(rawURL string) bool {
	if rawURL == "" {
		return false
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}

	if parsedURL.Host == "" {
		return false
	}

	host := strings.ToLower(parsedURL.Hostname())

	// Resolve DNS for domain names
	ips, err := net.LookupIP(host)
	if err == nil {
		for _, ip := range ips {
			if ip.IsLoopback() || ip.IsPrivate() {
				return false
			}
		}
	}

	// Check blacklist domains
	for _, blocked := range c.blacklist {
		blocked = strings.ToLower(blocked)
		if strings.HasSuffix(host, "."+blocked) || host == blocked {
			return false
		}
	}

	// Whitelist domains if set
	if len(c.whitelist) > 0 {
		allowed := false
		for _, allowedDomain := range c.whitelist {
			allowedDomain = strings.ToLower(allowedDomain)
			if strings.HasSuffix(host, "."+allowedDomain) || host == allowedDomain {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	return true
}

func normalizeURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
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

func (c *Crawler) isURLVisited(url string) (bool, error) {
	// Local memory cache to avoid repeated Redis hits
	if _, ok := c.visited.Load(url); ok {
		return true, nil
	}

	for attempt := 0; attempt < c.config.RedisRetryMax; attempt++ {
		ctx, cancel := context.WithTimeout(c.ctx, c.config.RedisTimeout)
		added, err := c.redisClient.SAdd(ctx, "visited_urls", url).Result()
		cancel()

		if err == nil {
			c.visited.Store(url, struct{}{})
			if added == 1 {
				c.metrics.IncrementRedisSuccessful()
				return false, nil
			}
			c.metrics.IncrementRedisSuccessful()
			return true, nil
		}

		c.log.Warnf("Redis attempt %d failed for URL %s: %v", attempt+1, url, err)
		if attempt < c.config.RedisRetryMax-1 {
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	c.metrics.IncrementRedisFailed()
	return false, fmt.Errorf("failed to check URL in Redis after %d attempts", c.config.RedisRetryMax)
}

func (c *Crawler) sendToKafka(url, html string) error {
	if int64(len(html)) > c.config.MaxContentSize {
		return &CrawlError{
			URL:       url,
			Operation: "kafka_send",
			Err:       fmt.Errorf("content size %d exceeds limit %d", len(html), c.config.MaxContentSize),
			Retry:     false,
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

		c.log.Warnf("Kafka attempt %d failed for URL %s: %v", attempt+1, url, err)
		if attempt < c.config.KafkaRetryMax-1 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	c.metrics.IncrementKafkaFailed()
	return &CrawlError{
		URL:       url,
		Operation: "kafka_send",
		Err:       fmt.Errorf("failed to send to Kafka after %d attempts", c.config.KafkaRetryMax),
		Retry:     true,
	}
}

// setupCollyCollector configures the colly collector
func (c *Crawler) setupCollyCollector() *colly.Collector {
	options := []colly.CollectorOption{
		colly.MaxDepth(c.config.CrawlDepth),
		colly.Async(true),
		colly.UserAgent(c.config.UserAgent),
	}

	if c.config.EnableDebug {
		options = append(options, colly.Debugger(&debug.LogDebugger{}))
	}

	collector := colly.NewCollector(options...)

	// Rate limits
	collector.Limit(&colly.LimitRule{
		DomainGlob:  "*wikipedia.org*",
		Parallelism: c.config.MaxConcurrency * 2,
		Delay:       50 * time.Millisecond,
	})

	collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: c.config.MaxConcurrency,
		Delay:       c.config.RequestDelay,
	})

	collector.SetRequestTimeout(c.config.RequestTimeout)

	// Request filtering
	collector.OnRequest(func(r *colly.Request) {
		select {
		case <-c.ctx.Done():
			r.Abort()
			return
		default:
		}

		urlStr := r.URL.String()
		skipExts := []string{".pdf", ".jpg", ".jpeg", ".png", ".gif", ".css", ".js", ".ico", ".svg", ".woff", ".ttf", ".mp4", ".mp3", ".zip", ".exe"}
		for _, ext := range skipExts {
			if strings.HasSuffix(strings.ToLower(urlStr), ext) {
				r.Abort()
				return
			}
		}

		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Accept-Encoding", "gzip, deflate")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
	})

	// Handle HTML pages
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		if atomic.LoadInt64(&c.metrics.PagesProcessed) >= c.config.MaxPages {
			return
		}

		c.metrics.IncrementPagesProcessed()
		url := e.Request.URL.String()

		go func() {
			html, err := e.DOM.Html()
			if err != nil {
				c.log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to extract HTML")
				c.metrics.IncrementPagesFailed()
				return
			}

			if err := c.sendToKafka(url, html); err != nil {
				c.log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to send to Kafka")
				c.metrics.IncrementPagesFailed()
				return
			}

			c.metrics.IncrementPagesSuccessful()
			c.log.WithFields(logrus.Fields{"url": url, "content_size": len(html)}).Debug("Page processed successfully")
		}()
	})

	// Deduplicate links on the page
	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		if atomic.LoadInt64(&c.metrics.PagesProcessed) >= c.config.MaxPages {
			return
		}

		// Use local map to deduplicate URLs per page
		var seenLinks = make(map[string]struct{})

		link := e.Attr("href")
		if link == "" {
			return
		}

		if strings.HasPrefix(link, "javascript:") ||
			strings.HasPrefix(link, "mailto:") ||
			strings.HasPrefix(link, "tel:") ||
			strings.Contains(link, "#") {
			return
		}

		absoluteURL := e.Request.AbsoluteURL(link)
		if !c.isValidURL(absoluteURL) {
			return
		}

		normalized, err := normalizeURL(absoluteURL)
		if err != nil {
			return
		}

		if _, exists := seenLinks[normalized]; exists {
			return
		}
		seenLinks[normalized] = struct{}{}

		go func(normURL string) {
			visited, err := c.isURLVisited(normURL)
			if err != nil {
				c.metrics.IncrementRedisFailed()
				return
			}

			if !visited {
				if err := e.Request.Visit(normURL); err != nil {
					if c.config.EnableDebug {
						c.log.WithFields(logrus.Fields{"url": normURL, "error": err}).Debug("Failed to visit URL")
					}
				}
			}
		}(normalized)
	})

	// Error handler
	collector.OnError(func(r *colly.Response, err error) {
		if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "connection refused") {
			c.log.WithFields(logrus.Fields{
				"url":         r.Request.URL.String(),
				"status_code": r.StatusCode,
				"error":       err,
			}).Warn("Request failed")
		}
		c.metrics.IncrementPagesFailed()
	})

	// Debug logging on response
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

// startHealthCheck starts HTTP health check server
func (c *Crawler) startHealthCheck() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
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
			stats["uptime_seconds"],
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
		<-c.shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			c.log.Errorf("Health check server shutdown error: %v", err)
		}
	}()
}

// logMetricsPeriodically logs metrics every 30 seconds
func (c *Crawler) logMetricsPeriodically() {
	ticker := time.NewTicker(30 * time.Second) // Reduced from 1 minute
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stats := c.metrics.GetStats()
				pagesPerSecond := float64(stats["pages_processed"].(int64)) / stats["uptime_seconds"].(float64)
				c.log.WithFields(logrus.Fields{
					"pages_processed":  stats["pages_processed"],
					"pages_successful": stats["pages_successful"],
					"pages_failed":     stats["pages_failed"],
					"kafka_successful": stats["kafka_successful"],
					"kafka_failed":     stats["kafka_failed"],
					"redis_successful": stats["redis_successful"],
					"redis_failed":     stats["redis_failed"],
					"uptime_seconds":   stats["uptime_seconds"],
					"pages_per_second": fmt.Sprintf("%.2f", pagesPerSecond),
				}).Info("Crawler metrics")
			case <-c.shutdown:
				return
			}
		}
	}()
}

// Start begins the crawling process
func (c *Crawler) Start() error {
	c.log.Info("Starting web crawler")
	c.log.WithFields(logrus.Fields{
		"kafka_brokers": c.config.KafkaBrokers,
		"start_urls":    c.config.StartURLs,
		"max_pages":     c.config.MaxPages,
		"max_depth":     c.config.CrawlDepth,
		"concurrency":   c.config.MaxConcurrency,
	}).Info("Crawler configuration")

	// Start health check server
	c.startHealthCheck()

	// Start metrics logging
	c.logMetricsPeriodically()

	// Setup crawler
	collector := c.setupCollyCollector()

	// Start crawling from configured URLs
	startURLs := strings.Split(c.config.StartURLs, ",")
	for _, rawURL := range startURLs {
		url := strings.TrimSpace(rawURL)
		if !c.isValidURL(url) {
			c.log.Warnf("Skipping invalid start URL: %s", url)
			continue
		}

		if err := collector.Visit(url); err != nil {
			c.log.WithFields(logrus.Fields{
				"url":   url,
				"error": err,
			}).Error("Failed to visit start URL")
		}
	}

	// Wait for all crawling to complete
	collector.Wait()

	stats := c.metrics.GetStats()
	c.log.WithFields(logrus.Fields{
		"pages_processed":  stats["pages_processed"],
		"pages_successful": stats["pages_successful"],
		"pages_failed":     stats["pages_failed"],
		"duration_seconds": stats["uptime_seconds"],
	}).Info("Crawling completed")

	return nil
}

// Shutdown gracefully shuts down the crawler
func (c *Crawler) Shutdown() {
	c.shutdownOnce.Do(func() {
		c.log.Info("Shutting down crawler...")
		close(c.shutdown)
		c.cancel()

		// Close connections
		if c.producer != nil {
			if err := c.producer.Close(); err != nil {
				c.log.Errorf("Failed to close Kafka producer: %v", err)
			}
		}

		if c.redisClient != nil {
			if err := c.redisClient.Close(); err != nil {
				c.log.Errorf("Failed to close Redis client: %v", err)
			}
		}

		// Wait for goroutines to finish
		done := make(chan struct{})
		go func() {
			c.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			c.log.Info("All goroutines finished")
		case <-time.After(10 * time.Second):
			c.log.Warn("Timeout waiting for goroutines to finish")
		}

		c.log.Info("Crawler shutdown complete")
	})
}

func main() {
	crawler, err := NewCrawler()
	if err != nil {
		logrus.Fatalf("Failed to create crawler: %v", err)
	}

	// OS signal channel
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Internal done channel
	done := make(chan struct{})

	// Start crawler in goroutine
	go func() {
		if err := crawler.Start(); err != nil {
			logrus.Errorf("Crawler error: %v", err)
		}
		close(done) // signal completion internally
	}()

	select {
	case <-sigChan:
		logrus.Info("Received OS shutdown signal")
	case <-done:
		logrus.Info("Crawler completed naturally")
	}

	crawler.Shutdown()
}
