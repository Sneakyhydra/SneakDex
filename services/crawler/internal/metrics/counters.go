package metrics

import (
	// StdLib
	"sync/atomic"
)

// Increment number of inflight pages.
func (m *Metrics) IncrementInflightPages() { atomic.AddInt64(&m.InflightPages, 1) }

// Decrement number of inflight pages.
func (m *Metrics) DecrementInflightPages() { atomic.AddInt64(&m.InflightPages, -1) }

// Increment total number of pages processed by the crawler.
func (m *Metrics) IncrementPagesProcessed() { atomic.AddInt64(&m.PagesProcessed, 1) }

// Increment pages processed successfully.
func (m *Metrics) IncrementPagesSuccessful() { atomic.AddInt64(&m.PagesSuccessful, 1) }

// Increment pages that failed to process.
func (m *Metrics) IncrementPagesFailed() { atomic.AddInt64(&m.PagesFailed, 1) }

// Increment pages that were successfully sent to Kafka.
func (m *Metrics) IncrementKafkaSuccessful() { atomic.AddInt64(&m.KafkaSuccessful, 1) }

// Increment pages that failed to send to Kafka due to conditions (e.g., msg too large).
func (m *Metrics) IncrementKafkaFailed() { atomic.AddInt64(&m.KafkaFailed, 1) }

// Increment pages that errored while sending to Kafka (e.g., connection issues).
func (m *Metrics) IncrementKafkaErrored() { atomic.AddInt64(&m.KafkaErrored, 1) }

// Increment successful Redis checks or operations.
func (m *Metrics) IncrementRedisSuccessful() { atomic.AddInt64(&m.RedisSuccessful, 1) }

// Increment Redis operations that failed (e.g., not found in queue).
func (m *Metrics) IncrementRedisFailed() { atomic.AddInt64(&m.RedisFailed, 1) }

// Increment Redis operations that errored (e.g., connection issues).
func (m *Metrics) IncrementRedisErrored() { atomic.AddInt64(&m.RedisErrored, 1) }
