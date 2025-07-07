use anyhow::{Context, Result};
use log::{debug, error, info, warn};
use rdkafka::config::ClientConfig;
use rdkafka::consumer::{Consumer, StreamConsumer};
use rdkafka::message::{Header, Headers, Message};
use rdkafka::producer::{FutureProducer, FutureRecord};
use scraper::{Html, Selector};
use serde::{Deserialize, Serialize};
use std::time::Duration;
use url::Url;
use regex::Regex;

#[derive(Debug, Deserialize)]
struct Config {
    #[serde(default = "default_kafka_brokers")]
    kafka_brokers: String,
    #[serde(default = "default_kafka_topic_html")]
    kafka_topic_html: String,
    #[serde(default = "default_kafka_topic_parsed")]
    kafka_topic_parsed: String,
    #[serde(default = "default_kafka_group_id")]
    kafka_group_id: String,
    #[serde(default = "default_log_level")]
    rust_log: String,
    #[serde(default = "default_max_content_length")]
    max_content_length: usize,
    #[serde(default = "default_min_content_length")]
    min_content_length: usize,
}

fn default_kafka_brokers() -> String { "kafka:9092".to_string() }
fn default_kafka_topic_html() -> String { "raw-html".to_string() }
fn default_kafka_topic_parsed() -> String { "parsed-pages".to_string() }
fn default_kafka_group_id() -> String { "parser-group".to_string() }
fn default_log_level() -> String { "info".to_string() }
fn default_max_content_length() -> usize { 5_000_000 } // 5MB
fn default_min_content_length() -> usize { 100 } // 100 chars

#[derive(Debug, Serialize)]
struct ImageData {
    src: String,
    alt: Option<String>,
    title: Option<String>,
}

#[derive(Debug, Serialize)]
struct LinkData {
    url: String,
    text: String,
    is_external: bool,
}

#[derive(Debug, Serialize)]
struct ParsedPage {
    url: String,
    title: String,
    description: Option<String>,
    body_text: String,
    cleaned_text: String,
    headings: Vec<String>,
    links: Vec<LinkData>,
    images: Vec<ImageData>,
    canonical_url: Option<String>,
    language: Option<String>,
    word_count: usize,
    meta_keywords: Option<String>,
    schema_data: Option<String>,
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
            max_content_length: default_max_content_length(),
            min_content_length: default_min_content_length(),
        }
    }
}

fn clean_text(text: &str) -> String {
    let re = Regex::new(r"\s+").unwrap();
    re.replace_all(text.trim(), " ").to_string()
}

fn extract_headings(document: &Html) -> Vec<String> {
    let heading_selector = Selector::parse("h1, h2, h3, h4, h5, h6").unwrap();
    document
        .select(&heading_selector)
        .map(|element| clean_text(&element.text().collect::<String>()))
        .filter(|text| !text.is_empty())
        .collect()
}

fn extract_links(document: &Html, base_url: &str) -> Vec<LinkData> {
    let link_selector = Selector::parse("a[href]").unwrap();
    let base = Url::parse(base_url).ok();
    
    document
        .select(&link_selector)
        .filter_map(|element| {
            let href = element.value().attr("href")?;
            let text = clean_text(&element.text().collect::<String>());
            
            // Skip empty links or javascript/mailto
            if href.starts_with("javascript:") || href.starts_with("mailto:") || text.is_empty() {
                return None;
            }
            
            // Resolve relative URLs
            let resolved_url = if let Some(base) = &base {
                base.join(href).map(|url| url.to_string()).unwrap_or_else(|_| href.to_string())
            } else {
                href.to_string()
            };
            
            // Check if external
            let is_external = if let (Ok(base_url), Ok(link_url)) = (Url::parse(base_url), Url::parse(&resolved_url)) {
                base_url.domain() != link_url.domain()
            } else {
                false
            };
            
            Some(LinkData {
                url: resolved_url,
                text,
                is_external,
            })
        })
        .collect()
}

fn extract_images(document: &Html, base_url: &str) -> Vec<ImageData> {
    let img_selector = Selector::parse("img[src]").unwrap();
    let base = Url::parse(base_url).ok();
    
    document
        .select(&img_selector)
        .filter_map(|element| {
            let src = element.value().attr("src")?;
            let alt = element.value().attr("alt").map(|s| s.to_string());
            let title = element.value().attr("title").map(|s| s.to_string());
            
            // Resolve relative URLs
            let resolved_src = if let Some(base) = &base {
                base.join(src).map(|url| url.to_string()).unwrap_or_else(|_| src.to_string())
            } else {
                src.to_string()
            };
            
            Some(ImageData {
                src: resolved_src,
                alt,
                title,
            })
        })
        .collect()
}

fn extract_main_content(document: &Html) -> String {
    // Try to find main content areas first
    let main_selectors = [
        "main",
        "article",
        ".content",
        "#content",
        ".post-content",
        ".entry-content",
        ".article-content",
    ];
    
    for selector_str in &main_selectors {
        if let Ok(selector) = Selector::parse(selector_str) {
            if let Some(element) = document.select(&selector).next() {
                return clean_text(&element.text().collect::<String>());
            }
        }
    }
    
    // Fallback to body, but remove common noise
    let body_selector = Selector::parse("body").unwrap();
    let noise_selectors = [
        "nav", "header", "footer", "aside", 
        ".navigation", ".sidebar", ".ads", 
        ".comments", ".related", ".social"
    ];
    
    if let Some(body) = document.select(&body_selector).next() {
        let mut text = body.text().collect::<String>();
        
        // Remove noise content (this is basic - could be more sophisticated)
        for noise_selector in &noise_selectors {
            if let Ok(selector) = Selector::parse(noise_selector) {
                for element in document.select(&selector) {
                    let noise_text = element.text().collect::<String>();
                    text = text.replace(&noise_text, "");
                }
            }
        }
        
        clean_text(&text)
    } else {
        String::new()
    }
}

