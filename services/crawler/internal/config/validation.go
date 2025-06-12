package config

import "fmt"

func (config *Config) validateKafka() error {
	if config.KafkaBrokers == "" {
		return fmt.Errorf("kafka_brokers must be set")
	}
	if config.KafkaTopic == "" {
		return fmt.Errorf("kafka_topic must be set")
	}
	if config.KafkaRetryMax <= 0 {
		return fmt.Errorf("kafka_retry_max must be positive")
	}
	return nil
}

func (config *Config) validateRedis() error {
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
	return nil
}

func (config *Config) validateCrawling() error {
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
	return nil
}

func (config *Config) validatePerformance() error {
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
	return nil
}

func (config *Config) validateApplication() error {
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
