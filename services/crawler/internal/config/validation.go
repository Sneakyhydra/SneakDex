package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ConfigError represents configuration-specific errors with context
type ConfigError struct {
	Field   string
	Value   string
	Reason  string
	Example string
}

func (e *ConfigError) Error() string {
	msg := fmt.Sprintf("invalid configuration for field '%s': %s", e.Field, e.Reason)
	if e.Value != "" {
		msg += fmt.Sprintf(" (got: '%s')", e.Value)
	}
	if e.Example != "" {
		msg += fmt.Sprintf(" (example: %s)", e.Example)
	}
	return msg
}

// Validate performs comprehensive validation of all configuration fields.
// Returns the first validation error encountered, wrapped in a ConfigError.
func (c *Config) Validate() error {
	// Define validation functions for each configuration category
	validators := []func() error{
		c.validateKafka,
		c.validateRedis,
		c.validateCrawling,
		c.validatePerformance,
		c.validateApplication,
	}

	// Execute all validators and return the first error
	for _, validate := range validators {
		if err := validate(); err != nil {
			return err
		}
	}

	return nil
}

// validateKafka ensures Kafka configuration is valid and complete
func (c *Config) validateKafka() error {
	if strings.TrimSpace(c.KafkaBrokers) == "" {
		return &ConfigError{
			Field:   "KAFKA_BROKERS",
			Reason:  "cannot be empty",
			Example: "localhost:9092",
		}
	}

	if strings.TrimSpace(c.KafkaTopic) == "" {
		return &ConfigError{
			Field:   "KAFKA_TOPIC_HTML",
			Reason:  "cannot be empty",
			Example: "raw-html",
		}
	}

	// Validate topic name follows Kafka naming conventions
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9._-]+$`, c.KafkaTopic); !matched {
		return &ConfigError{
			Field:   "KAFKA_TOPIC_HTML",
			Value:   c.KafkaTopic,
			Reason:  "must contain only alphanumeric characters, dots, hyphens, and underscores",
			Example: "raw-html-data",
		}
	}

	if c.KafkaRetryMax < 0 || c.KafkaRetryMax > 10 {
		return &ConfigError{
			Field:   "KAFKA_RETRY_MAX",
			Value:   fmt.Sprintf("%d", c.KafkaRetryMax),
			Reason:  "must be between 0 and 10",
			Example: "3",
		}
	}

	return nil
}

// validateRedis ensures Redis configuration is valid and complete
func (c *Config) validateRedis() error {
	if strings.TrimSpace(c.RedisHost) == "" {
		return &ConfigError{
			Field:   "REDIS_HOST",
			Reason:  "cannot be empty",
			Example: "localhost",
		}
	}

	if c.RedisPort < 1 || c.RedisPort > 65535 {
		return &ConfigError{
			Field:   "REDIS_PORT",
			Value:   fmt.Sprintf("%d", c.RedisPort),
			Reason:  "must be a valid port number (1-65535)",
			Example: "6379",
		}
	}

	if c.RedisDB < 0 || c.RedisDB > 15 {
		return &ConfigError{
			Field:   "REDIS_DB",
			Value:   fmt.Sprintf("%d", c.RedisDB),
			Reason:  "must be between 0 and 15 (Redis database index)",
			Example: "0",
		}
	}

	if c.RedisTimeout < time.Second || c.RedisTimeout > 10*time.Minute {
		return &ConfigError{
			Field:   "REDIS_TIMEOUT",
			Value:   c.RedisTimeout.String(),
			Reason:  "must be between 1s and 10m",
			Example: "30s",
		}
	}

	if c.RedisRetryMax < 0 || c.RedisRetryMax > 10 {
		return &ConfigError{
			Field:   "REDIS_RETRY_MAX",
			Value:   fmt.Sprintf("%d", c.RedisRetryMax),
			Reason:  "must be between 0 and 10",
			Example: "3",
		}
	}

	return nil
}

// validateCrawling ensures crawling behavior configuration is valid
func (c *Config) validateCrawling() error {
	if strings.TrimSpace(c.StartURLs) == "" {
		return &ConfigError{
			Field:   "START_URLS",
			Reason:  "cannot be empty - at least one starting URL is required",
			Example: "https://example.com,https://example.org",
		}
	}

	// Validate that all start URLs are properly formatted
	urls := strings.Split(c.StartURLs, ",")
	for i, rawURL := range urls {
		trimmedURL := strings.TrimSpace(rawURL)
		if trimmedURL == "" {
			continue // Skip empty URLs from splitting
		}

		if _, err := url.ParseRequestURI(trimmedURL); err != nil {
			return &ConfigError{
				Field:   "START_URLS",
				Value:   trimmedURL,
				Reason:  fmt.Sprintf("URL #%d is malformed: %v", i+1, err),
				Example: "https://example.com/path",
			}
		}
	}

	if c.CrawlDepth < 1 || c.CrawlDepth > 20 {
		return &ConfigError{
			Field:   "CRAWL_DEPTH",
			Value:   fmt.Sprintf("%d", c.CrawlDepth),
			Reason:  "must be between 1 and 20 to prevent infinite crawling",
			Example: "3",
		}
	}

	if c.MaxPages < 1 || c.MaxPages > 1000000 {
		return &ConfigError{
			Field:   "MAX_PAGES",
			Value:   fmt.Sprintf("%d", c.MaxPages),
			Reason:  "must be between 1 and 1,000,000",
			Example: "10000",
		}
	}

	return nil
}

// validatePerformance ensures performance settings are within reasonable bounds
func (c *Config) validatePerformance() error {
	if c.MaxConcurrency < 1 || c.MaxConcurrency > 1000 {
		return &ConfigError{
			Field:   "MAX_CONCURRENCY",
			Value:   fmt.Sprintf("%d", c.MaxConcurrency),
			Reason:  "must be between 1 and 1000 to prevent resource exhaustion",
			Example: "16",
		}
	}

	if c.RequestTimeout < time.Second || c.RequestTimeout > 5*time.Minute {
		return &ConfigError{
			Field:   "REQUEST_TIMEOUT",
			Value:   c.RequestTimeout.String(),
			Reason:  "must be between 1s and 5m",
			Example: "30s",
		}
	}

	if c.RequestDelay < 0 || c.RequestDelay > 30*time.Second {
		return &ConfigError{
			Field:   "REQUEST_DELAY",
			Value:   c.RequestDelay.String(),
			Reason:  "must be between 0 and 30s",
			Example: "100ms",
		}
	}

	// MaxContentSize validation (1KB to 100MB)
	if c.MaxContentSize < 1024 || c.MaxContentSize > 100*1024*1024 {
		return &ConfigError{
			Field:   "MAX_CONTENT_SIZE",
			Value:   fmt.Sprintf("%d bytes", c.MaxContentSize),
			Reason:  "must be between 1KB and 100MB",
			Example: "2621440 (2.5MB)",
		}
	}

	return nil
}

// validateApplication ensures application-level settings are valid
func (c *Config) validateApplication() error {
	validLogLevels := map[string]bool{
		"trace": true, "debug": true, "info": true,
		"warn": true, "error": true, "fatal": true, "panic": true,
	}

	logLevel := strings.ToLower(strings.TrimSpace(c.LogLevel))
	if !validLogLevels[logLevel] {
		return &ConfigError{
			Field:   "LOG_LEVEL",
			Value:   c.LogLevel,
			Reason:  "must be one of: trace, debug, info, warn, error, fatal, panic",
			Example: "info",
		}
	}

	if strings.TrimSpace(c.UserAgent) == "" {
		return &ConfigError{
			Field:   "USER_AGENT",
			Reason:  "cannot be empty for responsible web crawling",
			Example: "SneakdexCrawler/1.0",
		}
	}

	if c.HealthCheckPort < 1024 || c.HealthCheckPort > 65535 {
		return &ConfigError{
			Field:   "HEALTH_CHECK_PORT",
			Value:   fmt.Sprintf("%d", c.HealthCheckPort),
			Reason:  "must be between 1024 and 65535 (avoid privileged ports)",
			Example: "8080",
		}
	}

	return nil
}
