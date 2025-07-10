# 📝 Sneakdex HTML Parser Service

A high-performance HTML parsing service that extracts structured, cleaned content from crawled web pages. Built with Rust for speed, safety, and reliability.

## 📋 Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Configuration](#configuration)
- [Usage](#usage)
- [API Endpoints](#api-endpoints)
- [Monitoring & Observability](#monitoring--observability)
- [Deployment](#deployment)
- [Troubleshooting](#troubleshooting)
- [Security](#security)

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

- **Rust**: ≥ 1.70
- **Kafka**: ≥ 3.x
- **Docker**: optional for Kafka & deployment
- **Memory**: ≥ 512MB per instance

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
KAFKA_TOPIC_HTML=raw-html
KAFKA_TOPIC_PARSED=parsed-pages
KAFKA_GROUP_ID=parser-group
MAX_CONCURRENCY=8
MAX_CONTENT_LENGTH=5000000
MIN_CONTENT_LENGTH=100
MONITOR_PORT=8080
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
parser-dev:
    build:
      context: ./services/parser
      dockerfile: Dockerfile.dev
    environment:
      - KAFKA_BROKERS=kafka:9092
      - KAFKA_TOPIC_HTML=raw-html
      - KAFKA_TOPIC_PARSED=parsed-pages
      - KAFKA_GROUP_ID=parser-group
      - MAX_CONCURRENCY=64
      - MAX_CONTENT_LENGTH=5242880 # 5 MB
      - MIN_CONTENT_LENGTH=1024 # 1 KB
      - RUST_LOG=info
      - MONITOR_PORT=8080
    volumes:
      - ./services/parser:/app
    depends_on:
      kafka:
        condition: service_healthy
    networks:
      - sneakdex-network
      - monitoring
    healthcheck:
      test: [ "CMD", "curl", "-f", "http://localhost:8080/health" ]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 1000s
    restart: unless-stopped
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

### Metrics

**GET** `/metrics`

Prometheus-formatted metrics.

## 📊 Monitoring & Observability

### Metrics Exposed

- `pages_processed_total`
- `pages_failed_total`
- `kafka_successful_total`
- `parser_uptime_seconds`
- `content_too_large_total`
- `content_too_short_total`

### Sample Prometheus Queries

```promql
rate(pages_processed_total[5m])
pages_failed_total / pages_processed_total * 100
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
- Does not execute or render untrusted HTML
- Runs as a non-root user when containerized

## 📄 Sample ParsedPage Output

```json
{
  "url": "https://example.com",
  "title": "Example Domain",
  "description": "Illustrative example domain.",
  "cleaned_text": "Example Domain This domain is for use in illustrative examples.",
  "headings": [
    { "level": 1, "text": "Example Domain" }
  ],
  "links": [
    { "url": "https://www.iana.org/domains/example", "text": "More information.", "is_external": true }
  ],
  "images": [],
  "canonical_url": null,
  "language": "en",
  "word_count": 42,
  "meta_keywords": null,
  "timestamp": "2025-07-10T12:34:56Z",
  "content_type": "text/html",
  "encoding": "utf-8"
}
```

## 📜 License

MIT — feel free to use & contribute.