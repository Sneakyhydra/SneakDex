package crawler

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/sneakyhydra/sneakdex/crawler/internal/config"
	"github.com/sneakyhydra/sneakdex/crawler/internal/logger"
)

// initializeRedis sets up the Redis client and attempts connection with exponential backoff.
func (crawler *Crawler) initializeRedis() error {
	log := logger.GetLogger()
	cfg := config.GetConfig()

	redisAddr := fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort)

	// Initialize Redis client
	crawler.redisClient = redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		DialTimeout:  cfg.RedisTimeout,
		ReadTimeout:  cfg.RedisTimeout,
		WriteTimeout: cfg.RedisTimeout,
		MaxRetries:   cfg.RedisRetryMax,
	})

	// Attempt to connect with retries
	for attempt := 1; attempt <= cfg.RedisRetryMax; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RedisTimeout)
		err := crawler.redisClient.Ping(ctx).Err()
		cancel()

		if err == nil {
			log.Infof("Redis connection established at %s", redisAddr)
			return nil
		}

		log.Warnf("Redis connection attempt %d/%d failed: %v", attempt, cfg.RedisRetryMax, err)

		if attempt < cfg.RedisRetryMax {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed to connect to Redis after %d attempts (addr: %s)", cfg.RedisRetryMax, redisAddr)
}

// checkRedisSet checks whether a URL exists in the specified Redis set with retry logic.
func (crawler *Crawler) checkRedisSet(setKey, url string) (bool, error) {
	log := logger.GetLogger()
	cfg := config.GetConfig()

	var lastErr error
	for attempt := 1; attempt <= cfg.RedisRetryMax; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RedisTimeout)

		exists, err := crawler.redisClient.SIsMember(ctx, setKey, url).Result()
		cancel()

		if err == nil {
			if exists {
				crawler.stats.IncrementRedisSuccessful()
			}
			return exists, nil
		}

		lastErr = err
		log.Warnf("Redis SIsMember error (set: %s, url: %s, attempt: %d/%d): %v",
			setKey, url, attempt, cfg.RedisRetryMax, err)

		if attempt < cfg.RedisRetryMax {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	crawler.stats.IncrementRedisFailed()
	return false, fmt.Errorf("checkRedisSet failed after %d attempts for set %s: %w", cfg.RedisRetryMax, setKey, lastErr)
}

// cacheAndLogFound updates the local visited cache and logs the discovery.
func (crawler *Crawler) cacheAndLogFound(url, setType string) {
	log := logger.GetLogger()

	crawler.visited.Store(url, struct{}{})

	log.WithFields(logrus.Fields{
		"url":      url,
		"set_type": setType,
	}).Trace("URL found in Redis set and cached locally")
}

// isURLVisited checks if a URL has already been processed (in visited or pending sets).
func (crawler *Crawler) isURLVisited(url string) (bool, error) {
	log := logger.GetLogger()

	// Check in-memory local cache first
	if _, exists := crawler.visited.Load(url); exists {
		log.WithField("url", url).Trace("URL found in local cache")
		return true, nil
	}

	// Check visited set in Redis
	if visited, err := crawler.checkRedisSet("crawler:visited_urls", url); err != nil {
		return false, fmt.Errorf("failed checking visited_urls in Redis: %w", err)
	} else if visited {
		crawler.cacheAndLogFound(url, "visited")
		return true, nil
	}

	// Check pending set in Redis
	if pending, err := crawler.checkRedisSet("crawler:pending_urls", url); err != nil {
		return false, fmt.Errorf("failed checking pending_urls in Redis: %w", err)
	} else if pending {
		crawler.cacheAndLogFound(url, "pending")
		return true, nil
	}

	log.WithField("url", url).Trace("URL not found in Redis; treated as new")
	return false, nil
}
