package crawler

import (
	"sync/atomic"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sneakyhydra/sneakdex/crawler/internal/config"
	"github.com/sneakyhydra/sneakdex/crawler/internal/logger"
)

// feedCollyFromRedisQueue feeds URLs from the Redis queue to the Colly collector.
func (crawler *Crawler) feedCollyFromRedisQueue(collector *colly.Collector) {
	log := logger.GetLogger()
	cfg := config.GetConfig()

	defer wg.Done() // Decrement WaitGroup counter when this goroutine exits
	log.Info("Starting goroutine to feed URLs from Redis queue to Colly")

	// Ticker to periodically check the Redis queue
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	// Counter to track consecutive empty queue checks
	emptyQueueChecks := 0
	const maxEmptyChecks = 5

	for {
		select {
		case <-ctx.Done(): // Primary shutdown signal: context cancellation
			log.Info("Stopping Redis queue feeder goroutine due to context cancellation.")
			return
		case <-ticker.C:
			// Check if max pages limit is reached. If so, terminate.
			if atomic.LoadInt64(&crawler.stats.PagesProcessed) >= cfg.MaxPages {
				log.Debug("Max pages limit reached. Feeder will stop.")
				// Return immediately so we can continue our progress when we restart the crawler
				return
			}

			// Attempt to pull a URL from the 'pending' set
			url, err := crawler.redisClient.SPop(ctx, "crawler:pending_urls").Result()
			if err == redis.Nil {
				// Redis queue is empty
				emptyQueueChecks++
				log.WithFields(logrus.Fields{"empty_checks": emptyQueueChecks}).Trace("Redis pending queue empty.")

				// Check for termination conditions if queue is consistently empty
				if emptyQueueChecks >= maxEmptyChecks {
					// If queue is consistently empty AND (max pages reached OR context is cancelled)
					if atomic.LoadInt64(&crawler.stats.PagesProcessed) >= cfg.MaxPages || ctx.Err() != nil || crawler.GetInFlightPages() == 0 { // ctx.Done() always returns a channel so !=nill is always true
						log.Info("Redis pending queue consistently empty and termination condition met. Stopping feeder.")
						return
					}
					// If queue is empty but we haven't reached max pages or in-flight pages or context is not cancelled, just continue checking
					log.Debug("Redis pending queue empty, but max pages not reached. Continuing to check...")
				}
				continue
			} else if err != nil {
				// An actual error occurred with Redis (not just empty queue)
				log.WithFields(logrus.Fields{"error": err}).Error("Failed to pop URL from Redis pending queue, retrying...")
				crawler.stats.IncrementRedisFailed()
				continue
			}

			// If a URL was successfully popped, reset the empty queue counter
			emptyQueueChecks = 0

			log.WithFields(logrus.Fields{"url": url}).Debug("Pulled URL from Redis pending queue for visit")

			// Feed the URL to Colly. Colly's internal Limit rules will manage concurrency.
			if err := collector.Visit(url); err != nil {
				// This error means Colly failed to even initiate the visit so collector.onError won't be triggered (e.g., malformed URL given to Colly).
				// It's crucial to mark this URL as visited (failed) in Redis,
				// otherwise it might be re-added to pending if discovered again, leading to an infinite loop.
				log.WithFields(logrus.Fields{"url": url, "error": err}).Warn("Failed to initiate Colly visit for URL from Redis queue (e.g., invalid format). Marking as failed visited.")
				_, redisErr := crawler.redisClient.SAdd(ctx, "crawler:visited_urls", url).Result()
				if redisErr != nil {
					log.WithFields(logrus.Fields{"url": url, "error": redisErr}).Error("Failed to add URL to visited_urls Redis set after Colly visit initiation failure")
					crawler.stats.IncrementRedisFailed()
				}
				crawler.stats.IncrementPagesFailed()
			}
		}
	}
}
