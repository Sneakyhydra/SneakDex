package config

import (
	"fmt"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds the crawler configuration loaded from environment variables.
type Config struct {
	// Kafka Configuration - Message queue settings for sending crawled content to parser
	KafkaBrokers  string `envconfig:"KAFKA_BROKERS" default:"kafka:9092"`
	KafkaTopic    string `envconfig:"KAFKA_TOPIC_HTML" default:"raw-html"`
	KafkaRetryMax int    `envconfig:"KAFKA_RETRY_MAX" default:"3"`

	// Redis Configuration - Cache and queue management settings
	RedisHost     string        `envconfig:"REDIS_HOST" default:"redis"`
	RedisPort     int           `envconfig:"REDIS_PORT" default:"6379"`
	RedisPassword string        `envconfig:"REDIS_PASSWORD" default:""`
	RedisDB       int           `envconfig:"REDIS_DB" default:"0"`
	RedisTimeout  time.Duration `envconfig:"REDIS_TIMEOUT" default:"60s"`
	RedisRetryMax int           `envconfig:"REDIS_RETRY_MAX" default:"3"`

	// Crawling Behavior - Core crawling parameters and URL management
	StartURLs    string `envconfig:"START_URLS" default:"https://en.wikipedia.org/wiki/Special:Random,https://simple.wikipedia.org/wiki/Special:Random,https://news.ycombinator.com,https://www.reuters.com/news/archive/worldNews,https://www.bbc.com/news,https://github.com/trending,https://stackoverflow.com/questions,https://dev.to,https://developer.mozilla.org/en-US/docs/Web,https://arxiv.org/list/cs/new,https://eng.uber.com,https://netflixtechblog.com,https://blog.cloudflare.com"`
	CrawlDepth   int    `envconfig:"CRAWL_DEPTH" default:"3"`
	MaxPages     int64  `envconfig:"MAX_PAGES" default:"10000"`
	URLWhitelist string `envconfig:"URL_WHITELIST" default:""`
	URLBlacklist string `envconfig:"URL_BLACKLIST" default:""`

	// Performance & Limits - Resource management and rate limiting
	MaxConcurrency int           `envconfig:"MAX_CONCURRENCY" default:"8"`
	RequestTimeout time.Duration `envconfig:"REQUEST_TIMEOUT" default:"30s"`
	RequestDelay   time.Duration `envconfig:"REQUEST_DELAY" default:"100ms"`
	MaxContentSize int           `envconfig:"MAX_CONTENT_SIZE" default:"2621440"` // 2.5MB default

	// Application Settings - Logging, monitoring, and operational parameters
	LogLevel        string `envconfig:"LOG_LEVEL" default:"info"`
	UserAgent       string `envconfig:"USER_AGENT" default:"SneakdexCrawler/1.0"`
	EnableDebug     bool   `envconfig:"ENABLE_DEBUG" default:"false"`
	HealthCheckPort int    `envconfig:"HEALTH_CHECK_PORT" default:"8080"`
}

// Global configuration instance - initialized once during application startup
var (
	cfg      *Config
	initOnce sync.Once
)

// InitializeConfig loads configuration from environment variables and validates all settings.
// This function should be called exactly once during application initialization.
// Returns a ConfigError if any validation fails, or a generic error for processing failures.
func InitializeConfig() error {
	var initErr error
	initOnce.Do(func() {
		newCfg := &Config{}

		// Process environment variables into the config struct
		if err := envconfig.Process("", newCfg); err != nil {
			initErr = fmt.Errorf("failed to process environment variables: %w", err)
			return
		}

		// Validate all configuration values
		if err := newCfg.Validate(); err != nil {
			initErr = fmt.Errorf("configuration validation failed: %w", err)
			return
		}

		// Assign the new configuration to the global variable
		cfg = newCfg
	})
	return initErr
}

// GetConfig returns the initialized configuration instance. Panics if not initialized.
func GetConfig() *Config {
	if cfg == nil {
		panic("configuration is nil: InitializeConfig() must be called before accessing config. Check main.go startup sequence.")
	}
	return cfg
}
