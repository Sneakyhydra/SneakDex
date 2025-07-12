# 📝 Sneakdex HTML Parser Service

A high-performance HTML parsing service that extracts structured, cleaned content from crawled web pages. Built with Rust for speed, safety, and reliability.

## 📋 Table of Contents

- [📝 Sneakdex HTML Parser Service](#-sneakdex-html-parser-service)
  - [📋 Table of Contents](#-table-of-contents)
  - [🔍 Overview](#-overview)
    - [Key Capabilities](#key-capabilities)
  - [🏗️ Architecture](#️-architecture)
    - [Components](#components)
  - [✨ Features](#-features)
    - [Core Functionality](#core-functionality)
    - [Reliability \& Performance](#reliability--performance)
    - [Monitoring \& Operations](#monitoring--operations)
  - [🔧 Prerequisites](#-prerequisites)
  - [⚙️ Configuration](#️-configuration)
    - [Environment Variables](#environment-variables)
    - [Example .env](#example-env)
  - [🚀 Usage](#-usage)
    - [Build \& Run Locally](#build--run-locally)
    - [Docker Compose Example](#docker-compose-example)
  - [🔗 API Endpoints](#-api-endpoints)
    - [Health Check](#health-check)
    - [Liveness](#liveness)
    - [Metrics](#metrics)
  - [📊 Monitoring \& Observability](#-monitoring--observability)
    - [Metrics Exposed](#metrics-exposed)
    - [Sample Prometheus Queries](#sample-prometheus-queries)
  - [🛠️ Deployment](#️-deployment)
    - [Scaling](#scaling)
  - [🐛 Troubleshooting](#-troubleshooting)
    - [Common Issues](#common-issues)
    - [Debugging](#debugging)
  - [🔒 Security](#-security)
  - [📄 Sample ParsedPage Output](#-sample-parsedpage-output)
  - [📜 License](#-license)

## 🔍 Overview

The Sneakdex HTML Parser processes raw HTML documents from Kafka and produces clean, structured data suitable for indexing or downstream analysis. It enforces strict size/content validation, extracts readable text and metadata, and detects document language.

### Key Capabilities

- **HTML Parsing**: Extracts title, meta tags, readable content, links, images, and headings
- **Content Cleaning**: Normalizes whitespace & removes noise
- **Language Detection**: Identifies document language (en, fr, etc.)
- **Kafka Integration**: Consumes raw HTML, produces structured JSON
- **Monitoring**: Health checks & Prometheus-compatible metrics

## 🏗️ Architecture

```
[Crawler] → Kafka (raw-html) → [Parser Service] → Kafka (parsed-pages) → [Indexer]
```

### Components

- **📄 HTML Parser**: Uses scraper & readability to extract meaningful text
- **🔎 Language Detector**: Uses whatlang for language inference
- **🧽 Text Utilities**: Cleans & normalizes raw text
- **📤 Kafka Client**: Robust consumer/producer using rdkafka
- **📊 Monitor Server**: Health & metrics endpoints via actix-web

## ✨ Features

### Core Functionality

- ✅ Validates size & cleans content
- ✅ Detects main readable text
- ✅ Extracts metadata: title, description, canonical URL
- ✅ Detects headings (h1–h6)
- ✅ Extracts internal & external links
- ✅ Detects images & their URLs
- ✅ Detects language & word count

### Reliability & Performance

- ✅ Concurrent processing with backpressure
- ✅ Graceful shutdown with cleanup
- ✅ Robust error handling with retries
- ✅ Memory-safe & efficient with Rust

### Monitoring & Operations

- ✅ Liveness & readiness probes
- ✅ Prometheus metrics (/metrics)
- ✅ Structured JSON logs

## 🔧 Prerequisites

- **Rust**: = 1.82
- **Kafka**: = 4.0.0
- **Docker**: optional for Kafka & deployment

## ⚙️ Configuration

### Environment Variables

| Variable             | Default        | Description                       |
| -------------------- | -------------- | --------------------------------- |
| `KAFKA_BROKERS`      | `kafka:9092`   | Kafka broker list                 |
| `KAFKA_TOPIC_HTML`   | `raw-html`     | Input topic with raw HTML         |
| `KAFKA_TOPIC_PARSED` | `parsed-pages` | Output topic with parsed pages    |
| `KAFKA_GROUP_ID`     | `parser-group` | Consumer group ID                 |
| `MAX_CONCURRENCY`    | `8`            | Max concurrent workers            |
| `MAX_CONTENT_LENGTH` | `5000000`      | Max page size in bytes            |
| `MIN_CONTENT_LENGTH` | `100`          | Min acceptable text length        |
| `MONITOR_PORT`       | `8080`         | Health & metrics HTTP port        |
| `RUST_LOG`           | `info`         | Log level (info,debug,warn,error) |

### Example .env

```env
KAFKA_BROKERS=kafka:9092
KAFKA_GROUP_ID=parser-group
KAFKA_TOPIC_HTML=raw-html
KAFKA_TOPIC_PARSED=parsed-pages
MAX_CONCURRENCY=32
MAX_CONTENT_LENGTH=5242880
MIN_CONTENT_LENGTH=1024
MONITOR_PORT=8080
CARGO_INCREMENTAL=1
CARGO_TARGET_DIR=/app/target
RUST_BACKTRACE=1
RUST_LOG=info
```

for production:
```env
KAFKA_BROKERS=kafka:9092
KAFKA_GROUP_ID=parser-group
KAFKA_TOPIC_HTML=raw-html
KAFKA_TOPIC_PARSED=parsed-pages
MAX_CONCURRENCY=64
MAX_CONTENT_LENGTH=5242880
MIN_CONTENT_LENGTH=1024
MONITOR_PORT=8080
RUST_BACKTRACE=0
RUST_LOG=info
```

## 🚀 Usage

### Build & Run Locally

```bash
cargo build --release
RUST_LOG=info ./target/release/parser
```

Or run in debug:

```bash
cargo run
```

### Docker Compose Example

Add this to your `docker-compose.yml` alongside Kafka & other services:

```yaml
parser:
  build: .
  env_file:
    - .env
  volumes:
    - .:/app
    - cargo-registry:/usr/local/cargo/registry
    - cargo-git:/usr/local/cargo/git
    - target-cache:/app/target
  working_dir: /app
  depends_on:
    kafka:
      condition: service_healthy
  networks:
    - sneakdex-network
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
    interval: 10s
    timeout: 5s
    retries: 3
    start_period: 30s
  restart: unless-stopped
```

or for production:
```yaml
parser-prod:
  build:
    context: .
    dockerfile: Dockerfile.prod
  env_file:
    - .env.production
  depends_on:
    kafka:
      condition: service_healthy
  networks:
    - sneakdex-network
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
    interval: 10s
    timeout: 5s
    retries: 3
    start_period: 30s
  restart: unless-stopped
  deploy:
    resources:
      limits:
        cpus: '2.0'
        memory: 1G
      reservations:
        cpus: '1.0'
        memory: 512M
  security_opt:
    - no-new-privileges:true
  cap_drop:
    - ALL
  cap_add:
    - NET_BIND_SERVICE
  read_only: true
  tmpfs:
    - /tmp:noexec,nosuid,size=100m
  logging:
    driver: json-file
    options:
      max-size: "10m"
      max-file: "3"
  labels:
    - "com.sneakdex.service=parser"
    - "com.sneakdex.environment=production"
```

## 🔗 API Endpoints

### Health Check

**GET** `/health`

```bash
curl http://localhost:8080/health
```

Sample Response:

```json
{
  "status": "healthy",
  "uptime_seconds": 1234,
  "pages_processed": 456,
  "pages_failed": 7,
  "kafka_errored": 1,
  "last_message_age_seconds": 2,
  "kafka_connected": true
}
```

### Liveness

**GET** `/live`

Returns HTTP 200 OK if service is alive.

```json
{
  "status": "alive",
  "timestamp": "1996-12-19T16:39:57-08:00"
}
```

### Metrics

**GET** `/metrics`

Prometheus-formatted metrics.

## 📊 Monitoring & Observability

### Metrics Exposed

- `parser_pages_processed`
- `parser_pages_successful`
- `parser_pages_failed`
- `parser_kafka_successful`
- `parser_kafka_failed`
- `parser_kafka_errored`
- `parser_last_message_age`
- `parser_uptime_seconds`

### Sample Prometheus Queries

```promql
rate(parser_pages_processed[5m])
parser_pages_failed / parser_pages_processed * 100
up{job="parser"} == 0
```

## 🛠️ Deployment

### Scaling

- Horizontal scaling supported — use separate Kafka group IDs if needed
- Monitor CPU & memory, adjust `MAX_CONCURRENCY` accordingly

## 🐛 Troubleshooting

### Common Issues

| Symptom                      | Solution                                     |
| ---------------------------- | -------------------------------------------- |
| Kafka connection errors      | Check `KAFKA_BROKERS` & Kafka cluster health |
| Content rejected (too large) | Increase `MAX_CONTENT_LENGTH`                |
| Content rejected (too short) | Lower `MIN_CONTENT_LENGTH`                   |
| High failure rate            | Review logs (`RUST_LOG=debug`)               |

### Debugging

Enable detailed logs:

```bash
RUST_LOG=debug ./parser
```

## 🔒 Security

- Enforces maximum & minimum content size
- Validates Kafka payloads
- Runs as a non-root user when containerized in production

## 📄 Sample ParsedPage Output

```json
{
  "url": "https://example.com",
  "title": "Example Domain",
  "description": "(OPTIONAL FIELD) Illustrative example domain.",
  "cleaned_text": "Example Domain This domain is for use in illustrative examples.",
  "headings": [
    { "level": 1, "text": "Example Domain" }
  ],
  "links": [
    { "url": "https://www.iana.org/domains/example", "text": "More information.", "is_external": true }
  ],
  "images": [
    { "src": "https://image-url", "alt": "(OPTIONAL FIELD)", "title": "(OPTIONAL FIELD)"}
  ],
  "canonical_url": "(OPTIONAL FIELD)",
  "language": "(OPTIONAL FIELD) en",
  "word_count": 42,
  "meta_keywords": "(OPTIONAL FIELD)",
  "timestamp": "2025-07-10T12:34:56Z",
  "content_type": "text/html",
  "encoding": "utf-8"
}
```

## 📜 License

MIT — feel free to use & contribute.
