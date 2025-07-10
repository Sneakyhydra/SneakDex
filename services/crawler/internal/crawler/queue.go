package crawler

import (
	// Stdlib
	"sync/atomic"
	"time"

	// Third-party
	"github.com/gocolly/colly/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// feedCollyFromRedisQueue continuously feeds URLs from the Redis pending queue to the Colly collector.
func (c *Crawler) feedCollyFromRedisQueue(collector *colly.Collector, doneChan chan struct{}) {
	defer c.Wg.Done()
	defer close(doneChan)

	c.Log.Info("Starting Redis queue feeder goroutine")

	emptyQueueChecks := 0
	const maxEmptyChecks = 5

	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.Ctx.Done():
			c.Log.Info("Redis queue feeder stopping due to context cancellation")
			collector.Wait()
			return
		case <-ticker.C:
			// Check if page processing limit is reached
			if atomic.LoadInt64(&c.Stats.PagesProcessed) >= c.Cfg.MaxPages {
				c.Log.Info("Max page limit reached, stopping Redis queue feeder")
				return
			}

			// Check concurrency limits before processing
			if c.GetInFlightPages() >= int64(c.Cfg.MaxConcurrency) {
				time.Sleep(5 * time.Millisecond) // Brief pause if at capacity
				continue
			}

			url, err := c.RemoveFromPending()
			if err == redis.Nil {
				emptyQueueChecks++
				c.Log.WithField("empty_checks", emptyQueueChecks).Debug("No URLs in Redis pending queue")

				// Stop if queue is empty for prolonged checks and no active work remains
				if emptyQueueChecks >= maxEmptyChecks {
					if c.GetInFlightPages() == 0 {
						c.Log.Info("Queue is consistently empty and no pages being processed. Exiting feeder.")
						return
					}
					c.Log.Debug("Queue still empty, but conditions not met to terminate. Retrying...")
				}
				continue
			} else if err != nil {
				c.Log.WithError(err).Error("Redis error while popping URL from pending queue")
				continue
			}

			emptyQueueChecks = 0 // Reset counter on successful fetch
			c.Log.WithField("url", url).Debug("Dispatching URL from Redis queue to Colly")

			// Visit URL using Colly (non-blocking due to Colly's internal concurrency)
			if err := collector.Visit(url); err != nil {
				c.Log.WithFields(logrus.Fields{
					"url":   url,
					"error": err,
				}).Warn("Colly failed to initiate visit, marking URL as visited to avoid requeue")

				c.MarkVisited(url)
				c.Stats.IncrementPagesFailed()
			}
		}
	}
}
