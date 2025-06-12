package metrics

import (
	"sync/atomic"
	"time"
)

// Metrics holds crawler metrics
type Metrics struct {
	PagesProcessed  int64
	PagesSuccessful int64
	PagesFailed     int64
	KafkaSuccessful int64
	KafkaFailed     int64
	RedisSuccessful int64
	RedisFailed     int64
	StartTime       time.Time
}

// Uptime returns the duration since the crawler started.
func (metric *Metrics) Uptime() time.Duration { return time.Since(metric.StartTime) }

// GetStats returns a map of crawler statistics
func (metric *Metrics) GetStats() map[string]any {
	return map[string]any{
		"pages_processed":  atomic.LoadInt64(&metric.PagesProcessed),
		"pages_successful": atomic.LoadInt64(&metric.PagesSuccessful),
		"pages_failed":     atomic.LoadInt64(&metric.PagesFailed),
		"kafka_successful": atomic.LoadInt64(&metric.KafkaSuccessful),
		"kafka_failed":     atomic.LoadInt64(&metric.KafkaFailed),
		"redis_successful": atomic.LoadInt64(&metric.RedisSuccessful),
		"redis_failed":     atomic.LoadInt64(&metric.RedisFailed),
		"uptime_seconds":   metric.Uptime().Seconds(),
	}
}
