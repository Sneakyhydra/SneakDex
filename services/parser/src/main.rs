//! Entry point for the SneakDex parser service.

use anyhow::Result;
use std::{sync::Arc, time::Duration};
use tokio::{select, signal, sync::watch, task::JoinHandle, time};
use tracing::{error, info};

mod internal;

use internal::config::Config;
use internal::core::KafkaHandler;
use internal::monitor::{start_monitor_server, Metrics};
use internal::parser::HtmlParser;

/// Initializes and runs the parser service.
async fn run() -> Result<()> {
    // Load .env file if it exists (for local development)
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
    let metrics = Arc::new(Metrics::new());

    // Shutdown signal notifier
    let (shutdown_tx, shutdown_rx) = watch::channel(false);

    // Start monitor server
    let monitor_port = config.monitor_port;
    let metrics_clone = metrics.clone();
    let kafka_clone = kafka_handler.clone();
    let kafka_shutdown_send = shutdown_tx.clone();
    let monitor_shutdown_send = shutdown_tx.clone();
    let monitor_shutdown = shutdown_rx.clone();

    let mut monitor_task: Option<JoinHandle<()>> = Some(tokio::spawn(async move {
        if let Err(e) = start_monitor_server(
            monitor_port,
            metrics_clone,
            kafka_clone,
            monitor_shutdown,
            monitor_shutdown_send,
        )
        .await
        {
            error!("Monitor server failed: {}", e);
        }
    }));

    // Kafka processing task
    let mut kafka_task: Option<JoinHandle<()>> = Some(tokio::spawn({
        let shutdown_rx = shutdown_rx.clone();
        async move {
            kafka_handler
                .start_processing(parser, metrics, shutdown_rx, kafka_shutdown_send)
                .await
                .unwrap_or_else(|e| error!("Kafka processing error: {}", e));
        }
    }));

    info!("Service started. Waiting for shutdown signal…");

    // Listen for shutdown signal
    signal::ctrl_c().await.expect("Failed to listen for Ctrl+C");
    info!("Shutdown signal received.");
    let _ = shutdown_tx.send(true);

    let shutdown_timeout = Duration::from_secs(15);

    select! {
        _ = async {
            if let Some(handle) = &mut kafka_task {
                handle.await.ok();
            }
            if let Some(handle) = &mut monitor_task {
                handle.await.ok();
            }
        } => {
            info!("All tasks completed gracefully.");
        }

        _ = time::sleep(shutdown_timeout) => {
            error!("Shutdown timeout reached. Aborting remaining tasks.");
            if let Some(handle) = kafka_task.take() {
                handle.abort();
                let _ = handle.await;
            }
            if let Some(handle) = monitor_task.take() {
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
    if let Err(e) = run().await {
        error!("Parser service error: {}", e);
        Err(e)
    } else {
        Ok(())
    }
}
