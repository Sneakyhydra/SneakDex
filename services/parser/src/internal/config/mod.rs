//! Configuration for the parser service.

mod validation;

use serde::Deserialize;
use tracing_subscriber::EnvFilter;

pub use validation::{ConfigError, Validate};

#[derive(Debug, Deserialize, Clone)]
pub struct Config {
    #[serde(default = "default_kafka_brokers")]
    pub kafka_brokers: String,
    #[serde(default = "default_kafka_topic_html")]
    pub kafka_topic_html: String,
    #[serde(default = "default_kafka_topic_parsed")]
    pub kafka_topic_parsed: String,
    #[serde(default = "default_kafka_group_id")]
    pub kafka_group_id: String,
    #[serde(default = "default_max_concurrency")]
    pub max_concurrency: usize,
    #[serde(default = "default_max_content_length")]
    pub max_content_length: usize,
    #[serde(default = "default_min_content_length")]
    pub min_content_length: usize,
    #[serde(default = "default_log_level")]
    pub rust_log: String,
    #[serde(default = "default_monitor_port")]
    pub monitor_port: u16,
}

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
    pub fn init_logging(&self) {
        std::env::set_var("RUST_LOG", &self.rust_log);
        tracing_subscriber::fmt()
            .with_env_filter(EnvFilter::from_default_env())
            .init();
    }

    pub fn validate(&self) -> Result<(), ConfigError> {
        Validate::validate(self)
    }
}

// defaults
fn default_kafka_brokers() -> String {
    "kafka:9092".into()
}
fn default_kafka_topic_html() -> String {
    "raw-html".into()
}
fn default_kafka_topic_parsed() -> String {
    "parsed-pages".into()
}
fn default_kafka_group_id() -> String {
    "parser-group".into()
}
fn default_max_concurrency() -> usize {
    32
}
fn default_max_content_length() -> usize {
    5_242_880
}
fn default_min_content_length() -> usize {
    1
}
fn default_log_level() -> String {
    "info".into()
}
fn default_monitor_port() -> u16 {
    8080
}
