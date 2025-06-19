package config

import (
	// Stdlib
	"fmt"
	"time"

	// Third-party
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
	RedisTimeout  time.Duration `envconfig:"REDIS_TIMEOUT" default:"15s"`
	RedisRetryMax int           `envconfig:"REDIS_RETRY_MAX" default:"3"`

	// Crawling Behavior - Core crawling parameters and URL management
	StartURLs    string `envconfig:"START_URLS" default:"https://en.wikipedia.org/wiki/Special:Random,https://simple.wikipedia.org/wiki/Special:Random,https://news.ycombinator.com,https://www.reuters.com/news/archive/worldNews,https://www.bbc.com/news,https://github.com/trending,https://stackoverflow.com/questions,https://dev.to,https://developer.mozilla.org/en-US/docs/Web,https://arxiv.org/list/cs/new,https://eng.uber.com,https://netflixtechblog.com,https://blog.cloudflare.com"`
	CrawlDepth   int    `envconfig:"CRAWL_DEPTH" default:"3"`
	MaxPages     int64  `envconfig:"MAX_PAGES" default:"1000"`
	URLWhitelist string `envconfig:"URL_WHITELIST" default:""`
	URLBlacklist string `envconfig:"URL_BLACKLIST" default:""`

	// Performance & Limits - Resource management and rate limiting
	MaxConcurrency int           `envconfig:"MAX_CONCURRENCY" default:"32"`
	RequestTimeout time.Duration `envconfig:"REQUEST_TIMEOUT" default:"15s"`
	RequestDelay   time.Duration `envconfig:"REQUEST_DELAY" default:"50ms"`
	MaxContentSize int           `envconfig:"MAX_CONTENT_SIZE" default:"2621440"` // 2.5MB default

	// Application Settings - Logging, monitoring, and operational parameters
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`
	UserAgent   string `envconfig:"USER_AGENT" default:"Sneakdex/1.0"`
	EnableDebug bool   `envconfig:"ENABLE_DEBUG" default:"false"`
	MonitorPort int    `envconfig:"MONITOR_PORT" default:"8080"`
}

// InitializeConfig loads configuration from environment variables and validates all settings.
func InitializeConfig() (*Config, error) {
	cfg := &Config{}

	// Process environment variables into the config struct
	if err := envconfig.Process("", cfg); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}

	// Validate all configuration values
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}
