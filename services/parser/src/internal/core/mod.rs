//! Core of the parser service.
//!
//! This module provides a `KafkaHandler` that consumes raw HTML messages
//! from Kafka, parses them, and produces structured `ParsedPage` messages
//! back to another Kafka topic.

use anyhow::{bail, Context, Result};
use rdkafka::config::ClientConfig;
use rdkafka::consumer::{Consumer, StreamConsumer};
use rdkafka::message::Message;
use rdkafka::producer::{FutureProducer, FutureRecord};
use std::sync::atomic::Ordering;
use std::sync::Arc;
use tokio::sync::Semaphore;
use tokio::time::{sleep, Duration};
use tracing::{debug, error, info, warn};

use crate::internal::config::Config;
use crate::internal::monitor::Metrics;
use crate::internal::parser::HtmlParser;

/// Handles Kafka interactions: consuming raw HTML and producing parsed pages.
pub struct KafkaHandler {
    consumer: StreamConsumer,
    producer: FutureProducer,
    config: Arc<Config>,
}

impl KafkaHandler {
    /// Create and initialize a new `KafkaHandler`.
    ///
    /// Connects to Kafka as both a consumer and a producer, and subscribes to the
    /// configured `kafka_topic_html`.
    ///
    /// # Errors
    /// Returns an error if the Kafka consumer or producer cannot be created or
    /// if subscribing to the topic fails.
    pub async fn new(config: Arc<Config>) -> Result<Self> {
        info!("SneakDex Parser Starting...");
        debug!("Configuration: {:?}", config);

        // Initialize Kafka consumer.
        let consumer: StreamConsumer = ClientConfig::new()
            .set("group.id", &config.kafka_group_id)
            .set("bootstrap.servers", &config.kafka_brokers)
            .set("enable.partition.eof", "false")
            .set("session.timeout.ms", "6000")
            .set("enable.auto.commit", "true")
            .create()
            .context("Failed to create Kafka consumer")?;

        // Initialize Kafka producer.
        let producer: FutureProducer = ClientConfig::new()
            .set("bootstrap.servers", &config.kafka_brokers)
            .set("message.timeout.ms", "5000")
            .set("compression.type", "snappy")
            .create()
            .context("Failed to create Kafka producer")?;

        // Subscribe consumer to the HTML topic.
        consumer
            .subscribe(&[&config.kafka_topic_html])
            .context("Failed to subscribe to topics")?;

        info!("Subscribed to topic: {}", config.kafka_topic_html);

        Ok(Self {
            consumer,
            producer,
            config: config.clone(),
        })
    }

    pub async fn is_connected(&self) -> bool {
        let client = self.consumer.client();
        match client.fetch_metadata(None, std::time::Duration::from_secs(2)) {
            Ok(_) => true,
            Err(e) => {
                warn!("Kafka health check failed: {:?}", e);
                false
            }
        }
    }

