# SneakDex Web Crawler Service

A high-performance, distributed web crawler service designed for enterprise-scale content discovery and processing. Built with Go for optimal performance and resource efficiency.

## 📋 Table of Contents

- [SneakDex Web Crawler Service](#sneakdex-web-crawler-service)
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
      - [Redis Configuration](#redis-configuration)
      - [Kafka Configuration](#kafka-configuration)
      - [Crawling Behavior](#crawling-behavior)
      - [Performance Settings](#performance-settings)
      - [Application Settings](#application-settings)
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
      - [1. Redis Connection Failures](#1-redis-connection-failures)
      - [2. Kafka Publishing Errors](#2-kafka-publishing-errors)
      - [3. High Memory Usage](#3-high-memory-usage)
      - [4. Slow Crawling Performance](#4-slow-crawling-performance)
    - [Debug Mode](#debug-mode)
  - [🔒 Security](#-security)
    - [Network Security](#network-security)
      - [Firewall Rules](#firewall-rules)
    - [URL Validation Security](#url-validation-security)
      - [IP Address Filtering](#ip-address-filtering)
      - [Domain Filtering](#domain-filtering)
      - [Content Size Limits](#content-size-limits)
    - [Container Security](#container-security)
      - [Dockerfile Security Best Practices](#dockerfile-security-best-practices)
  - [📜 License](#-license)

## 🔍 Overview

The SneakDex Web Crawler is a production-ready, distributed web crawling service that efficiently discovers, delivers web content for downstream analysis. It's designed to handle high-throughput crawling operations while maintaining respectful crawling practices and robust error handling.

### Key Capabilities

- **Distributed Crawling**: Uses Redis for URL queue management across multiple instances
- **High Performance**: Concurrent crawling with configurable parallelism
- **Content Processing**: Kafka integration for real-time content delivery
- **URL Validation**: Comprehensive URL filtering and normalization
- **Monitoring**: Built-in health checks and Prometheus metrics
- **Graceful Handling**: Respectful crawling with rate limiting and robots.txt compliance

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐
│   Crawler 1     │    │   Crawler N     │
│   Instance      │    │   Instance      │
└─────────────────┘    └─────────────────┘
         │                        │
         │        Redis           │
  ┌──────┴────────────────────────┴──────┐
  │    Distributed URL Queue             │
  │    - pending_urls (Lists depth wise) │
  │    - visited_urls (Set)              │
  │    - requeued_urls (Set)             │
  └─────────────────┬────────────────────┘
                    │
  ┌─────────────────┴─────────────────────┐
  │           Kafka Cluster               │
  │    Topic: raw-html                    │
  │    (Crawled Content)                  │
  └───────────────────────────────────────┘
                    │
  ┌─────────────────┴─────────────────────┐
  │       Downstream Processors           │
  │    (Parser, Indexer)                  │
  └───────────────────────────────────────┘


Optional Enterprise Configuration:
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Load Balancer │    │   Crawler 1     │    │   Crawler N     │
│  (Enterprise)   │────│   Instance      │    │   Instance      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
     (Only for centralized health checks and metrics aggregation)
```

### Components

- **Crawler Engine**: Colly-based high-performance web scraper
- **URL Manager**: Redis-backed distributed queue with deduplication
- **Content Publisher**: Kafka producer for real-time content streaming
- **URL Validator**: Security and quality filtering with caching
- **Monitor Server**: Health checks and metrics exposition
- **Configuration Manager**: Environment-based configuration with validation

## ✨ Features

### Core Functionality

- ✅ **Distributed Crawling** - Multiple instances coordinate via Redis
- ✅ **High Concurrency** - Configurable parallel request handling
- ✅ **URL Deduplication** - Intelligent caching prevents duplicate work
- ✅ **Content Filtering** - Skip non-HTML content automatically
- ✅ **Rate Limiting** - Respectful crawling with configurable delays
- ✅ **Depth Control** - Configurable crawling depth limits

### Reliability & Performance

- ✅ **Auto Retry** - Exponential backoff for transient failures
- ✅ **Graceful Shutdown** - Clean resource cleanup on termination
- ✅ **Circuit Breaking** - Automatic error handling and recovery
- ✅ **Memory Optimization** - Multi-level caching reduces resource usage
- ✅ **Connection Pooling** - Efficient resource utilization

### Monitoring & Operations

- ✅ **Health Checks** - HTTP endpoints for direct instance monitoring
- ✅ **Prometheus Metrics** - Comprehensive performance monitoring
- ✅ **Structured Logging** - JSON logs for easy parsing and analysis
- ✅ **Real-time Stats** - Live performance metrics and dashboards
- ✅ **Alerting Ready** - Metrics compatible with monitoring stacks

### Security & Compliance

- ✅ **URL Validation** - Prevent access to private/malicious IPs
- ✅ **Content Size Limits** - Prevent memory exhaustion attacks
- ✅ **Domain Filtering** - Whitelist/blacklist support
- ✅ **User Agent** - Proper identification for transparency
- ✅ **Timeout Protection** - Prevent resource exhaustion

## 🔧 Prerequisites

### System Requirements

- **Go**: = 1.24
- **Redis**: = 7.0 (for URL queue management)
- **Kafka**: = 4.0.0 (for content publishing)

### Infrastructure Dependencies

- **Redis Cluster**: High-availability Redis setup recommended
- **Kafka Cluster**: Multi-broker setup for production workloads
- **Monitoring Stack**: Prometheus + Grafana for observability
- **Load Balancer**: _(Optional)_ Only needed for enterprise scenarios requiring centralized health check endpoints

### Network Requirements

- **Outbound HTTP/HTTPS**: Access to target websites
- **Redis Access**: Port 6379 (configurable)
- **Kafka Access**: Port 9092 (configurable)
- **Health Check Port**: Port 8080 (configurable)

## ⚙️ Configuration

### Environment Variables

The crawler is configured entirely through environment variables for container-friendly deployment:

#### Redis Configuration

```bash
REDIS_HOST=redis                      # Redis server hostname
REDIS_PORT=6379                       # Redis server port
REDIS_PASSWORD=                       # Redis password (optional)
REDIS_DB=0                            # Redis database number
REDIS_TIMEOUT=15s                     # Redis operation timeout
REDIS_RETRY_MAX=3                     # Maximum Redis retry attempts
```

#### Kafka Configuration

```bash
KAFKA_BROKERS=kafka:9092                # Comma-separated Kafka brokers
KAFKA_TOPIC_HTML=raw-html               # Topic for crawled HTML content
KAFKA_RETRY_MAX=3                       # Maximum Kafka retry attempts
```

#### Crawling Behavior

```bash
START_URLS=https://example.com,https://example.org  # Initial crawling seeds
CRAWL_DEPTH=3                                       # Maximum crawling depth
MAX_PAGES=10000                                     # Maximum pages per session
URL_WHITELIST=example.com,trusted.org               # Allowed domains (optional)
URL_BLACKLIST=spam.com,malicious.org                # Blocked domains (optional)
```

#### Performance Settings

```bash
MAX_CONCURRENCY=32                    # Concurrent request limit
REQUEST_TIMEOUT=15s                   # HTTP request timeout
REQUEST_DELAY=100ms                   # Delay between requests
MAX_CONTENT_SIZE=2621440              # Maximum content size (2.5MB)
```

#### Application Settings

```bash
LOG_LEVEL=info                        # Logging level (trace,debug,info,warn,error)
USER_AGENT=SneakDex/1.0               # HTTP User-Agent string
ENABLE_DEBUG=false                    # Enable debug features
MONITOR_PORT=8080                     # Health check and metrics port
```

### Configuration Examples

#### Production Environment

```env
CRAWL_DEPTH=3
ENABLE_DEBUG=false
KAFKA_BROKERS=kafka:9092
KAFKA_RETRY_MAX=3
KAFKA_TOPIC_HTML=raw-html
LOG_LEVEL=info
MAX_CONCURRENCY=64
MAX_CONTENT_SIZE=10485760
MAX_PAGES=10000
MONITOR_PORT=8080
REDIS_DB=0
REDIS_HOST=redis
REDIS_PORT=6379
REQUEST_DELAY=10ms
REQUEST_TIMEOUT=10s
START_URLS=https://en.wikipedia.org/wiki/Special:Random,https://simple.wikipedia.org/wiki/Special:Random,https://news.ycombinator.com,https://www.reuters.com/news/archive/worldNews,https://www.bbc.com/news,https://github.com/trending,https://stackoverflow.com/questions,https://dev.to,https://developer.mozilla.org/en-US/docs/Web,https://arxiv.org/list/cs/new,https://eng.uber.com,https://netflixtechblog.com,https://blog.cloudflare.com,https://www.dhruvrishishwar.com
USER_AGENT=SneakDex/1.0
GO_ENV=production
CGO_ENABLED=0
```

#### Development Environment

```env
CRAWL_DEPTH=3
ENABLE_DEBUG=false
KAFKA_BROKERS=kafka:9092
KAFKA_RETRY_MAX=3
KAFKA_TOPIC_HTML=raw-html
LOG_LEVEL=info
MAX_CONCURRENCY=16
MAX_CONTENT_SIZE=10485760
MAX_PAGES=10000
MONITOR_PORT=8080
REDIS_DB=0
REDIS_HOST=redis
REDIS_PORT=6379
REQUEST_DELAY=50ms
REQUEST_TIMEOUT=10s
START_URLS=https://en.wikipedia.org/wiki/Special:Random,https://simple.wikipedia.org/wiki/Special:Random,https://news.ycombinator.com,https://www.reuters.com/news/archive/worldNews,https://www.bbc.com/news,https://github.com/trending,https://stackoverflow.com/questions,https://dev.to,https://developer.mozilla.org/en-US/docs/Web,https://arxiv.org/list/cs/new,https://eng.uber.com,https://netflixtechblog.com,https://blog.cloudflare.com,https://www.dhruvrishishwar.com
USER_AGENT=SneakDex/1.0
GO_ENV=development
CGO_ENABLED=0
```

## 🚀 Usage

### Basic Operation

```bash
# From project root
go run cmd/crawler/main.go

# Alternative syntax
go run ./cmd/crawler

# Build from project root
go build -o crawler cmd/crawler/main.go

# Run
./crawler

# Or build with package path
go build -o crawler ./cmd/crawler
./crawler
```

### Docker Compose Example

Add this to your `docker-compose.yml` alongside Kafka & other services:

```yaml
crawler:
  build:
    context: ./services/crawler
  init: true
  env_file:
    - ./services/crawler/.env
  volumes:
    - ./services/crawler:/app
    - go-mod-cache:/go/pkg/mod
  depends_on:
    kafka:
      condition: service_healthy
    redis:
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
crawler:
  build:
    context: ./services/crawler
    dockerfile: Dockerfile.prod
  env_file:
    - ./services/crawler/.env.production
  depends_on:
    kafka:
      condition: service_healthy
    redis:
      condition: service_healthy
  networks:
    - sneakdex-network
    - monitoring
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
    interval: 10s
    timeout: 5s
    retries: 3
    start_period: 10s
  restart: unless-stopped
```

## 🔗 API Endpoints

### Health Check Endpoint

**GET /health**

Returns the health status of the crawler service.

```bash
curl http://localhost:8080/health
```

**Responses:**

```bash
# Healthy service
HTTP/1.1 200 OK
Content-Type: application/json
{
  Status:    "healthy",
  Timestamp: time.Now().UTC(),
  Services:  make(map[string]string),
  Errors:    []string{},
}

# Unhealthy service
HTTP/1.1 503 Service Unavailable
Content-Type: application/json
{
  Status:    "unhealthy",
  Timestamp: time.Now().UTC(),
  Services:  make(map[string]string),
  Errors:    []string{},
}
```

**Health Check Criteria:**

- ✅ Redis connectivity and responsiveness
- ✅ Kafka producer availability
- ✅ System resource availability

### Metrics Endpoint

**GET /metrics**

Returns Prometheus-formatted metrics for monitoring.

```bash
curl http://localhost:8080/metrics
```

**Sample Output:**

```
# HELP pages_processed_total Total number of pages processed
# TYPE pages_processed_total gauge
pages_processed_total 1523

# HELP pages_successful_total Total number of pages successfully processed
# TYPE pages_successful_total gauge
pages_successful_total 1445

# HELP kafka_successful_total Successful Kafka messages sent
# TYPE kafka_successful_total gauge
kafka_successful_total 1445

# HELP crawler_uptime_seconds Crawler uptime in seconds
# TYPE crawler_uptime_seconds gauge
crawler_uptime_seconds 3661.23
```

## 📊 Monitoring & Observability

### Metrics Exposed

- `crawler_inflight_pages`
- `crawler_pages_processed_total`
- `crawler_pages_successful_total`
- `crawler_pages_failed_total`
- `crawler_kafka_successful_total`
- `crawler_kafka_failed_total`
- `crawler_kafka_errored_total`
- `crawler_redis_successful_total`
- `crawler_redis_failed_total`
- `crawler_redis_errored_total`
- `crawler_uptime_seconds`

### Key Performance Indicators (KPIs)

#### Throughput Metrics

- **Pages/Second**: `pages_processed_total / crawler_uptime_seconds`
- **Success Rate**: `pages_successful_total / pages_processed_total * 100`
- **Kafka Delivery Rate**: `kafka_successful_total / pages_successful_total * 100`

#### Health Metrics

- **Redis Connectivity**: `redis_successful_total / (redis_successful_total + redis_errored_total)`
- **Error Rate**: `pages_failed_total / pages_processed_total * 100`
- **Resource Utilization**: Memory and CPU usage

### Sample Prometheus Queries

```promql
# Pages per second
rate(pages_processed_total[5m])

# Success rate percentage
pages_successful_total / pages_processed_total * 100

# Error rate alert
rate(pages_failed_total[5m]) > 0.1

# Redis health
redis_successful_total / (redis_successful_total + redis_errored_total) < 0.95
```

## 🚀 Deployment

### Scaling Guidelines

#### Horizontal Scaling

- **CPU Usage**: Scale up when CPU > 70% for 5+ minutes
- **Memory Usage**: Scale up when memory > 80%
- **Queue Depth**: Scale up when Redis queue length > 10,000
- **Error Rate**: Scale up when error rate > 5%

#### Vertical Scaling

- **Memory**: Increase for high URL deduplication cache hit rates
- **CPU**: Increase for high workloads
- **Network**: Ensure adequate bandwidth for high-throughput scenarios

## 🐛 Troubleshooting

### Common Issues

#### 1. Redis Connection Failures

```bash
# Symptoms
ERROR: Failed to connect to Redis after 3 attempts

# Diagnosis
redis-cli -h $REDIS_HOST -p $REDIS_PORT ping
telnet $REDIS_HOST $REDIS_PORT

# Solutions
- Verify Redis is running and accessible
- Check network connectivity and firewall rules
- Verify Redis authentication credentials
- Increase REDIS_TIMEOUT if network is slow
```

#### 2. Kafka Publishing Errors

```bash
# Symptoms
ERROR: Failed to send message to Kafka: connection refused

# Diagnosis
kafka-topics.sh --list --bootstrap-server $KAFKA_BROKERS
kafka-console-producer.sh --topic $KAFKA_TOPIC_HTML --bootstrap-server $KAFKA_BROKERS

# Solutions
- Verify Kafka cluster is running
- Check topic exists and has proper permissions
- Verify network connectivity to Kafka brokers
- Check Kafka broker configuration
```

#### 3. High Memory Usage

```bash
# Symptoms
Container killed due to OOMKilled

# Diagnosis
- Monitor /metrics endpoint for memory metrics
- Check Redis cache sizes
- Review URL deduplication cache size

# Solutions
- Increase container memory limits
- Reduce MAX_CONCURRENCY setting
- Implement cache size limits
- Optimize URL normalization logic
```

#### 4. Slow Crawling Performance

```bash
# Symptoms
Low pages/second rate in metrics

# Diagnosis
curl http://localhost:8080/metrics | grep pages_processed

# Solutions
- Increase MAX_CONCURRENCY (be respectful)
- Reduce REQUEST_DELAY if appropriate
- Optimize Redis connection pooling
- Check network latency to target sites
```

### Debug Mode

Enable comprehensive debugging:

```bash
export ENABLE_DEBUG=true
export LOG_LEVEL=debug
./crawler
```

Debug mode provides:

- Detailed request/response logging
- Cache hit/miss statistics
- URL validation decision logs
- Performance timing information

## 🔒 Security

### Network Security

#### Firewall Rules

- **Inbound**: Only port 8080 for health checks
- **Outbound**: HTTP/HTTPS (80, 443) for crawling
- **Internal**: Redis (6379) and Kafka (9092) access

### URL Validation Security

The crawler implements multiple security layers:

#### IP Address Filtering

```go
// Prevent access to private networks
SetAllowPrivateIPs(false)   // Block 10.0.0.0/8, 192.168.0.0/16
SetAllowLoopback(false)     // Block 127.0.0.0/8
```

#### Domain Filtering

```bash
# Use whitelist for restricted crawling
URL_WHITELIST=trusted.com,partner.org

# Use blacklist for security
URL_BLACKLIST=malicious.com,spam.org
```

#### Content Size Limits

```bash
# Prevent memory exhaustion
MAX_CONTENT_SIZE=2621440  # 2.5MB limit
```

### Container Security

#### Dockerfile Security Best Practices

```dockerfile
# Use non-root user
USER appuser

# Minimal base image
FROM alpine:3.19
```

## 📜 License

MIT — feel free to use & contribute.
