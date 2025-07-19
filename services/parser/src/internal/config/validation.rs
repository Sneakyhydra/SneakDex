use super::Config;
use std::fmt;

#[derive(Debug)]
pub struct ConfigError {
    pub field: &'static str,
    pub value: String,
    pub reason: &'static str,
    pub example: &'static str,
}

impl std::error::Error for ConfigError {}

impl fmt::Display for ConfigError {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(
            f,
            "Invalid config: field='{}', reason='{}', got='{}', example='{}'",
            self.field, self.reason, self.value, self.example
        )
    }
}

/// Trait so you can call `config.validate()` cleanly
pub trait Validate {
    fn validate(&self) -> Result<(), ConfigError>;
}

impl Validate for Config {
    fn validate(&self) -> Result<(), ConfigError> {
        self.validate_kafka()?;
        self.validate_concurrency()?;
        self.validate_content_length()?;
        self.validate_log_level()?;
        self.validate_monitor_port()?;
        Ok(())
    }
}

impl Config {
    fn validate_kafka(&self) -> Result<(), ConfigError> {
        if self.kafka_brokers.trim().is_empty() {
            return Err(ConfigError {
                field: "kafka_brokers",
                value: self.kafka_brokers.clone(),
                reason: "cannot be empty",
                example: "localhost:9092",
            });
        }
        if self.kafka_topic_html.trim().is_empty() {
            return Err(ConfigError {
                field: "kafka_topic_html",
                value: self.kafka_topic_html.clone(),
                reason: "cannot be empty",
                example: "raw-html",
            });
        }
        if self.kafka_topic_parsed.trim().is_empty() {
            return Err(ConfigError {
                field: "kafka_topic_parsed",
                value: self.kafka_topic_parsed.clone(),
                reason: "cannot be empty",
                example: "parsed-pages",
            });
        }
        if self.kafka_group_id.trim().is_empty() {
            return Err(ConfigError {
                field: "kafka_group_id",
                value: self.kafka_group_id.clone(),
                reason: "cannot be empty",
                example: "parser-group",
            });
        }
        Ok(())
    }

    fn validate_concurrency(&self) -> Result<(), ConfigError> {
        if self.max_concurrency == 0 || self.max_concurrency > 1024 {
            return Err(ConfigError {
                field: "max_concurrency",
                value: self.max_concurrency.to_string(),
                reason: "must be between 1 and 1024",
                example: "8",
            });
        }
        Ok(())
    }

    fn validate_content_length(&self) -> Result<(), ConfigError> {
        if self.min_content_length < 0 {
            return Err(ConfigError {
                field: "min_content_length",
                value: self.min_content_length.to_string(),
                reason: "must be at least 0 bytes",
                example: "50",
            });
        }

        if self.max_content_length <= self.min_content_length {
            return Err(ConfigError {
                field: "max_content_length",
                value: self.max_content_length.to_string(),
                reason: "must be greater than min_content_length",
                example: "5242880",
            });
        }

        Ok(())
    }

    fn validate_log_level(&self) -> Result<(), ConfigError> {
        let valid_levels = ["trace", "debug", "info", "warn", "error"];
        if !valid_levels.contains(&self.rust_log.as_str()) {
            return Err(ConfigError {
                field: "rust_log",
                value: self.rust_log.clone(),
                reason: "must be one of: trace, debug, info, warn, error",
                example: "info",
            });
        }
        Ok(())
    }

    fn validate_monitor_port(&self) -> Result<(), ConfigError> {
        if self.monitor_port <= 0 {
            return Err(ConfigError {
                field: "monitor_port",
                value: self.monitor_port.to_string(),
                reason: "must be between 1 and 65535",
                example: "8080",
            });
        }
        Ok(())
    }
}
