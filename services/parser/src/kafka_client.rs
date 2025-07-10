//! Kafka client for the parser service.
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
use std::time::Duration;
use tokio::sync::Semaphore;
use tracing::{debug, error, info, warn};

use crate::config::Config;
use crate::monitor::Metrics;
use crate::parser::HtmlParser;

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
        info!("Sneakdex Parser Starting...");
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

    /// Start processing messages in an infinite loop.
    ///
    /// For each message, the HTML payload is parsed using the provided `HtmlParser`
    /// and the result is sent to the parsed-pages Kafka topic.
    pub async fn start_processing(&self, parser: HtmlParser, metrics: Metrics) -> Result<()> {
        let semaphore = Arc::new(Semaphore::new(self.config.max_concurrency));
        info!(
            "Starting with max {} concurrent workers, waiting for messages...",
            self.config.max_concurrency
        );

        loop {
            let msg = match self.consumer.recv().await {
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

            // Spawn a task to handle this message
            tokio::spawn(async move {
                if metrics_clone.pages_processed.load(Ordering::Relaxed) % 100 == 0 {
                    info!(
                    "Metrics: processed={}, successful={}, failed={}, kafka_ok={}, kafka_fail={}, kafka_err={}",
                    metrics_clone.pages_processed.load(Ordering::Relaxed),
                    metrics_clone.pages_successful.load(Ordering::Relaxed),
                    metrics_clone.pages_failed.load(Ordering::Relaxed),
                    metrics_clone.kafka_successful.load(Ordering::Relaxed),
                    metrics_clone.kafka_failed.load(Ordering::Relaxed),
                    metrics_clone.kafka_errored.load(Ordering::Relaxed),
                );
                }

                metrics_clone.inc_pages_processed();

                if let Err(e) = KafkaHandler::process_message(
                    &owned_msg,
                    &parser_clone,
                    &metrics_clone,
                    &producer_clone,
                    Arc::clone(&config_clone),
                )
                .await
                {
                    error!("Error processing message: {}", e);
                    metrics_clone.inc_pages_failed();
                }

                drop(permit); // release the semaphore slot
            });
        }
    }

    /// Process a single Kafka message.
    ///
    /// Decodes the key and payload, parses the HTML, and sends the parsed result
    /// to the parsed-pages topic.
    async fn process_message(
        message: &rdkafka::message::OwnedMessage,
        parser: &HtmlParser,
        metrics: &Metrics,
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
                metrics.inc_pages_failed();
            }
        }

        Ok(())
    }

    /// Serialize and send a parsed page to the `parsed-pages` Kafka topic.
    async fn send_parsed_page(
        url: &str,
        parsed: &crate::models::ParsedPage,
        metrics: &Metrics,
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