    /// Start processing messages in an infinite loop with graceful shutdown.
    ///
    /// For each message, the HTML payload is parsed using the provided `HtmlParser`
    /// and the result is sent to the parsed-pages Kafka topic.
    pub async fn start_processing(
        &self,
        parser: HtmlParser,
        metrics: Arc<Metrics>,
        mut shutdown: tokio::sync::watch::Receiver<bool>,
        shutdown_tx: tokio::sync::watch::Sender<bool>,
    ) -> anyhow::Result<()> {
        let semaphore = Arc::new(Semaphore::new(self.config.max_concurrency));

        info!(
            "Starting with max {} concurrent workers, waiting for messages...",
            self.config.max_concurrency
        );

        loop {
            tokio::select! {
                // watch for shutdown
                res = shutdown.changed() => {
                    let _ = shutdown_tx.send(true);
                    if res.is_ok() {
                        info!("Shutdown signal received, stopping Kafka processing loop.");
                        sleep(Duration::from_secs(10)).await;
                        break;
                    } else {
                        error!("Shutdown channel closed unexpectedly.");
                        sleep(Duration::from_secs(10)).await;
                        break;
                    }
                }

                // process Kafka messages
                msg_res = self.consumer.recv() => {
                    let msg = match msg_res {
                        Ok(msg) => msg,
                        Err(e) => {
                            error!("Failed to receive message from Kafka: {}", e);
                            metrics.inc_kafka_errored();
                            continue;
                        }
                    };

                    let permit = match semaphore.clone().acquire_owned().await {
                        Ok(permit) => permit,
                        Err(e) => {
                            error!("Semaphore acquisition failed: {}", e);
                            continue;
                        }
                    };

                    let parser_clone = parser.clone();
                    let metrics_clone = metrics.clone();
                    let producer_clone = self.producer.clone();
                    let config_clone = self.config.clone();
                    let owned_msg = msg.detach();

                    // spawn a task to process the message
                    tokio::spawn(async move {
                        if metrics_clone.pages_processed.load(Ordering::Relaxed) % 100 == 0 {
                            info!(
                                "Metrics: inflight={}, processed={}, successful={}, failed={}, kafka_ok={}, kafka_fail={}, kafka_err={}",
                                metrics_clone.get_inflight_pages(),
                                metrics_clone.get_pages_processed(),
                                metrics_clone.get_pages_successful(),
                                metrics_clone.get_pages_failed(),
                                metrics_clone.get_kafka_successful(),
                                metrics_clone.get_kafka_failed(),
                                metrics_clone.get_kafka_errored(),
                            );
                        }

                        metrics_clone.inc_pages_processed();
                        metrics_clone.inc_inflight_pages();

                        if let Err(e) = KafkaHandler::process_message(
                            &owned_msg,
                            &parser_clone,
                            &metrics_clone,
                            &producer_clone,
                            Arc::clone(&config_clone),
                        ).await {
                            error!("Error processing message: {}", e);
                            metrics_clone.inc_pages_failed();
                        }

                        metrics_clone.dec_inflight_pages();
                        drop(permit); // release the semaphore slot
                    });
                }
            }
        }

        info!("Kafka processing loop exited.");
        Ok(())
    }

    /// Process a single Kafka message.
    ///
    /// Decodes the key and payload, parses the HTML, and sends the parsed result
    /// to the parsed-pages topic.
    async fn process_message(
        message: &rdkafka::message::OwnedMessage,
        parser: &HtmlParser,
        metrics: &Arc<Metrics>,
        producer: &FutureProducer,
        config: Arc<Config>,
    ) -> Result<()> {
        // Extract URL (key).
        let url = match message.key() {
            Some(key) => String::from_utf8_lossy(key).to_string(),
            None => {
                bail!("No URL key, page skipped");
            }
        };

        // Extract HTML payload.
        let payload = match message.payload() {
            Some(data) => data,
            None => {
                bail!("No Payload, page skipped");
            }
        };

        let html = String::from_utf8_lossy(payload);
        info!("Processing HTML from URL: {}", url);

        // Parse the HTML.
        match parser.parse_html(&html, &url) {
            Ok(parsed) => {
                metrics.inc_pages_successful();
                KafkaHandler::send_parsed_page(
                    &url,
                    &parsed,
                    metrics,
                    producer,
                    Arc::clone(&config),
                )
                .await?;
            }
            Err(e) => {
                error!("Failed to parse HTML from {}: {}", url, e);
                return Err(e);
            }
        }

        Ok(())
    }

    /// Serialize and send a parsed page to the `parsed-pages` Kafka topic.
    async fn send_parsed_page(
        url: &str,
        parsed: &crate::internal::parser::models::ParsedPage,
        metrics: &Arc<Metrics>,
        producer: &FutureProducer,
        config: Arc<Config>,
    ) -> Result<()> {
        // Serialize the parsed page to JSON.
        let json_data = serde_json::to_string(parsed).context("Failed to serialize parsed page")?;

        let record = FutureRecord::to(&config.kafka_topic_parsed)
            .key(url)
            .payload(&json_data);

        // Send to Kafka.
        match producer.send(record, Duration::from_secs(0)).await {
            Ok(_) => {
                metrics.inc_kafka_successful();
                info!(
                    "Parsed and sent page: {} (words: {}, total: {})",
                    url,
                    parsed.word_count,
                    metrics.pages_processed.load(Ordering::Relaxed)
                );
            }
            Err((e, _)) => {
                error!("Failed to send message to Kafka: {}", e);
                // Heuristically decide if it’s a payload / message size or network error
                if e.to_string().contains("MessageSizeTooLarge") {
                    metrics.inc_kafka_failed();
                } else {
                    metrics.inc_kafka_errored();
                }
            }
        }

        Ok(())
    }
}