fn detect_language(text: &str) -> Option<String> {
    // Basic language detection - you might want to use a proper library
    // This is a placeholder for actual language detection
    if text.len() < 50 {
        return None;
    }
    
    // Simple heuristic - count common English words
    let english_words = ["the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by"];
    let words: Vec<&str> = text.split_whitespace().collect();
    let english_count = words.iter()
        .filter(|word| english_words.contains(&word.to_lowercase().as_str()))
        .count();
    
    if english_count > words.len() / 20 {
        Some("en".to_string())
    } else {
        None
    }
}

fn parse_html(html: &str, url: &str, config: &Config) -> Result<ParsedPage> {
    // Content length validation
    if html.len() > config.max_content_length {
        return Err(anyhow::anyhow!("Content too large: {} bytes", html.len()));
    }
    
    let document = Html::parse_document(html);
    
    // Extract title
    let title_selector = Selector::parse("title").unwrap();
    let title = document
        .select(&title_selector)
        .next()
        .map(|element| clean_text(&element.inner_html()))
        .unwrap_or_else(|| "No Title".to_string());
    
    // Extract meta description
    let description_selector = Selector::parse("meta[name='description']").unwrap();
    let description = document
        .select(&description_selector)
        .next()
        .and_then(|element| element.value().attr("content"))
        .map(|content| clean_text(content));
    
    // Extract meta keywords
    let keywords_selector = Selector::parse("meta[name='keywords']").unwrap();
    let meta_keywords = document
        .select(&keywords_selector)
        .next()
        .and_then(|element| element.value().attr("content"))
        .map(|content| clean_text(content));
    
    // Extract canonical URL
    let canonical_selector = Selector::parse("link[rel='canonical']").unwrap();
    let canonical_url = document
        .select(&canonical_selector)
        .next()
        .and_then(|element| element.value().attr("href"))
        .map(|href| href.to_string());
    
    // Extract main content
    let cleaned_text = extract_main_content(&document);
    
    // Validate minimum content length
    if cleaned_text.len() < config.min_content_length {
        return Err(anyhow::anyhow!("Content too short: {} characters", cleaned_text.len()));
    }
    
    // Extract other elements
    let headings = extract_headings(&document);
    let links = extract_links(&document, url);
    let images = extract_images(&document, url);
    
    // Word count
    let word_count = cleaned_text.split_whitespace().count();
    
    // Language detection
    let language = detect_language(&cleaned_text);
    
    // Extract structured data (JSON-LD)
    let schema_selector = Selector::parse("script[type='application/ld+json']").unwrap();
    let schema_data = document
        .select(&schema_selector)
        .next()
        .map(|element| element.inner_html());
    
    // Fallback body text (for backward compatibility)
    let body_selector = Selector::parse("body").unwrap();
    let body_text = document
        .select(&body_selector)
        .next()
        .map(|element| clean_text(&element.text().collect::<String>()))
        .unwrap_or_else(|| String::new());
    
    Ok(ParsedPage {
        url: url.to_string(),
        title,
        description,
        body_text,
        cleaned_text,
        headings,
        links,
        images,
        canonical_url,
        language,
        word_count,
        meta_keywords,
        schema_data,
        timestamp: chrono::Utc::now(),
    })
}

// ... rest of your existing main function and run() function stays the same
// just update the parse_html call to include config:
// match parse_html(&html, &url, &config) {

async fn run() -> Result<()> {
    let config: Config = envy::from_env().unwrap_or_default();
    
    std::env::set_var("RUST_LOG", &config.rust_log);
    env_logger::init();
    
    info!("Starting enhanced HTML parser service");
    debug!("Configuration: {:?}", config);
    
    let consumer: StreamConsumer = ClientConfig::new()
        .set("group.id", &config.kafka_group_id)
        .set("bootstrap.servers", &config.kafka_brokers)
        .set("enable.partition.eof", "false")
        .set("session.timeout.ms", "6000")
        .set("enable.auto.commit", "true")
        .create()
        .context("Failed to create Kafka consumer")?;
    
    let producer: FutureProducer = ClientConfig::new()
        .set("bootstrap.servers", &config.kafka_brokers)
        .set("message.timeout.ms", "5000")
        .create()
        .context("Failed to create Kafka producer")?;
    
    consumer
        .subscribe(&[&config.kafka_topic_html])
        .context("Failed to subscribe to topics")?;
    
    info!("Subscribed to topic: {}", config.kafka_topic_html);
    info!("Waiting for messages...");
    
    let mut message_count = 0;
    
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
                
                match parse_html(&html, &url, &config) {
                    Ok(parsed) => {
                        let json_data = match serde_json::to_string(&parsed) {
                            Ok(json) => json,
                            Err(e) => {
                                error!("Failed to serialize parsed page: {}", e);
                                continue;
                            }
                        };
                        
                        let record = FutureRecord::to(&config.kafka_topic_parsed)
                            .key(&url)
                            .payload(&json_data);
                        
                        match producer.send(record, Duration::from_secs(0)).await {
                            Ok(_) => {
                                message_count += 1;
                                info!("Parsed and sent page: {} (words: {}, total: {})", 
                                      url, parsed.word_count, message_count);
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