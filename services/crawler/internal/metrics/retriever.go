package metrics

import (
	// StdLib
	"sync/atomic"
)

// GetInflightPages returns the current number of in-flight pages being processed
func (m *Metrics) GetInflightPages() int64 { return atomic.LoadInt64(&m.InflightPages) }

// Get total number of pages processed by the crawler.
func (m *Metrics) GetPagesProcessed() int64 { return atomic.LoadInt64(&m.PagesProcessed) }

// Get pages processed successfully.
func (m *Metrics) GetPagesSuccessful() int64 { return atomic.LoadInt64(&m.PagesSuccessful) }

// Get pages that failed to process.
func (m *Metrics) GetPagesFailed() int64 { return atomic.LoadInt64(&m.PagesFailed) }

// Get pages that were successfully sent to Kafka.
func (m *Metrics) GetKafkaSuccessful() int64 { return atomic.LoadInt64(&m.KafkaSuccessful) }

// Get pages that failed to send to Kafka due to conditions (e.g., msg too large).
func (m *Metrics) GetKafkaFailed() int64 { return atomic.LoadInt64(&m.KafkaFailed) }

// Get pages that errored while sending to Kafka (e.g., connection issues).
func (m *Metrics) GetKafkaErrored() int64 { return atomic.LoadInt64(&m.KafkaErrored) }

// Get successful Redis checks or operations.
func (m *Metrics) GetRedisSuccessful() int64 { return atomic.LoadInt64(&m.RedisSuccessful) }

// Get Redis operations that failed (e.g., not found in queue).
func (m *Metrics) GetRedisFailed() int64 { return atomic.LoadInt64(&m.RedisFailed) }

// Get Redis operations that errored (e.g., connection issues).
func (m *Metrics) GetRedisErrored() int64 { return atomic.LoadInt64(&m.RedisErrored) }
