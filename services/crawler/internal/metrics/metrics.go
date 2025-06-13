package metrics

import (
	"sync/atomic"
	"time"
)

// Metrics holds counters for crawler performance statistics.
type Metrics struct {
	PagesProcessed  int64
	PagesSuccessful int64
	PagesFailed     int64
	KafkaSuccessful int64
	KafkaFailed     int64
	RedisSuccessful int64
	RedisFailed     int64

	startTime time.Time
}

// NewMetrics creates and initializes a new Metrics instance with current time.
func NewMetrics() *Metrics {
	return &Metrics{
		startTime: time.Now(),
	}
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
		"redis_successful": atomic.LoadInt64(&m.RedisSuccessful),
		"redis_failed":     atomic.LoadInt64(&m.RedisFailed),
		"uptime_seconds":   m.Uptime().Seconds(),
	}
}
