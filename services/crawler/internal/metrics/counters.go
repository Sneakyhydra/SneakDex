package metrics

import "sync/atomic"

// IncrementMetrics provides atomic increment methods for crawler metrics
func (metric *Metrics) IncrementPagesProcessed()  { atomic.AddInt64(&metric.PagesProcessed, 1) }
func (metric *Metrics) IncrementPagesSuccessful() { atomic.AddInt64(&metric.PagesSuccessful, 1) }
func (metric *Metrics) IncrementPagesFailed()     { atomic.AddInt64(&metric.PagesFailed, 1) }
func (metric *Metrics) IncrementKafkaSuccessful() { atomic.AddInt64(&metric.KafkaSuccessful, 1) }
func (metric *Metrics) IncrementKafkaFailed()     { atomic.AddInt64(&metric.KafkaFailed, 1) }
func (metric *Metrics) IncrementRedisSuccessful() { atomic.AddInt64(&metric.RedisSuccessful, 1) }
func (metric *Metrics) IncrementRedisFailed()     { atomic.AddInt64(&metric.RedisFailed, 1) }
