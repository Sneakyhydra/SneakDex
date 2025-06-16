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

// feedCollyFromRedisQueue continuously feeds URLs from the Redis pending queue to the Colly collector.
func (crawler *Crawler) feedCollyFromRedisQueue(collector *colly.Collector) {
	log := logger.GetLogger()
	cfg := config.GetConfig()

	defer wg.Done() // Mark this goroutine as done when exiting
	log.Info("Starting Redis queue feeder goroutine")

	ticker := time.NewTicker(200 * time.Millisecond)
	waitTicker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	defer waitTicker.Stop()

	emptyQueueChecks := 0
	const maxEmptyChecks = 5

	for {
		select {
		case <-ctx.Done():
			log.Info("Redis queue feeder stopping due to context cancellation")
			collector.Wait()
			return

		case <-waitTicker.C:
			collector.Wait()

		case <-ticker.C:
			// Check if page processing limit is reached
			if atomic.LoadInt64(&crawler.stats.PagesProcessed) >= cfg.MaxPages {
				log.Debug("Max page limit reached, stopping Redis queue feeder")
				return
			}

			// Pop a URL from Redis pending queue
			url, err := crawler.redisClient.SPop(ctx, "crawler:pending_urls").Result()
			if err == redis.Nil {
				emptyQueueChecks++
				log.WithField("empty_checks", emptyQueueChecks).Trace("No URLs in Redis pending queue")

				// Stop if queue is empty for prolonged checks and no active work remains
				if emptyQueueChecks >= maxEmptyChecks {
					if atomic.LoadInt64(&crawler.stats.PagesProcessed) >= cfg.MaxPages || ctx.Err() != nil || crawler.GetInFlightPages() == 0 {
						log.Info("Queue is consistently empty and termination condition met. Exiting feeder.")
						return
					}
					log.Debug("Queue still empty, but conditions not met to terminate. Retrying...")
				}
				continue
			} else if err != nil {
				log.WithError(err).Error("Redis error while popping URL from pending queue")
				crawler.stats.IncrementRedisFailed()
				continue
			}

			emptyQueueChecks = 0 // Reset counter on successful fetch
			log.WithField("url", url).Debug("Dispatching URL from Redis queue to Colly")

			// Visit URL using Colly
			if err := collector.Visit(url); err != nil {
				log.WithFields(logrus.Fields{
					"url":   url,
					"error": err,
				}).Warn("Colly failed to initiate visit, marking URL as visited to avoid requeue")

				if _, redisErr := crawler.redisClient.SAdd(ctx, "crawler:visited_urls", url).Result(); redisErr != nil {
					log.WithFields(logrus.Fields{
						"url":   url,
						"error": redisErr,
					}).Error("Failed to mark URL as visited after failed Colly initiation")
					crawler.stats.IncrementRedisFailed()
				}

				crawler.stats.IncrementPagesFailed()
			}

			if crawler.GetInFlightPages() == 0 {
				collector.Wait()
			}
		}
	}
}
