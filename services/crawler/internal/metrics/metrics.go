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
	"sync/atomic"
	"time"

	// Third-party
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds counters for crawler performance statistics.
type Metrics struct {
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

		pagesProcessedGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pages_processed_total",
			Help: "Total number of pages processed",
		}),
		pagesSuccessfulGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pages_successful_total",
			Help: "Total number of pages successfully processed",
		}),
		pagesFailedGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pages_failed_total",
			Help: "Total number of pages failed",
		}),
		kafkaSuccessfulGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "kafka_successful_total",
			Help: "Successful Kafka messages sent",
		}),
		kafkaFailedGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "kafka_failed_total",
			Help: "Failed Kafka messages",
		}),
		kafkaErroredGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "kafka_errored_total",
			Help: "Errored Kafka messages",
		}),
		redisSuccessfulGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "redis_successful_total",
			Help: "Successful Redis writes",
		}),
		redisFailedGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "redis_failed_total",
			Help: "Failed Redis writes",
		}),
		redisErroredGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "redis_errored_total",
			Help: "Errored Redis writes",
		}),
		uptimeGauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_uptime_seconds",
			Help: "Crawler uptime in seconds",
		}),
	}

	// Register all metrics
	prometheus.MustRegister(
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

// Uptime returns the time elapsed since the Metrics was initialized.
func (m *Metrics) Uptime() time.Duration {
	return time.Since(m.startTime)
}

// GetStats returns a snapshot of crawler metrics in a map format.
func (m *Metrics) GetStats() map[string]any {
	return map[string]any{
		"pages_processed":  atomic.LoadInt64(&m.PagesProcessed),
		"pages_successful": atomic.LoadInt64(&m.PagesSuccessful),
		"pages_failed":     atomic.LoadInt64(&m.PagesFailed),
		"kafka_successful": atomic.LoadInt64(&m.KafkaSuccessful),
		"kafka_failed":     atomic.LoadInt64(&m.KafkaFailed),
		"kafka_errored":    atomic.LoadInt64(&m.KafkaErrored),
		"redis_successful": atomic.LoadInt64(&m.RedisSuccessful),
		"redis_failed":     atomic.LoadInt64(&m.RedisFailed),
		"redis_errored":    atomic.LoadInt64(&m.RedisErrored),
		"uptime_seconds":   m.Uptime().Seconds(),
	}
}

// SyncPrometheusMetrics updates the Prometheus gauges with the current metrics values.
// This function should be called periodically to ensure that Prometheus metrics are up-to-date.
func (m *Metrics) SyncPrometheusMetrics() {
	m.pagesProcessedGauge.Set(float64(atomic.LoadInt64(&m.PagesProcessed)))
	m.pagesSuccessfulGauge.Set(float64(atomic.LoadInt64(&m.PagesSuccessful)))
	m.pagesFailedGauge.Set(float64(atomic.LoadInt64(&m.PagesFailed)))

	m.kafkaSuccessfulGauge.Set(float64(atomic.LoadInt64(&m.KafkaSuccessful)))
	m.kafkaFailedGauge.Set(float64(atomic.LoadInt64(&m.KafkaFailed)))
	m.kafkaErroredGauge.Set(float64(atomic.LoadInt64(&m.KafkaErrored)))

	m.redisSuccessfulGauge.Set(float64(atomic.LoadInt64(&m.RedisSuccessful)))
	m.redisFailedGauge.Set(float64(atomic.LoadInt64(&m.RedisFailed)))
	m.redisErroredGauge.Set(float64(atomic.LoadInt64(&m.RedisErrored)))

	m.uptimeGauge.Set(m.Uptime().Seconds())
}
