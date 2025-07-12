//! Entry point for the parser service.
//!
//! This binary initializes configuration, logging, Kafka client, and HTML parser,
//! and starts processing Kafka messages asynchronously.

use anyhow::Result;
use std::sync::Arc;
use tracing::error;

mod internal;

use internal::config::Config;
use internal::core::KafkaHandler;
use internal::monitor::{start_monitor_server, Metrics};
use internal::parser::HtmlParser;

/// Initializes and runs the parser service.
///
/// Loads configuration from environment variables (with defaults),
/// sets up logging, initializes Kafka consumer/producer, and starts
/// processing messages.
async fn run() -> Result<()> {
    // Load .env file if it exists (for local development)
    // This will be ignored in Docker if env vars are already set
    dotenv::dotenv().ok();

    // Load config from environment; fall back to defaults if missing.
    let config: Arc<Config> = Arc::new(envy::from_env().unwrap_or_default());
    config.init_logging();

    // Initialize Kafka handler and HTML parser.
    let kafka_handler = Arc::new(KafkaHandler::new(Arc::clone(&config)).await?);
    let parser = HtmlParser::new(&config);

    // Initialize metrics
    let metrics = Metrics::new();

    // Start monitor server in background
    let monitor_port = config.monitor_port;
    let metrics_clone = metrics.clone(); // Members are wrapped in Arc
    let kafka_clone = kafka_handler.clone();
    std::thread::spawn(move || {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            if let Err(e) = start_monitor_server(monitor_port, metrics_clone, kafka_clone).await {
                error!("Monitor server failed: {}", e);
            }
        });
    });

    // Start consuming & processing messages.
    kafka_handler.start_processing(parser, metrics).await
}

/// Main function — entry point of the binary.
///
/// Runs the service inside a Tokio async runtime and logs any fatal errors.
#[tokio::main]
async fn main() -> Result<()> {
    match run().await {
        Ok(_) => Ok(()),
        Err(e) => {
            error!("Parser service error: {}", e);
            Err(e)
        }
    }
}
