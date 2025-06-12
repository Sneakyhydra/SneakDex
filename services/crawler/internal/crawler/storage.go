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

// initializeRedis sets up Redis client with proper configuration
func (crawler *Crawler) initializeRedis() error {
	log := logger.GetLogger()
	cfg := config.GetConfig()
	// Construct the Redis address from host and port
	redisAddr := fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort)

	// Create a new Redis client with the provided configuration
	crawler.redisClient = redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		DialTimeout:  cfg.RedisTimeout,
		ReadTimeout:  cfg.RedisTimeout,
		WriteTimeout: cfg.RedisTimeout,
		MaxRetries:   cfg.RedisRetryMax,
	})

	// Test connection with retries
	for attempt := 1; attempt <= cfg.RedisRetryMax; attempt++ {
		ctx, cancel := context.WithTimeout(ctx, cfg.RedisTimeout)
		err := crawler.redisClient.Ping(ctx).Err()
		cancel()

		if err == nil {
			log.Info("Redis connection established")
			return nil
		}

		log.Warnf("Redis connection attempt %d/%d failed: %v", attempt, cfg.RedisRetryMax, err)
		if attempt < cfg.RedisRetryMax {
			// Exponential backoff for retries
			backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed to connect to redis after %d attempts. please ensure redis is running on %s", cfg.RedisRetryMax, redisAddr)
}

// checkRedisSet checks if URL exists in the specified Redis set with retry logic
func (crawler *Crawler) checkRedisSet(setKey, url string) (bool, error) {
	log := logger.GetLogger()
	cfg := config.GetConfig()
	var lastErr error

	for attempt := 1; attempt <= cfg.RedisRetryMax; attempt++ {
		ctx, cancel := context.WithTimeout(ctx, cfg.RedisTimeout)

		exists, err := crawler.redisClient.SIsMember(ctx, setKey, url).Result()
		cancel()

		if err == nil {
			if exists {
				crawler.stats.IncrementRedisSuccessful()
			}
			return exists, nil
		}

		lastErr = err
		log.Warnf("Redis SIsMember '%s' attempt %d/%d failed for URL %s: %v",
			setKey, attempt, cfg.RedisRetryMax, url, err)

		// Don't sleep on the last attempt
		if attempt < cfg.RedisRetryMax {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	crawler.stats.IncrementRedisFailed()
	return false, fmt.Errorf("failed after %d attempts: %w", cfg.RedisRetryMax, lastErr)
}

// cacheAndLogFound caches the URL locally and logs the discovery
func (crawler *Crawler) cacheAndLogFound(url, setType string) {
	log := logger.GetLogger()
	crawler.visited.Store(url, struct{}{})
	log.WithFields(logrus.Fields{
		"url":      url,
		"set_type": setType,
	}).Trace("URL found in redis")
}

// isURLVisited checks if a URL has already been visited or is pending,
// using both local memory cache and Redis sets for quick lookups.
func (crawler *Crawler) isURLVisited(url string) (bool, error) {
	log := logger.GetLogger()
	// Check local memory cache first for quick lookups
	if _, ok := crawler.visited.Load(url); ok {
		log.WithFields(logrus.Fields{"url": url}).Trace("URL found in local visited/pending cache")
		return true, nil
	}

	// Check both Redis sets with retry logic
	visited, err := crawler.checkRedisSet("crawler:visited_urls", url)
	if err != nil {
		return false, fmt.Errorf("failed to check visited urls in redis: %w", err)
	}
	if visited {
		crawler.cacheAndLogFound(url, "visited")
		return true, nil
	}

	pending, err := crawler.checkRedisSet("crawler:pending_urls", url)
	if err != nil {
		return false, fmt.Errorf("failed to check pending urls in redis: %w", err)
	}
	if pending {
		crawler.cacheAndLogFound(url, "pending")
		return true, nil
	}

	log.WithFields(logrus.Fields{"url": url}).Trace("URL not found in redis, considered new")
	return false, nil
}
