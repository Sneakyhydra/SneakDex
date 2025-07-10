# Parser Service

A high-performance HTML parsing service for the SneakDex search engine, built in Rust.

## Features

- **HTML Content Extraction**: Extracts structured data from HTML pages including titles, meta descriptions, headings, links, and images
- **Readability Processing**: Uses the `readability` crate to extract main content from web pages
- **Language Detection**: Automatically detects the language of page content
- **Kafka Integration**: Consumes raw HTML from Kafka and produces structured parsed pages
- **Health Monitoring**: Built-in health checks and metrics endpoints
- **Error Handling**: Comprehensive error handling with custom error types
- **Configuration**: Flexible configuration via environment variables
- **Docker Support**: Production-ready Docker containers

## Architecture

```
Kafka (raw-html) → Parser Service → Kafka (parsed-pages)
                     ↓
              Health Endpoints
```

## Data Model

The parser extracts the following structured data from HTML pages:

- **Basic Metadata**: URL, title, description, keywords, canonical URL
- **Content**: Main text content, cleaned text, word count, reading time
- **Structure**: Headings (H1-H6), links (internal/external), images
- **SEO Data**: Open Graph tags, Twitter Cards, schema.org structured data
- **Technical**: Content type, encoding, language detection, readability score

## Quick Start

### Prerequisites

- Rust 1.82+
- Kafka cluster
- Docker (optional)

### Local Development

1. **Clone and navigate to the parser service:**
   ```bash
   cd services/parser
   ```

2. **Set environment variables:**
   ```bash
   export KAFKA_BROKERS=localhost:9092
   export KAFKA_TOPIC_HTML=raw-html
   export KAFKA_TOPIC_PARSED=parsed-pages
   export RUST_LOG=debug
   ```

3. **Run the service:**
   ```bash
   cargo run
   ```

### Docker Development

```bash
# Build and run with Docker Compose
docker-compose -f docker-compose.dev.yml up parser

# Or build manually
docker build -f Dockerfile.dev -t parser:dev .
docker run -p 8080:8080 parser:dev
```

### Production Deployment

```bash
# Build production image
docker build -t parser:latest .

# Run with environment variables
docker run -d \
  -e KAFKA_BROKERS=kafka:9092 \
  -e KAFKA_TOPIC_HTML=raw-html \
  -e KAFKA_TOPIC_PARSED=parsed-pages \
  -e RUST_LOG=info \
  -p 8080:8080 \
  parser:latest
```

## Configuration

| Environment Variable | Default        | Description                         |
| -------------------- | -------------- | ----------------------------------- |
| `KAFKA_BROKERS`      | `kafka:9092`   | Kafka bootstrap servers             |
| `KAFKA_TOPIC_HTML`   | `raw-html`     | Topic to consume raw HTML from      |
| `KAFKA_TOPIC_PARSED` | `parsed-pages` | Topic to publish parsed pages to    |
| `KAFKA_GROUP_ID`     | `parser-group` | Kafka consumer group ID             |
| `KAFKA_RETRY_MAX`    | `3`            | Maximum Kafka retry attempts        |
| `MAX_CONCURRENCY`    | `16`           | Maximum concurrent parsing tasks    |
| `MAX_CONTENT_LENGTH` | `5000000`      | Maximum HTML content size (bytes)   |
| `MIN_CONTENT_LENGTH` | `100`          | Minimum content length (characters) |
| `RUST_LOG`           | `info`         | Logging level                       |
| `MONITOR_PORT`       | `8080`         | Health check server port            |

## Health Checks

The service exposes several health check endpoints:

- **`GET /health`**: Overall service health with metrics
- **`GET /ready`**: Readiness probe (checks Kafka connectivity)
- **`GET /live`**: Liveness probe
- **`GET /metrics`**: Prometheus-compatible metrics

### Example Health Response

```json
{
  "status": "healthy",
  "uptime_seconds": 3600,
  "messages_processed": 1500,
  "errors_total": 5,
  "last_message_age_seconds": 30
}
```

## Testing

Run the test suite:

```bash
# Run all tests
cargo test

# Run tests with output
cargo test -- --nocapture

# Run specific test
cargo test test_extract_title

# Run tests with coverage (requires cargo-tarpaulin)
cargo tarpaulin --out Html
```

## Performance

The parser service is designed for high performance:

- **Async Processing**: Uses Tokio for non-blocking I/O
- **Connection Pooling**: Efficient Kafka connection management
- **Memory Efficient**: Streaming HTML parsing with size limits
- **Concurrent Processing**: Configurable concurrency limits
- **Optimized Dependencies**: Uses high-performance Rust crates

### Benchmarks

Typical performance metrics:
- **Throughput**: 1000+ pages/second (depending on content size)
- **Memory Usage**: ~50MB baseline + ~1MB per concurrent task
- **Latency**: <100ms average parsing time

## Error Handling

The service implements comprehensive error handling:

- **Content Validation**: Size limits, content type validation
- **Kafka Errors**: Retry logic, dead letter queue support
- **Parsing Errors**: Graceful degradation, detailed error messages
- **Monitoring**: Error metrics and alerting

## Development

### Project Structure

```
src/
├── main.rs              # Application entry point
├── config.rs            # Configuration management
├── models.rs            # Data models and structures
├── error.rs             # Error types and handling
├── health.rs            # Health checks and monitoring
├── kafka_client.rs      # Kafka consumer/producer
└── parser/
    ├── mod.rs           # Main parser logic
    ├── extractors.rs    # HTML content extractors
    ├── text_utils.rs    # Text processing utilities
    ├── language_detector.rs # Language detection
    └── tests.rs         # Comprehensive test suite
```

### Adding New Extractors

1. Add extraction logic to `src/parser/extractors.rs`
2. Update the `ParsedPage` model in `src/models.rs`
3. Add tests in `src/parser/tests.rs`
4. Update the main parser in `src/parser/mod.rs`

### Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## Troubleshooting

### Common Issues

**Kafka Connection Errors**
- Verify Kafka brokers are accessible
- Check network connectivity
- Ensure topics exist and have proper permissions

**High Memory Usage**
- Reduce `MAX_CONCURRENCY` setting
- Increase `MAX_CONTENT_LENGTH` limit
- Monitor for memory leaks

**Slow Processing**
- Check Kafka consumer lag
- Monitor system resources
- Consider scaling horizontally

### Logs

Enable debug logging for troubleshooting:

```bash
export RUST_LOG=debug
cargo run
```

### Metrics

Monitor service health via metrics endpoint:

```bash
curl http://localhost:8080/metrics
```

## License

This project is part of the SneakDex search engine and is licensed under the MIT License. 