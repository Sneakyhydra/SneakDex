# Sneakdex Web Crawler Service

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![Build Status](https://img.shields.io/badge/Build-Passing-green.svg)]()

A high-performance, distributed web crawler service designed for enterprise-scale content discovery and processing. Built with Go for optimal performance and resource efficiency.

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
- [Performance Tuning](#performance-tuning)
- [Security](#security)

## 🔍 Overview

The Sneakdex Web Crawler is a production-ready, distributed web crawling service that efficiently discovers, delivers web content for downstream analysis. It's designed to handle high-throughput crawling operations while maintaining respectful crawling practices and robust error handling.

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
  │    - pending_urls (List)             │
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
  │    (Parser, Indexer, etc.)            │
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

- **Go**: Version 1.24 or higher
- **Redis**: Version 7.0+ (for URL queue management)
- **Kafka**: Version 3.0+ (for content publishing)
- **Memory**: Minimum 2GB RAM per instance (1GB for light workloads)

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
USER_AGENT=Sneakdex/1.0               # HTTP User-Agent string
ENABLE_DEBUG=false                    # Enable debug features
MONITOR_PORT=8080                     # Health check and metrics port
```

### Configuration Examples

#### Production Environment

```bash
# Production configuration for high-throughput crawling
export REDIS_HOST=redis-cluster.prod.company.com
export REDIS_PORT=6379
export REDIS_DB=0
export KAFKA_BROKERS=kafka1.prod:9092,kafka2.prod:9092,kafka3.prod:9092
export KAFKA_TOPIC_HTML=raw-html
export MAX_CONCURRENCY=64
export MAX_PAGES=100000
export REQUEST_DELAY=50ms
export LOG_LEVEL=info
export MONITOR_PORT=8080
```

#### Development Environment

```bash
# Development configuration for testing
export REDIS_HOST=redis
export KAFKA_BROKERS=kafka:9092
export MAX_CONCURRENCY=8
export MAX_PAGES=100
export REQUEST_DELAY=500ms
export LOG_LEVEL=debug
export ENABLE_DEBUG=true
```

## 🚀 Usage

### Basic Operation

```bash
# Start the crawler with default configuration
./crawler

# Start with custom configuration
REDIS_HOST=myredis.com KAFKA_BROKERS=mykafka.com:9092 ./crawler

# Run in development mode with debug logging
LOG_LEVEL=debug ENABLE_DEBUG=true ./crawler
```

### Docker Compose Example

```yaml
networks:
  monitoring:
    driver: bridge
  sneakdex-network:
    driver: bridge

volumes:
  kafka-data:
  redis-data:

services:
  kafka:
    image: bitnami/kafka:latest
    container_name: sneakdex-kafka
    ports:
      - "9092:9092"
      - "9999:9999"
    environment:
      - KAFKA_CFG_NODE_ID=1
      - KAFKA_CFG_PROCESS_ROLES=broker,controller
      - KAFKA_CFG_LISTENERS=PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093
      - KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://kafka:9092
      - KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      - KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=1@kafka:9093
      - KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER
      - ALLOW_PLAINTEXT_LISTENER=yes
      - KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE=true
      - KAFKA_CFG_MESSAGE_MAX_BYTES=10485760
      - KAFKA_CFG_REPLICA_FETCH_MAX_BYTES=10485760
      # Enable JMX for metrics
      - KAFKA_ENABLE_KRAFT=yes
      - KAFKA_CFG_JMX_PORT=9999
    volumes:
      - kafka-data:/bitnami/kafka
    networks:
      - sneakdex-network
      - monitoring
    healthcheck:
      test:
        ["CMD-SHELL", "kafka-topics.sh --bootstrap-server 0.0.0.0:9092 --list"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 60s
    restart: unless-stopped

  redis:
    image: redis:7
    container_name: sneakdex-redis
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes
    networks:
      - sneakdex-network
      - monitoring
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5
      start_period: 15s
    restart: unless-stopped

  crawler-dev:
    build:
      context: ./services/crawler
      dockerfile: Dockerfile.dev
    # For Air live reload, enable init to true
    init: true
    environment:
      - KAFKA_BROKERS=kafka:9092
      - KAFKA_TOPIC_HTML=raw-html
      - KAFKA_RETRY_MAX=3
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - REDIS_PASSWORD=""
      - REDIS_DB=0
      - REDIS_TIMEOUT=5s
      - REDIS_RETRY_MAX=3
      - START_URLS=https://www.dhruvrishishwar.com,https://en.wikipedia.org/wiki/Special:Random,https://simple.wikipedia.org/wiki/Special:Random,https://news.ycombinator.com,https://www.reuters.com/news/archive/worldNews,https://www.bbc.com/news,https://github.com/trending,https://stackoverflow.com/questions,https://dev.to,https://developer.mozilla.org/en-US/docs/Web,https://arxiv.org/list/cs/new,https://eng.uber.com,https://netflixtechblog.com,https://blog.cloudflare.com
      - CRAWL_DEPTH=3
      - MAX_PAGES=10000
      - MAX_CONCURRENCY=64
      - REQUEST_TIMEOUT=10s
      - REQUEST_DELAY=1ms
      - MAX_CONTENT_SIZE=2621440 # 2.5 MB
      - LOG_LEVEL=info
      - USER_AGENT=Sneakdex/1.0
      - ENABLE_DEBUG=false
      - MONITOR_PORT=8080
    volumes:
      - ./services/crawler/cmd:/app/cmd
      - ./services/crawler/internal:/app/internal
      - ./services/crawler/.air.toml:/app/.air.toml
      - ./services/crawler/go.mod:/app/go.mod
      - ./services/crawler/go.sum:/app/go.sum
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
      start_period: 60s
    restart: unless-stopped
```

## 🔗 API Endpoints

The crawler exposes HTTP endpoints for monitoring and health checking:

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
Content-Type: text/plain
ok

# Unhealthy service
HTTP/1.1 503 Service Unavailable
Content-Type: text/plain
Redis unhealthy: connection refused
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

### Key Performance Indicators (KPIs)

#### Throughput Metrics

- **Pages/Second**: `pages_processed_total / crawler_uptime_seconds`
- **Success Rate**: `pages_successful_total / pages_processed_total * 100`
- **Kafka Delivery Rate**: `kafka_successful_total / pages_successful_total * 100`

#### Health Metrics

- **Redis Connectivity**: `redis_successful_total / (redis_successful_total + redis_errored_total)`
- **Error Rate**: `pages_failed_total / pages_processed_total * 100`
- **Resource Utilization**: Memory and CPU usage

### Prometheus Queries

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

### Grafana Dashboard

Key dashboard panels:

1. **Throughput Graph**: Pages processed over time
2. **Success Rate Gauge**: Current success percentage
3. **Error Log Table**: Recent errors and warnings
4. **Resource Usage**: Memory and CPU utilization
5. **Queue Depth**: Redis queue length trends

### Alerting Rules

```yaml
groups:
  - name: crawler.rules
    rules:
      - alert: CrawlerHighErrorRate
        expr: rate(pages_failed_total[5m]) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Crawler error rate is high"

      - alert: CrawlerDown
        expr: up{job="crawler"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Crawler instance is down"

      - alert: RedisConnectionIssues
        expr: redis_errored_total > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Redis connection issues detected"
```

### Log Analysis

The crawler outputs structured JSON logs suitable for log aggregation:

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:45Z",
  "msg": "Page processed successfully",
  "url": "https://example.com/page1",
  "content_size": 45231,
  "duration_ms": 234
}
```

**Log Levels:**

- `ERROR`: Critical issues requiring immediate attention
- `WARN`: Non-critical issues that should be monitored
- `INFO`: Normal operational events
- `DEBUG`: Detailed debugging information (development only)

### Code Quality Tools

```bash
# Format code
go fmt ./...
goimports -w .

# Lint code
golangci-lint run
```

### Project Structure

```
├── cmd/crawler/           # Application entry point
│   └── main.go           # Main application logic
├── internal/             # Private application code
│   ├── config/          # Configuration management
│   ├── crawler/         # Core crawling logic
│   ├── logger/          # Logging setup
│   ├── metrics/         # Performance metrics
│   ├── monitor/         # Health checks and monitoring
│   └── validator/       # URL validation and filtering
├── Dockerfile           # Production container image
├── Dockerfile.dev       # Development container image
├── .air.toml           # Hot reload configuration
├── go.mod              # Go module definition
└── README.md           # This file
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

### Log Analysis

Common log patterns to monitor:

```bash
# Error patterns
grep "ERROR" /var/log/crawler.log | tail -50

# Performance patterns
grep "pages_per_second" /var/log/crawler.log

# URL validation failures
grep "invalid.*URL" /var/log/crawler.log
```

## ⚡ Performance Tuning

### Crawler Optimization

#### Concurrency Settings

```bash
# Conservative (respectful crawling)
MAX_CONCURRENCY=8
REQUEST_DELAY=500ms

# Moderate (balanced performance)
MAX_CONCURRENCY=32
REQUEST_DELAY=100ms

# Aggressive (maximum performance)
MAX_CONCURRENCY=64
REQUEST_DELAY=50ms
```

#### Memory Optimization

```bash
# Reduce memory usage
MAX_CONTENT_SIZE=1048576    # 1MB limit
CRAWL_DEPTH=2               # Shallow crawling

# Cache optimization
REDIS_TIMEOUT=5s            # Faster Redis timeouts
```

### Monitoring-Based Tuning

Use metrics to guide optimization:

```bash
# Monitor queue depth
redis-cli llen crawler:pending_urls

# Monitor success rates
curl -s http://localhost:8080/metrics | grep success

# Monitor resource usage
docker stats crawler-container
```

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

# No sensitive data in layers
RUN --mount=type=secret,id=password ...
```

#### Runtime Security

```bash
# Read-only filesystem
docker run --read-only --tmpfs /tmp sneakdex-crawler

# Security context (Kubernetes)
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  readOnlyRootFilesystem: true
```
