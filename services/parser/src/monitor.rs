//! Health check and monitoring for the parser service.
//!
//! Provides HTTP endpoints for liveness, health checks, and basic metrics.

use actix_web::{get, web, App, HttpResponse, HttpServer, Responder};
use serde::Serialize;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use std::time::Instant;
use tracing::info;

use crate::kafka_client::KafkaHandler;

/// Metrics shared across the service.
#[derive(Debug, Clone)]
pub struct Metrics {
    pub pages_processed: Arc<AtomicU64>,
    pub pages_successful: Arc<AtomicU64>,
    pub pages_failed: Arc<AtomicU64>,
    pub kafka_successful: Arc<AtomicU64>,
    pub kafka_failed: Arc<AtomicU64>,
    pub kafka_errored: Arc<AtomicU64>,
    pub last_message_time: Arc<tokio::sync::RwLock<Option<Instant>>>,
    pub start_time: Instant,
}

impl Metrics {
    pub fn new() -> Self {
        Self {
            pages_processed: Arc::new(AtomicU64::new(0)),
            pages_successful: Arc::new(AtomicU64::new(0)),
            pages_failed: Arc::new(AtomicU64::new(0)),
            kafka_successful: Arc::new(AtomicU64::new(0)),
            kafka_failed: Arc::new(AtomicU64::new(0)),
            kafka_errored: Arc::new(AtomicU64::new(0)),
            last_message_time: Arc::new(tokio::sync::RwLock::new(None)),
            start_time: Instant::now(),
        }
    }

    pub fn inc_pages_processed(&self) {
        self.pages_processed.fetch_add(1, Ordering::Relaxed);

        let last_time = self.last_message_time.clone();
        tokio::spawn(async move {
            let mut time = last_time.write().await;
            *time = Some(Instant::now());
        });
    }

    pub fn inc_pages_successful(&self) {
        self.pages_successful.fetch_add(1, Ordering::Relaxed);
    }

    pub fn inc_pages_failed(&self) {
        self.pages_failed.fetch_add(1, Ordering::Relaxed);
    }

    pub fn inc_kafka_successful(&self) {
        self.kafka_successful.fetch_add(1, Ordering::Relaxed);
    }

    pub fn inc_kafka_failed(&self) {
        self.kafka_failed.fetch_add(1, Ordering::Relaxed);
    }

    pub fn inc_kafka_errored(&self) {
        self.kafka_errored.fetch_add(1, Ordering::Relaxed);
    }
}

/// Health check response.
#[derive(Serialize)]
struct HealthResponse {
    status: String,
    uptime_seconds: u64,
    pages_processed: u64,
    pages_failed: u64,
    kafka_errored: u64,
    last_message_age_seconds: Option<u64>,
    kafka_connected: bool,
}

/// Health check endpoint.
#[get("/health")]
async fn health(
    metrics: web::Data<Metrics>,
    kafka: web::Data<Arc<KafkaHandler>>,
) -> impl Responder {
    let uptime = metrics.start_time.elapsed().as_secs();
    let pages_processed = metrics.pages_processed.load(Ordering::Relaxed);
    let pages_failed = metrics.pages_failed.load(Ordering::Relaxed);
    let kafka_errored = metrics.kafka_errored.load(Ordering::Relaxed);

    let last_message_age = {
        let last_time = metrics.last_message_time.read().await;
        last_time.map(|time| time.elapsed().as_secs())
    };

    let kafka_ok = kafka.is_connected().await;

    let response = HealthResponse {
        status: if kafka_ok {
            "healthy".to_string()
        } else {
            "not_healthy".to_string()
        },
        uptime_seconds: uptime,
        pages_processed,
        pages_failed,
        kafka_errored,
        last_message_age_seconds: last_message_age,
        kafka_connected: kafka_ok,
    };

    HttpResponse::Ok().json(response)
}

/// Liveness check endpoint.
#[get("/live")]
async fn live() -> impl Responder {
    HttpResponse::Ok().json(serde_json::json!({
        "status": "alive",
        "timestamp": chrono::Utc::now().to_rfc3339()
    }))
}

/// Metrics endpoint (Prometheus-friendly).
#[get("/metrics")]
async fn metrics_endpoint(metrics: web::Data<Metrics>) -> impl Responder {
    let uptime = metrics.start_time.elapsed().as_secs();
    let last_message_age = {
        let last_time = metrics.last_message_time.read().await;
        last_time
            .map(|time| time.elapsed().as_secs() as i64)
            .unwrap_or(-1)
    };

    let metrics_text = format!(
        "# HELP parser_pages_processed Total pages processed\n\
         # TYPE parser_pages_processed counter\n\
         parser_pages_processed {}\n\
         \n\
         # HELP parser_pages_successful Pages processed successfully\n\
         # TYPE parser_pages_successful counter\n\
         parser_pages_successful {}\n\
         \n\
         # HELP parser_pages_failed Pages failed to process\n\
         # TYPE parser_pages_failed counter\n\
         parser_pages_failed {}\n\
         \n\
         # HELP parser_kafka_successful Kafka messages sent successfully\n\
         # TYPE parser_kafka_successful counter\n\
         parser_kafka_successful {}\n\
         \n\
         # HELP parser_kafka_failed Kafka messages failed (e.g., too big)\n\
         # TYPE parser_kafka_failed counter\n\
         parser_kafka_failed {}\n\
         \n\
         # HELP parser_kafka_errored Kafka errors (e.g., network issues)\n\
         # TYPE parser_kafka_errored counter\n\
         parser_kafka_errored {}\n\
         \n\
         # HELP parser_last_message_age Last message age in seconds\n\
         # TYPE parser_last_message_age gauge\n\
         parser_last_message_age {}\n\
         \n\
         # HELP parser_uptime_seconds Service uptime in seconds\n\
         # TYPE parser_uptime_seconds gauge\n\
         parser_uptime_seconds {}\n",
        metrics.pages_processed.load(Ordering::Relaxed),
        metrics.pages_successful.load(Ordering::Relaxed),
        metrics.pages_failed.load(Ordering::Relaxed),
        metrics.kafka_successful.load(Ordering::Relaxed),
        metrics.kafka_failed.load(Ordering::Relaxed),
        metrics.kafka_errored.load(Ordering::Relaxed),
        last_message_age,
        uptime,
    );

    HttpResponse::Ok()
        .content_type("text/plain; version=0.0.4; charset=utf-8")
        .body(metrics_text)
}

#[get("/")]
async fn index() -> impl Responder {
    HttpResponse::Ok().body("Parser monitor is running. See /health, /live, /metrics.")
}

/// Start the monitor server, with metrics & kafka checker.
pub async fn start_monitor_server(
    port: u16,
    metrics: Metrics,
    kafka_handler: Arc<KafkaHandler>,
) -> std::io::Result<()> {
    let metrics_data = web::Data::new(metrics);
    let kafka_data = web::Data::new(kafka_handler);

    info!("Starting monitor server on port {}", port);

    HttpServer::new(move || {
        App::new()
            .app_data(metrics_data.clone())
            .app_data(kafka_data.clone())
            .service(health)
            .service(live)
            .service(metrics_endpoint)
            .service(index)
    })
    .bind(("0.0.0.0", port))?
    .run()
    .await
}
