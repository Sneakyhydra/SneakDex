use anyhow::{Context, Result};
use log::{debug, error, info, warn};
use rdkafka::config::ClientConfig;
use rdkafka::consumer::{Consumer, StreamConsumer};
use rdkafka::message::{Header, Headers, Message};
use rdkafka::producer::{FutureProducer, FutureRecord};
use scraper::{Html, Selector};
use serde::{Deserialize, Serialize};
use std::time::Duration;

#[derive(Debug, Deserialize)]
struct Config {
    // Kafka configuration
    #[serde(default = "default_kafka_brokers")]
    kafka_brokers: String,
    #[serde(default = "default_kafka_topic_html")]
    kafka_topic_html: String,
    #[serde(default = "default_kafka_topic_parsed")]
    kafka_topic_parsed: String,
    #[serde(default = "default_kafka_group_id")]
    kafka_group_id: String,
    // Logging configuration
    #[serde(default = "default_log_level")]
    rust_log: String,
}

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

fn default_log_level() -> String {
    "info".to_string()
}

#[derive(Debug, Serialize)]
struct ParsedPage {
    url: String,
    title: String,
    description: Option<String>,
    body_text: String,
    links: Vec<String>,
    timestamp: chrono::DateTime<chrono::Utc>,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            kafka_brokers: default_kafka_brokers(),
            kafka_topic_html: default_kafka_topic_html(),
            kafka_topic_parsed: default_kafka_topic_parsed(),
            kafka_group_id: default_kafka_group_id(),
            rust_log: default_log_level(),
        }
    }
}

fn parse_html(html: &str, url: &str) -> Result<ParsedPage> {
    let document = Html::parse_document(html);
    
    // Extract title
    let title_selector = Selector::parse("title").unwrap();
    let title = document
        .select(&title_selector)
        .next()
        .map(|element| element.inner_html())
        .unwrap_or_else(|| "No Title".to_string());
    
    // Extract description
    let description_selector = Selector::parse("meta[name='description']").unwrap();
    let description = document
        .select(&description_selector)
        .next()
        .and_then(|element| element.value().attr("content"))
        .map(|content| content.to_string());
    
    // Extract body text
    let body_selector = Selector::parse("body").unwrap();
    let body_text = document
        .select(&body_selector)
        .next()
        .map(|element| element.text().collect::<Vec<_>>().join(" "))
        .unwrap_or_else(|| "".to_string());
    
    // Extract links
    let link_selector = Selector::parse("a[href]").unwrap();
    let links = document
        .select(&link_selector)
        .filter_map(|element| element.value().attr("href"))
        .map(|href| href.to_string())
        .collect::<Vec<String>>();
    
    Ok(ParsedPage {
        url: url.to_string(),
        title,
        description,
        body_text,
        links,
        timestamp: chrono::Utc::now(),
    })
}

async fn run() -> Result<()> {
    // Load configuration from environment variables
    let config: Config = envy::from_env().unwrap_or_default();
    
    // Set up logging
    std::env::set_var("RUST_LOG", &config.rust_log);
    env_logger::init();
    
    info!("Starting HTML parser service");
    debug!("Configuration: {:?}", config);
    
    // Set up Kafka consumer
    let consumer: StreamConsumer = ClientConfig::new()
        .set("group.id", &config.kafka_group_id)
        .set("bootstrap.servers", &config.kafka_brokers)
        .set("enable.partition.eof", "false")
        .set("session.timeout.ms", "6000")
        .set("enable.auto.commit", "true")
        .create()
        .context("Failed to create Kafka consumer")?;
    
    // Set up Kafka producer
    let producer: FutureProducer = ClientConfig::new()
        .set("bootstrap.servers", &config.kafka_brokers)
        .set("message.timeout.ms", "5000")
        .create()
        .context("Failed to create Kafka producer")?;
    
    // Subscribe to the HTML topic
    consumer
        .subscribe(&[&config.kafka_topic_html])
        .context("Failed to subscribe to topics")?;
    
    info!("Subscribed to topic: {}", config.kafka_topic_html);
    info!("Waiting for messages...");
    
    // Process messages
    let mut message_count = 0;
    
    // Main loop
    loop {
        match consumer.recv().await {
            Ok(message) => {
                let url = match message.key() {
                    Some(key) => String::from_utf8_lossy(key).to_string(),
                    None => {
                        warn!("Received message without URL key, skipping");
                        continue;
                    }
                };
                
                let payload = match message.payload() {
                    Some(data) => data,
                    None => {
                        warn!("Received empty message payload, skipping");
                        continue;
                    }
                };
                
                let html = String::from_utf8_lossy(payload);
                
                info!("Processing HTML from URL: {}", url);
                
                // Parse the HTML
                match parse_html(&html, &url) {
                    Ok(parsed) => {
                        // Serialize to JSON
                        let json_data = match serde_json::to_string(&parsed) {
                            Ok(json) => json,
                            Err(e) => {
                                error!("Failed to serialize parsed page: {}", e);
                                continue;
                            }
                        };
                        
                        // Send to Kafka
                        let record = FutureRecord::to(&config.kafka_topic_parsed)
                            .key(&url)
                            .payload(&json_data);
                        
                        match producer.send(record, Duration::from_secs(0)).await {
                            Ok(_) => {
                                message_count += 1;
                                info!("Parsed and sent page: {} (total: {})", url, message_count);
                            }
                            Err((e, _)) => {
                                error!("Failed to send message to Kafka: {}", e);
                            }
                        }
                    }
                    Err(e) => {
                        error!("Failed to parse HTML from {}: {}", url, e);
                    }
                }
            }
            Err(e) => {
                error!("Error while receiving message: {}", e);
            }
        }
    }
}

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

