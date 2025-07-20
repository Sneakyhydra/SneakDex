# SneakDex Indexer Service

A scalable, distributed document and image indexing service designed for enterprise-scale search and discovery. Built with Python and powered by Sentence Transformers, Qdrant, and Supabase for high-quality semantic indexing and sparse indexing.

## 📋 Table of Contents

- [SneakDex Indexer Service](#sneakdex-indexer-service)
  - [📋 Table of Contents](#-table-of-contents)
  - [🔍 Overview](#-overview)
    - [Key Capabilities](#key-capabilities)
  - [🏗️ Architecture](#️-architecture)
    - [Components](#components)
  - [✨ Features](#-features)
    - [Core Functionality](#core-functionality)
    - [Reliability \& Performance](#reliability--performance)
    - [Monitoring \& Operations](#monitoring--operations)
    - [Security \& Compliance](#security--compliance)
  - [🔧 Prerequisites](#-prerequisites)
    - [System Requirements](#system-requirements)
    - [Infrastructure Dependencies](#infrastructure-dependencies)
    - [Network Requirements](#network-requirements)
  - [⚙️ Configuration](#️-configuration)
    - [Environment Variables](#environment-variables)
      - [Kafka Configuration](#kafka-configuration)
      - [Qdrant Configuration](#qdrant-configuration)
      - [Supabase/Postgres Configuration](#supabasepostgres-configuration)
      - [Application Settings](#application-settings)
      - [Database Settings](#database-settings)
    - [Configuration Examples](#configuration-examples)
      - [Production Environment](#production-environment)
      - [Development Environment](#development-environment)
  - [🚀 Usage](#-usage)
    - [Basic Operation](#basic-operation)
    - [Docker Compose Example](#docker-compose-example)
  - [🔗 API Endpoints](#-api-endpoints)
    - [Health Check Endpoint](#health-check-endpoint)
    - [Metrics Endpoint](#metrics-endpoint)
  - [📊 Monitoring \& Observability](#-monitoring--observability)
    - [Metrics Exposed](#metrics-exposed)
    - [Key Performance Indicators (KPIs)](#key-performance-indicators-kpis)
      - [Throughput Metrics](#throughput-metrics)
      - [Health Metrics](#health-metrics)
    - [Sample Prometheus Queries](#sample-prometheus-queries)
  - [🚀 Deployment](#-deployment)
    - [Scaling Guidelines](#scaling-guidelines)
      - [Horizontal Scaling](#horizontal-scaling)
      - [Vertical Scaling](#vertical-scaling)
  - [🐛 Troubleshooting](#-troubleshooting)
    - [Common Issues](#common-issues)
      - [1. Qdrant Connection Failures](#1-qdrant-connection-failures)
      - [2. Supabase Insert Errors](#2-supabase-insert-errors)
      - [3. Kafka Consumption Stalls](#3-kafka-consumption-stalls)
      - [4. High Memory/CPU Usage](#4-high-memorycpu-usage)
  - [🔒 Security](#-security)
    - [Network Security](#network-security)
      - [Firewall Rules](#firewall-rules)
    - [Data Validation \& Sanitization](#data-validation--sanitization)
      - [Payload Validation](#payload-validation)
      - [Qdrant \& Supabase Authentication](#qdrant--supabase-authentication)
      - [Batch Size Limits](#batch-size-limits)
    - [Container Security](#container-security)
      - [Dockerfile Security Best Practices](#dockerfile-security-best-practices)
  - [📜 License](#-license)

## 🔍 Overview

SneakDex Indexer consumes parsed web pages and images from Kafka, generates semantic embeddings, and indexes them into Qdrant (vector search) and Supabase (Postgres) for efficient retrieval and search.

### Key Capabilities

- **Semantic Indexing**: Generates dense vector embeddings for documents and images using Sentence Transformers.
- **Batch Processing**: Processes and indexes documents in configurable batches for high throughput.
- **Content Storage**: Simultaneously stores metadata & text in Supabase (Sparse indexing) and embeddings in Qdrant.
- **Monitoring**: Built-in health checks and Prometheus-compatible metrics.
- **Graceful Handling**: Signal-aware shutdown and error-tolerant consumer loop.

## 🏗️ Architecture

```
┌──────────────────────────┐
│        Kafka Topic       │
│       parsed-pages       │
│   (Parsed HTML & Images) │
└──────────────┬───────────┘
               │
    ┌──────────┴──────────┐
    │     Indexer Service │
    │  (Orchestrator.py)  │
    └──────────┬──────────┘
               │
   ┌───────────┼────────────┬────────────┐
   │           │            │            │
┌──┴──┐    ┌───┴───┐   ┌────┴────┐  ┌────┴────┐
│Embed│    │Qdrant │   │Supabase │  │Monitor  │
│Model│    │Client │   │Client   │  │Server   │
└─────┘    └───────┘   └─────────┘  └─────────┘

Optional Enterprise Configuration:
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Kafka Load Bal. │────│ Indexer 1       │────│ Indexer N       │
│ (Enterprise)    │    │ Instance        │    │ Instance        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
     (With horizontal scaling & centralized metrics)

```

### Components

- **Kafka Consumer**: Reads parsed pages & images from Kafka (parsed-pages topic).
- **Embedding Model**: Sentence Transformers-based semantic vector generator.
- **Vector Store Client**: Indexes vectors & metadata into Qdrant collections.
- **Relational Store Client**: Inserts structured metadata & text into Supabase/Postgres for sparse indexing using tsvector.
- **Monitor Server**: Health & metrics endpoints (/health, /metrics).
- **Configuration Manager**: Pydantic-based environment configuration loader & validator.

## ✨ Features

### Core Functionality

- ✅ **Semantic Indexing** - Embeds documents & images into vector space using Transformers
- ✅ **Batch Processing** - Processes documents in configurable batches for efficiency
- ✅ **Multi-Store Indexing** - Stores vectors in Qdrant and metadata in Supabase
- ✅ **Image & Text Support** - Captions & embeds both page text and associated images
- ✅ **Language Detection Ready** - Stores language metadata alongside content
- ✅ **Text Snippets** - Saves short text snippets for previews & search display

### Reliability & Performance

- ✅ **Batch Size Control** - Adjustable batch size & max document limits for tuning
- ✅ **Fault Tolerant** - Skips bad messages, logs failures, continues processing
- ✅ **Concurrent Encoding** - Offloads embedding computation efficiently
- ✅ **Optimized Storage Writes** - Uses Qdrant & Supabase clients efficiently for low latency

### Monitoring & Operations

- ✅ **Health Checks** - /health endpoint checks Qdrant, Supabase, and Kafka config
- ✅ **Prometheus Metrics** - Exposes processing metrics
- ✅ **Structured Logging** - Rich logs with context for debugging & observability
- ✅ **Vector Count Gauge** - Tracks Qdrant collection size in real time
- ✅ **Alerting Ready** - Metrics compatible with Prometheus & Grafana dashboards

### Security & Compliance

- ✅ **Environment-based Secrets** - No hardcoded credentials, fully configurable
- ✅ **Payload Sanitization** - Cleans and validates input data before indexing
- ✅ **Content Size Limits** - Truncates excessively large fields to protect storage & memory
- ✅ **Signal-safe** - Properly closes Kafka consumers & releases resources
- ✅ **Container Friendly** - Runs well in Docker/Kubernetes environments

## 🔧 Prerequisites

### System Requirements

- **Python**: = 3.12
- **Kafka**: = 4.0.0 (for content publishing)
- **Qdrant**: = 1.0.0 (for vector storage)
- **Supabase**: = 2.0.0 (for metadata storage)

### Infrastructure Dependencies

- **Kafka Cluster**: Multi-broker setup for reliability & throughput

- **Qdrant Cluster**: Recommended for production-scale vector storage

- **Supabase/Postgres**: Managed Postgres or Supabase project

- **Monitoring Stack**: Prometheus + Grafana for metrics & dashboards

- **Load Balancer**: _(Optional)_ For horizontally scaled deployments with centralized monitoring

### Network Requirements

- **Kafka Access**: Port 9092 (or as configured)
- **Qdrant Access**: Port 6333 (or as configured)
- **Supabase/Postgres Access**: HTTPS endpoint (provided by Supabase)
- **Health Check Port**: Port 8080 (configurable)

## ⚙️ Configuration

### Environment Variables

The indexer is configured entirely through environment variables for container-friendly deployment:

#### Kafka Configuration

```bash
KAFKA_BROKERS=kafka:9092                # Comma-separated Kafka brokers
KAFKA_TOPIC_PARSED=parsed-pages         # Topic for parsed page & image content
KAFKA_GROUP_ID=indexer-group            # Kafka consumer group ID
```

#### Qdrant Configuration

```bash
QDRANT_URL=http://qdrant:6333           # Qdrant service endpoint
QDRANT_API_KEY=                         # Qdrant API key (if required)
COLLECTION_NAME=sneakdex               # Qdrant collection for page embeddings
COLLECTION_NAME_IMAGES=sneakdex-images # Qdrant collection for image embeddings

```

#### Supabase/Postgres Configuration

```bash
SUPABASE_URL=https://your-project.supabase.co  # Supabase project URL
SUPABASE_API_KEY=                              # Supabase service key

```

#### Application Settings

```bash
BATCH_SIZE=100
MAX_DOCS=10000
MONITOR_PORT=8080
```

#### Database Settings

```sql
DROP TABLE IF EXISTS documents;

CREATE TABLE documents (
    id UUID PRIMARY KEY,
    url TEXT,
    title TEXT,
    lang TEXT DEFAULT 'simple',
    content_tsv TSVECTOR
);

CREATE INDEX idx_documents_content_tsv ON documents USING GIN (content_tsv);

-- temp column just for insert-time text
ALTER TABLE documents ADD COLUMN _tmp_content TEXT;

-- trigger function
CREATE OR REPLACE FUNCTION update_content_tsv_and_strip()
RETURNS trigger AS $$
BEGIN
    NEW.content_tsv := to_tsvector(COALESCE(NEW.lang, 'simple')::regconfig, NEW._tmp_content);
    NEW._tmp_content := NULL; -- discard cleaned_text
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- attach trigger
CREATE TRIGGER trg_update_content_tsv
BEFORE INSERT ON documents
FOR EACH ROW
EXECUTE FUNCTION update_content_tsv_and_strip();
```

Search function -
```sql
CREATE OR REPLACE FUNCTION search_documents(q text, limit_count int)
RETURNS TABLE (
    id uuid,
    url text,
    title text,
    rank double precision
) AS $$
BEGIN
  RETURN QUERY
  SELECT
    d.id,
    d.url,
    d.title,
    ts_rank(d.content_tsv, plainto_tsquery(q))::double precision AS rank
  FROM documents d
  WHERE d.content_tsv @@ plainto_tsquery(q)
  ORDER BY rank DESC
  LIMIT limit_count;
END;
$$ LANGUAGE plpgsql STABLE;
```

Table summary-
```sql
WITH stats AS (
  SELECT count(*) AS cnt FROM documents
), sizes AS (
  SELECT 
    pg_relation_size('documents') AS table_bytes,
    pg_total_relation_size('documents') AS total_bytes,
    pg_indexes_size('documents') AS index_bytes
)
SELECT 
    s.cnt AS row_count,
    pg_size_pretty(sz.table_bytes) AS table_size,
    pg_size_pretty(sz.index_bytes) AS index_size,
    pg_size_pretty(sz.total_bytes) AS total_size,
    pg_size_pretty(sz.total_bytes / GREATEST(s.cnt,1)) AS avg_row_size
FROM stats s, sizes sz;
```

### Configuration Examples

#### Production Environment

```env
KAFKA_BROKERS=kafka:9092
KAFKA_TOPIC_PARSED=parsed-pages
KAFKA_GROUP_ID=indexer-group
BATCH_SIZE=1000
MAX_DOCS=100000
QDRANT_URL=http://qdrant:6333
QDRANT_API_KEY=some-qdrant-api-key
COLLECTION_NAME=sneakdex
COLLECTION_NAME_IMAGES=sneakdex-images
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_API_KEY=some-supabase-api-key
MONITOR_PORT=8080
```

#### Development Environment

```env
KAFKA_BROKERS=kafka:9092
KAFKA_TOPIC_PARSED=parsed-pages
KAFKA_GROUP_ID=indexer-group
BATCH_SIZE=100
MAX_DOCS=1000
QDRANT_URL=http://qdrant:6333
QDRANT_API_KEY=some-qdrant-api-key
COLLECTION_NAME=sneakdex
COLLECTION_NAME_IMAGES=sneakdex-images
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_API_KEY=some-supabase-api-key
MONITOR_PORT=8080
```

## 🚀 Usage

### Basic Operation

```bash
# From project root
python -m src.main

# Or explicitly run main.py
python src/main.py

# With environment variables loaded from .env
export $(cat .env | xargs) && python -m src.main

# Example with custom config
BATCH_SIZE=500 MAX_DOCS=10000 python -m src.main
```

### Docker Compose Example

Add this to your `docker-compose.yml` alongside Kafka & other services:

```yaml
indexer:
  build:
    context: ./services/indexer
  environment:
    - PYTHONUNBUFFERED=1
    - PYTHONDONTWRITEBYTECODE=1
    - PYTHON_ENV=development
  env_file:
    - ./services/indexer/.env
  volumes:
    - ./services/indexer:/app
  working_dir: /app
  depends_on:
    kafka:
      condition: service_healthy
  networks:
    - sneakdex-network
    - monitoring
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
    interval: 10s
    timeout: 5s
    retries: 3
    start_period: 15s
  restart: unless-stopped
```

or for production:
```yaml
indexer:
  build:
    context: ./services/indexer
    dockerfile: Dockerfile.prod
  environment:
    - PYTHONUNBUFFERED=1
    - PYTHONDONTWRITEBYTECODE=1
    - PYTHON_ENV=production
  env_file:
    - ./services/indexer/.env.production
  depends_on:
    kafka:
      condition: service_healthy
  networks:
    - sneakdex-network
    - monitoring
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
    interval: 30s
    timeout: 5s
    retries: 3
    start_period: 10s
```

## 🔗 API Endpoints

### Health Check Endpoint

**GET /health**

Returns the health status of the indexer service.

```bash
curl http://localhost:8080/health
```

**Responses:**

```json
# Healthy service
HTTP/1.1 200 OK
Content-Type: application/json
{
  "status": "ok",
  "components": {
    "qdrant": "ok",
    "supabase": "ok",
    "kafka": "configured"
  },
  "current_vectors": 5234
}

# Degraded service
HTTP/1.1 200 OK
Content-Type: application/json
{
  "status": "degraded",
  "components": {
    "qdrant": "error: connection timeout",
    "supabase": "ok",
    "kafka": "configured"
  },
  "current_vectors": null
}
```

**Health Check Criteria:**

- ✅ Qdrant connectivity and collection stats
- ✅ Supabase query execution
- ✅ Kafka configuration detected

### Metrics Endpoint

**GET /metrics**

Returns Prometheus-formatted metrics for monitoring.

```bash
curl http://localhost:8080/metrics
```

**Sample Output:**

```
# HELP indexer_messages_consumed_total Total number of messages consumed
# TYPE indexer_messages_consumed_total counter
indexer_messages_consumed_total 3021

# HELP indexer_batches_indexed_total Total number of batches successfully indexed
# TYPE indexer_batches_indexed_total counter
indexer_batches_indexed_total 121

# HELP indexer_messages_failed_total Total number of messages that failed to process
# TYPE indexer_messages_failed_total counter
indexer_messages_failed_total 7

# HELP indexer_current_vectors Current number of vectors in Qdrant collection
# TYPE indexer_current_vectors gauge
indexer_current_vectors 5234

```

## 📊 Monitoring & Observability

### Metrics Exposed

- `indexer_messages_consumed_total`
- `indexer_batches_indexed_total`
- `indexer_messages_failed_total`
- `indexer_current_vectors`

### Key Performance Indicators (KPIs)

#### Throughput Metrics

- **Messages/Second**: `indexer_messages_consumed_total / process_uptime_seconds`
- **Batch Success Rate**: `indexer_batches_indexed_total / (indexer_batches_indexed_total + indexer_messages_failed_total) * 100`
- **Qdrant Vector Count**: `indexer_current_vectors`

#### Health Metrics

- **Qdrant Connectivity**: monitored via `/health` and `indexer_current_vectors`
- **Supabase Insert Success**: inferred from logs and low `indexer_messages_failed_total`
- **Error Rate**: `indexer_messages_failed_total / indexer_messages_consumed_total * 100`
- **Resource Utilization**: Memory and CPU usage (external monitoring recommended)

### Sample Prometheus Queries

```promql
# Messages consumed per second
rate(indexer_messages_consumed_total[5m])

# Batch success rate percentage
indexer_batches_indexed_total / (indexer_batches_indexed_total + indexer_messages_failed_total) * 100

# Error rate alert
rate(indexer_messages_failed_total[5m]) > 0.1

# Current vector count in Qdrant
indexer_current_vectors
```

## 🚀 Deployment

### Scaling Guidelines

#### Horizontal Scaling

- **CPU Usage**: Scale up when CPU > 70% for 5+ minutes
- **Memory Usage**: Scale up when memory > 80%
- **Kafka Lag**: Scale out when Kafka consumer lag increases and stays > 10,000 messages
- **Error Rate**: Scale out when indexer_messages_failed_total / indexer_messages_consumed_total > 5%

#### Vertical Scaling

- **Memory**: Increase for handling large batches & embedding large documents/images
- **CPU**: Increase to speed up embedding computation (especially with CPU-bound SentenceTransformers)
- **Disk I/O & Network**: Ensure low latency access to Qdrant & Supabase in high-throughput scenarios
- **Add GPU**: For GPU-based embedding models, ensure sufficient GPU resources are available (Change requirements.txt accordingly)

## 🐛 Troubleshooting

### Common Issues

#### 1. Qdrant Connection Failures

```bash
# Symptoms
ERROR: Qdrant health check failed

# Diagnosis
curl -X GET http://<QDRANT_URL>/collections
ping <QDRANT_HOST>

# Solutions
- Verify Qdrant is running and accessible
- Check Qdrant API key and URL configuration
- Check network connectivity and firewall rules
```

#### 2. Supabase Insert Errors

```bash
# Symptoms
ERROR: Supabase insert failed. Response: …

# Diagnosis
Check Supabase logs or query directly via SQL:
SELECT * FROM documents LIMIT 1;

# Solutions
- Verify Supabase credentials (URL, API key)
- Ensure `documents` table exists
- Check database quota or row limits
- Validate payload schema matches table definition
```

#### 3. Kafka Consumption Stalls

```bash
# Symptoms
No messages consumed; consumer lag growing

# Diagnosis
kafka-consumer-groups.sh --describe --group $KAFKA_GROUP_ID --bootstrap-server $KAFKA_BROKERS

# Solutions
- Verify Kafka cluster is healthy
- Check `parsed-pages` topic exists and has messages
- Restart consumer to reset offset if needed
- Review group ID and topic configurations
```

#### 4. High Memory/CPU Usage

```bash
# Symptoms
Container killed due to OOMKilled; high CPU load

# Diagnosis
- Monitor `/metrics` endpoint for resource usage
- Check batch size & document sizes

# Solutions
- Reduce BATCH_SIZE
- Increase container memory/CPU limits
- Optimize SentenceTransformer model choice
- Run with GPU acceleration if available
```

## 🔒 Security

### Network Security

#### Firewall Rules

- **Inbound**: Only port 8080 open for health checks and metrics
- **Outbound**: Access to Supabase & Qdrant endpoints, Kafka brokers
- **Internal**: Kafka (9092) and Qdrant/Supabase communication over secure channels if available

### Data Validation & Sanitization

The indexer implements several security measures for input and storage safety:

#### Payload Validation

- Validates and sanitizes parsed-page JSON from Kafka before processing
- Enforces max document & image sizes
- Ignores malformed or incomplete records

#### Qdrant & Supabase Authentication

- Requires API keys and secure connections to both Qdrant and Supabase
- Stores no credentials in code — relies on environment variables

#### Batch Size Limits

```bash
# Prevent resource exhaustion
BATCH_SIZE=1000
```

### Container Security

#### Dockerfile Security Best Practices

```dockerfile
# Use non-root user if possible
USER appuser

# Minimal base image
FROM python:3.12-slim

# Install only necessary dependencies
RUN pip install -r requirements.txt
```

## 📜 License

MIT — feel free to use & contribute.
