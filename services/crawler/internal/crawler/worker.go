package crawler

import (
	// Stdlib
	"sync/atomic"
)

// IncrementInFlightPages increments the count of in-flight pages being processed
func (c *Crawler) IncrementInFlightPages() { atomic.AddInt64(&c.InFlightPages, 1) }

// DecrementInFlightPages decrements the count of in-flight pages being processed
func (c *Crawler) DecrementInFlightPages() { atomic.AddInt64(&c.InFlightPages, -1) }

// GetInFlightPages returns the current number of in-flight pages being processed
func (c *Crawler) GetInFlightPages() int64 { return atomic.LoadInt64(&c.InFlightPages) }
