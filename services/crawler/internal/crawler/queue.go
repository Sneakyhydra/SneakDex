package crawler

import (
	// Stdlib
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

	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.Ctx.Done():
			c.Log.Info("Redis queue feeder stopping due to context cancellation")
			collector.Wait()
			return
		case <-ticker.C:
			// 0 means unlimited — keep crawling until the process is stopped or stores fail.
			if c.Cfg.MaxPages > 0 && c.Stats.GetPagesProcessed() >= c.Cfg.MaxPages {
				c.Log.Info("Max page limit reached, stopping Redis queue feeder")
				return
			}

			// Check concurrency limits before processing
			if c.Stats.GetInflightPages() >= int64(c.Cfg.MaxConcurrency) {
				time.Sleep(20 * time.Millisecond) // Brief pause if at capacity
				continue
			}

			item, err := c.RemoveFromPending()
			if err == redis.Nil {
				emptyQueueChecks++
				if emptyQueueChecks == 1 || emptyQueueChecks%12000 == 0 {
					c.Log.WithField("empty_checks", emptyQueueChecks).Info("Pending queue empty; waiting for more URLs")
				}
				continue
			} else if err != nil {
				c.Log.WithError(err).Error("Redis error while popping URL from pending queue")
				continue
			}

			emptyQueueChecks = 0 // Reset counter on successful fetch
			c.Log.WithField("url", item.URL).Debug("Dispatching URL from Redis queue to Colly")
			if item.Depth > c.Cfg.CrawlDepth {
				c.Log.WithFields(logrus.Fields{
					"url":   item.URL,
					"depth": item.Depth,
				}).Debug("Skipping URL due to exceeding MaxDepth")
				continue
			}

			// Visit URL using Colly (non-blocking due to Colly's internal concurrency)
			ctx := colly.NewContext()
			ctx.Put("depth", item.Depth)

			if err := collector.Request("GET", item.URL, nil, ctx, nil); err != nil {
				c.Log.WithFields(logrus.Fields{
					"url":   item.URL,
					"error": err,
				}).Warn("Colly failed to initiate visit, marking URL as visited to avoid requeue")

				c.MarkVisited(item.URL)
				c.Stats.IncrementPagesFailed()
			}
		}
	}
}
