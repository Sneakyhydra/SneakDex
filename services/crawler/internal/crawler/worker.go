package crawler

import "sync/atomic"

// IncrementInFlightPages increments the count of in-flight pages being processed
func (crawler *Crawler) IncrementInFlightPages() { atomic.AddInt64(&crawler.inFlightPages, 1) }

// DecrementInFlightPages decrements the count of in-flight pages being processed
func (crawler *Crawler) DecrementInFlightPages() { atomic.AddInt64(&crawler.inFlightPages, -1) }

// GetInFlightPages returns the current number of in-flight pages being processed
func (crawler *Crawler) GetInFlightPages() int64 { return atomic.LoadInt64(&crawler.inFlightPages) }
