package config

import (
	// StdLib
	"fmt"
	"time"

	// Third-party
	"github.com/kelseyhightower/envconfig"
)

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
	validators := []func() error{
		config.validateKafka,
		config.validateRedis,
		config.validateCrawling,
		config.validatePerformance,
		config.validateApplication,
	}

	for _, validate := range validators {
		if err := validate(); err != nil {
			return err
		}
	}

	return nil
}

var cfg *Config

// InitializeConfig loads and validates configuration from environment variables
func InitializeConfig() error {
	newcfg := &Config{}
	if err := envconfig.Process("", newcfg); err != nil {
		return fmt.Errorf("{ failed to process environment variables: %w }", err)
	}
	if err := newcfg.Validate(); err != nil {
		return fmt.Errorf("{ invalid configuration: %w }", err)
	}
	cfg = newcfg
	return nil
}

func GetConfig() *Config {
	return cfg
}
