//! Configuration for the parser service.
//!
//! Defines the `Config` struct and its defaults, which control Kafka topics,
//! number of workers, logging, content size constraints, etc.
//!
//! Can be deserialized from a config file or environment variables using `serde`.

use serde::Deserialize;
use tracing_subscriber::EnvFilter;

/// Application configuration.
///
/// Fields can be loaded from a config file or environment variables using `serde`,
/// and all fields have reasonable defaults if not provided.
///
/// Example:
/// ```toml
/// kafka_brokers = "localhost:9092"
/// max_concurrency = 8
/// ```
///
#[derive(Debug, Deserialize, Clone)]
pub struct Config {
    /// Kafka bootstrap brokers, e.g., `localhost:9092`.
    #[serde(default = "default_kafka_brokers")]
    pub kafka_brokers: String,

    /// Kafka topic to consume raw HTML pages from.
    #[serde(default = "default_kafka_topic_html")]
    pub kafka_topic_html: String,

    /// Kafka topic to publish parsed pages to.
    #[serde(default = "default_kafka_topic_parsed")]
    pub kafka_topic_parsed: String,

    /// Kafka consumer group ID.
    #[serde(default = "default_kafka_group_id")]
    pub kafka_group_id: String,

    /// Number of workers.
    #[serde(default = "default_max_concurrency")]
    pub max_concurrency: usize,

    /// Maximum allowed content length (in bytes) for a page.
    #[serde(default = "default_max_content_length")]
    pub max_content_length: usize,

    /// Minimum allowed content length (in bytes) for a page.
    #[serde(default = "default_min_content_length")]
    pub min_content_length: usize,

    /// Logging level, e.g., `info`, `debug`, `warn`.
    #[serde(default = "default_log_level")]
    pub rust_log: String,

    /// Port to expose monitoring/health endpoints on.
    #[serde(default = "default_monitor_port")]
    pub monitor_port: u16,
}

/// Provides default values for `Config`.
impl Default for Config {
    fn default() -> Self {
        Self {
            kafka_brokers: default_kafka_brokers(),
            kafka_topic_html: default_kafka_topic_html(),
            kafka_topic_parsed: default_kafka_topic_parsed(),
            kafka_group_id: default_kafka_group_id(),
            max_concurrency: default_max_concurrency(),
            max_content_length: default_max_content_length(),
            min_content_length: default_min_content_length(),
            rust_log: default_log_level(),
            monitor_port: default_monitor_port(),
        }
    }
}

impl Config {
    /// Initialize logging using the configured log level.
    ///
    /// Sets the `RUST_LOG` environment variable and initializes `tracing_subscriber`
    /// with environment filters.
    pub fn init_logging(&self) {
        std::env::set_var("RUST_LOG", &self.rust_log);
        tracing_subscriber::fmt()
            .with_env_filter(EnvFilter::from_default_env())
            .init();
    }
}

// Below are the default value functions for each field.
// They are also used by serde to supply defaults when a field is missing.

fn default_kafka_brokers() -> String {
    "kafka:9092".to_string()
}

fn default_kafka_topic_html() -> String {
    "raw-html".to_string()
}

fn default_kafka_topic_parsed() -> String {
    "parsed-pages".to_string()
}

fn default_kafka_group_id() -> String {
    "parser-group".to_string()
}

fn default_max_concurrency() -> usize {
    64
}

fn default_max_content_length() -> usize {
    5_242_880
}

fn default_min_content_length() -> usize {
    1024
}

fn default_log_level() -> String {
    "info".to_string()
}

fn default_monitor_port() -> u16 {
    8080
}
