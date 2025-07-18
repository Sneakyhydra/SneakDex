// Package metrics provides comprehensive performance monitoring and statistics collection
// for the web crawler. It implements thread-safe counters for various crawler operations
// and integrates with Prometheus for external monitoring and alerting.
//
// The metrics system tracks:
//   - Page processing statistics (total, successful, failed)
//   - Kafka message delivery statistics (successful, failed, errored)
//   - Redis operation statistics (successful, failed, errored)
//   - System uptime and performance ratios
//
// All counters use atomic operations for thread safety and are exposed both as
// internal statistics (via GetStats) and as Prometheus metrics for integration
// with monitoring infrastructure.
//
// Example usage:
//
//	metrics := NewMetrics()
//	metrics.IncrementPagesProcessed()
//	stats := metrics.GetStats() // Get current statistics
//	metrics.SyncPrometheusMetrics() // Update Prometheus gauges
package metrics

import (
	// StdLib
	"time"

	// Third-party
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds counters for crawler performance statistics.
type Metrics struct {
	InflightPages   int64 // Number of pages currently being processed.
	PagesProcessed  int64 // Total number of pages processed in the HTMLHandler.
	PagesSuccessful int64 // Number of pages processed successfully.
	PagesFailed     int64 // Number of pages that failed to process.
	KafkaSuccessful int64 // Number of pages successfully sent to Kafka.
	KafkaFailed     int64 // Number of pages that failed to send to Kafka because of conditions (e.g., msg too large).
	KafkaErrored    int64 // Number of pages that errored while sending to Kafka (e.g., connection issues).
	RedisSuccessful int64 // Number of successful Redis checks or operations.
	RedisFailed     int64 // Number of Redis operations that failed (e.g., not found in queue).
	RedisErrored    int64 // Number of Redis operations that errored (e.g., connection issues).

	startTime time.Time // startTime records the time when the Metrics instance was created.

	// Prometheus metrics
	inflightPagesGauge prometheus.Gauge

	pagesProcessedGauge  prometheus.Gauge
	pagesSuccessfulGauge prometheus.Gauge
	pagesFailedGauge     prometheus.Gauge

	kafkaSuccessfulGauge prometheus.Gauge
	kafkaFailedGauge     prometheus.Gauge
	kafkaErroredGauge    prometheus.Gauge

	redisSuccessfulGauge prometheus.Gauge
	redisFailedGauge     prometheus.Gauge
	redisErroredGauge    prometheus.Gauge

	uptimeGauge prometheus.Gauge
}

// NewMetrics creates and initializes a new Metrics instance with current time and registers all Prometheus gauges.
func NewMetrics() *Metrics {
	m := &Metrics{
		startTime: time.Now(),

		inflightPagesGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_inflight_pages",
			Help: "Number of pages currently being processed",
		}),
		pagesProcessedGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_pages_processed_total",
			Help: "Total number of pages processed",
		}),
		pagesSuccessfulGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_pages_successful_total",
			Help: "Total number of pages successfully processed",
		}),
		pagesFailedGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_pages_failed_total",
			Help: "Total number of pages failed",
		}),
		kafkaSuccessfulGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_kafka_successful_total",
			Help: "Successful Kafka messages sent",
		}),
		kafkaFailedGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_kafka_failed_total",
			Help: "Failed Kafka messages",
		}),
		kafkaErroredGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_kafka_errored_total",
			Help: "Errored Kafka messages",
		}),
		redisSuccessfulGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_redis_successful_total",
			Help: "Successful Redis writes",
		}),
		redisFailedGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_redis_failed_total",
			Help: "Failed Redis writes",
		}),
		redisErroredGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_redis_errored_total",
			Help: "Errored Redis writes",
		}),
		uptimeGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_uptime_seconds",
			Help: "Crawler uptime in seconds",
		}),
	}

	// Register all metrics
	prometheus.MustRegister(
		m.inflightPagesGauge,
		m.pagesProcessedGauge,
		m.pagesSuccessfulGauge,
		m.pagesFailedGauge,
		m.kafkaSuccessfulGauge,
		m.kafkaFailedGauge,
		m.kafkaErroredGauge,
		m.redisSuccessfulGauge,
		m.redisFailedGauge,
		m.redisErroredGauge,
		m.uptimeGauge,
	)

	return m
}

// Uptime returns the time elapsed in seconds since the Metrics was initialized.
func (m *Metrics) Uptime() float64 {
	return time.Since(m.startTime).Seconds()
}

// GetStats returns a snapshot of crawler metrics in a map format.
func (m *Metrics) GetStats() map[string]any {
	return map[string]any{
		"inflight_pages":   m.GetInflightPages(),
		"pages_processed":  m.GetPagesProcessed(),
		"pages_successful": m.GetPagesSuccessful(),
		"pages_failed":     m.GetPagesFailed(),
		"kafka_successful": m.GetKafkaSuccessful(),
		"kafka_failed":     m.GetKafkaFailed(),
		"kafka_errored":    m.GetKafkaErrored(),
		"redis_successful": m.GetRedisSuccessful(),
		"redis_failed":     m.GetRedisFailed(),
		"redis_errored":    m.GetRedisErrored(),
		"uptime_seconds":   m.Uptime(),
	}
}

// SyncPrometheusMetrics updates the Prometheus gauges with the current metrics values.
// This function should be called periodically to ensure that Prometheus metrics are up-to-date.
func (m *Metrics) SyncPrometheusMetrics() {
	m.inflightPagesGauge.Set(float64(m.GetInflightPages()))

	m.pagesProcessedGauge.Set(float64(m.GetPagesProcessed()))
	m.pagesSuccessfulGauge.Set(float64(m.GetPagesSuccessful()))
	m.pagesFailedGauge.Set(float64(m.GetPagesFailed()))

	m.kafkaSuccessfulGauge.Set(float64(m.GetKafkaSuccessful()))
	m.kafkaFailedGauge.Set(float64(m.GetKafkaFailed()))
	m.kafkaErroredGauge.Set(float64(m.GetKafkaErrored()))

	m.redisSuccessfulGauge.Set(float64(m.GetRedisSuccessful()))
	m.redisFailedGauge.Set(float64(m.GetRedisFailed()))
	m.redisErroredGauge.Set(float64(m.GetRedisErrored()))

	m.uptimeGauge.Set(m.Uptime())
}
