//! Entry point for the SneakDex parser service.
//!
//! Initializes configuration, logging, Kafka client, and HTML parser,
//! starts processing Kafka messages asynchronously, and supports graceful shutdown.

use anyhow::Result;
use std::{sync::Arc, time::Duration};
use tokio::{select, signal, task::JoinHandle, time};
use tracing::{error, info};

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
    if let Err(err) = config.validate() {
        eprintln!("Configuration error: {}", err);
        std::process::exit(1);
    }

    // Initialize Kafka handler and HTML parser.
    let kafka_handler = Arc::new(KafkaHandler::new(Arc::clone(&config)).await?);
    let parser = HtmlParser::new(&config);
    // Initialize metrics
    let metrics = Metrics::new();

    // Start monitor server
    let monitor_port = config.monitor_port;
    let metrics_clone = metrics.clone();
    let kafka_clone = kafka_handler.clone();

    std::thread::spawn(move || {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            if let Err(e) = start_monitor_server(monitor_port, metrics_clone, kafka_clone).await {
                error!("Monitor server failed: {}", e);
            }
        });
    });

    // Shutdown signal notifier
    let (shutdown_tx, shutdown_rx) = tokio::sync::watch::channel(false);

    // Kafka processing task
    let kafka_task: JoinHandle<()> = tokio::spawn({
        let shutdown_rx = shutdown_rx.clone();
        async move {
            kafka_handler
                .start_processing(parser, metrics, shutdown_rx)
                .await
                .unwrap_or_else(|e| error!("Kafka processing error: {}", e));
        }
    });

    info!("Service started. Waiting for shutdown signal…");

    // Listen for shutdown signal
    signal::ctrl_c().await.expect("Failed to listen for Ctrl+C");
    info!("Shutdown signal received.");
    let _ = shutdown_tx.send(true);

    let shutdown_timeout = Duration::from_secs(15);

    let mut kafka_task = Some(kafka_task);

    select! {
        res = &mut kafka_task.as_mut().unwrap() => {
            match res {
                Ok(_) => info!("Kafka task completed gracefully."),
                Err(e) => error!("Kafka task panicked: {:?}", e),
            }
        }

        _ = time::sleep(shutdown_timeout) => {
            error!("Shutdown timeout reached. Aborting kafka task.");
            if let Some(handle) = kafka_task.take() {
                handle.abort();
                let _ = handle.await;
            }
        }
    }

    info!("Shutdown complete.");
    Ok(())
}

/// Main function — entry point.
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
