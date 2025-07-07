package crawler

import (
	// Stdlib
	"context"
	"fmt"
	"math"
	"time"

	// Third-party
	"github.com/redis/go-redis/v9"
)

// initializeRedis establishes a connection to Redis with comprehensive retry logic and optimized timeouts.
// Redis serves as the distributed queue and deduplication store for URLs across multiple crawler instances.
// It implements exponential backoff retry strategy to handle temporary connection issues gracefully.
//
// Configuration includes:
//   - Connection timeouts for network operations
//   - Database selection for multi-tenant support
//   - Authentication if password is provided
//   - Retry limits with exponential backoff
//
// Returns an error if connection cannot be established after all retry attempts.
func (c *Crawler) initializeRedis() error {
	redisAddr := fmt.Sprintf("%s:%d", c.Cfg.RedisHost, c.Cfg.RedisPort)

	// Initialize Redis client
	c.RedisClient = redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     c.Cfg.RedisPassword,
		DB:           c.Cfg.RedisDB,
		DialTimeout:  c.Cfg.RedisTimeout,
		ReadTimeout:  c.Cfg.RedisTimeout,
		WriteTimeout: c.Cfg.RedisTimeout,
		MaxRetries:   c.Cfg.RedisRetryMax,
	})

	// Attempt to connect with retries
	for attempt := 1; attempt <= c.Cfg.RedisRetryMax; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
		defer cancel()
		err := c.RedisClient.Ping(ctx).Err()

		if err == nil {
			c.Log.Infof("Redis connection established at %s", redisAddr)
			return nil
		}

		c.Log.Warnf("Redis connection attempt %d/%d failed: %v", attempt, c.Cfg.RedisRetryMax, err)

		if attempt < c.Cfg.RedisRetryMax {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed to connect to Redis after %d attempts (addr: %s)", c.Cfg.RedisRetryMax, redisAddr)
}

// isURLSeen efficiently checks if a URL has been seen before using a multi-level caching strategy.
// This function is performance-critical as it's called for every discovered URL during crawling.
//
// Performance optimization strategy:
//  1. Local cache check
//  2. Redis pipeline query for both visited and pending sets (single round trip)
//  3. Automatic local caching of results to avoid future Redis calls
//  4. Graceful error handling with fallback behavior
//
// Returns true if URL has been seen (visited or pending), false if it's new.
func (c *Crawler) isURLSeen(url string) (bool, error) {
	// Check comprehensive local cache first
	if _, exists := c.Seen.Load(url); exists {
		return true, nil
	}
	c.Seen.Store(url, struct{}{})

	// For URLs not in local cache, do a quick Redis check (but only once per URL)
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	// Use pipeline to check both sets in a single round trip
	pipe := c.RedisClient.Pipeline()
	visitedCmd := pipe.SIsMember(ctx, "crawler:visited_urls", url)
	pendingCmd := pipe.SIsMember(ctx, "crawler:pending_urls_set", url)

	_, err := pipe.Exec(ctx)
	if err != nil {
		c.Stats.IncrementRedisErrored()
		return false, nil
	}

	visited, err := visitedCmd.Result()
	if err == nil && visited {
		c.Stats.IncrementRedisSuccessful()
		return true, nil
	}

	pending, err := pendingCmd.Result()
	if err == nil && pending {
		c.Stats.IncrementRedisSuccessful()
		return true, nil
	}

	c.Stats.IncrementRedisFailed()
	return false, nil
}

func (c *Crawler) isURLRequeued(url string) (bool, error) {
	// Check in-memory local cache first
	if _, exists := c.Requeued.Load(url); exists {
		c.Log.WithField("url", url).Trace("URL found in local cache")
		return true, nil
	}

	// Check requeued set
	if requeued, err := c.IsRequeued(url); err != nil {
		return false, fmt.Errorf("failed checking requeued_urls: %w", err)
	} else if requeued {
		c.RemoveFromRequeued(url)
		return true, nil
	}

	c.Log.WithField("url", url).Trace("URL not found in requeued set; treated as new")
	return false, nil
}

func (c *Crawler) MarkVisited(url string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	if err := c.RedisClient.SAdd(ctx, "crawler:visited_urls", url).Err(); err != nil {
		c.Log.WithField("url", url).WithError(err).Error("Failed to add to visited_urls")
		c.Stats.IncrementRedisErrored()
	}

	c.Seen.Store(url, struct{}{})
}

func (c *Crawler) AddToRequeued(url string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	c.Requeued.Store(url, struct{}{})
	if err := c.RedisClient.SAdd(ctx, "crawler:requeued_urls", url).Err(); err != nil {
		c.Log.WithField("url", url).WithError(err).Error("Failed to add to requeued_urls")
		c.Stats.IncrementRedisErrored()
	}
}

func (c *Crawler) RemoveFromRequeued(url string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	if err := c.RedisClient.SRem(ctx, "crawler:requeued_urls", url).Err(); err != nil {
		c.Log.WithField("url", url).WithError(err).Error("Failed to remove from requeued_urls")
		c.Stats.IncrementRedisErrored()
	}

	c.Requeued.Delete(url)
}

func (c *Crawler) IsRequeued(url string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	exists, err := c.RedisClient.SIsMember(ctx, "crawler:requeued_urls", url).Result()
	if err != nil {
		c.Log.WithField("url", url).WithError(err).Error("Failed to check requeued_urls")
		c.Stats.IncrementRedisErrored()
		return false, err
	}
	return exists, nil
}

func (c *Crawler) AddToPending(url string) {
	// Check if already in pending locally first (to avoid duplicate Redis calls)
	if _, exists := c.Pending.Load(url); exists {
		return
	}

	// Mark as pending locally
	c.Pending.Store(url, struct{}{})

	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	// Try to add to set first
	added, err := c.RedisClient.SAdd(ctx, "crawler:pending_urls_set", url).Result()
	if err != nil {
		c.Log.WithField("url", url).WithError(err).Error("Failed to add to pending_urls_set")
		c.Stats.IncrementRedisErrored()
		return
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel2()
	// Only push to queue if it wasn't already in the set
	if added == 1 {
		if err := c.RedisClient.RPush(ctx2, "crawler:pending_urls", url).Err(); err != nil {
			c.Log.WithField("url", url).WithError(err).Error("Failed to enqueue to pending_urls")
			c.Stats.IncrementRedisErrored()
			// Optionally remove from set on failure to keep consistency
			ctx3, cancel3 := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
			defer cancel3()
			_ = c.RedisClient.SRem(ctx3, "crawler:pending_urls_set", url).Err()
		}
	}
}

func (c *Crawler) RemoveFromPending() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	url, err := c.RedisClient.LPop(ctx, "crawler:pending_urls").Result()
	if err == redis.Nil {
		return "", redis.Nil
	}
	if err != nil {
		c.Log.WithError(err).Error("Failed to dequeue from pending_urls")
		c.Stats.IncrementRedisErrored()
		return "", err
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel2()
	// Remove from set to keep consistency
	if err := c.RedisClient.SRem(ctx2, "crawler:pending_urls_set", url).Err(); err != nil {
		c.Log.WithField("url", url).WithError(err).Error("Failed to remove from pending_urls_set")
		c.Stats.IncrementRedisErrored()
	}

	// Remove from local pending cache
	c.Pending.Delete(url)

	return url, nil
}

// preloadLocalCaches optimizes crawler performance by bulk-loading existing URLs from Redis into local memory.
// This is a startup optimization that dramatically reduces Redis queries during crawling by pre-populating
// the local caches with a representative sample of already-processed URLs.
//
// Cache preloading strategy:
//   - Visited URLs: Sample 10,000 random URLs to avoid memory overflow while maximizing cache hits
//   - Pending URLs: Sample 5,000 random URLs from the pending set
//   - Requeued URLs: Load all requeued URLs (typically a smaller set)
//
// This function significantly improves crawling performance by reducing the "cold start" effect
// where many Redis queries would be needed to rebuild the local cache organically.
func (c *Crawler) preloadLocalCaches() {
	c.Log.Info("Preloading local caches from Redis to minimize future Redis calls...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Longer timeout for bulk operation
	defer cancel()

	// Load visited URLs (sample to avoid memory issues)
	visitedURLs, err := c.RedisClient.SRandMemberN(ctx, "crawler:visited_urls", 10000).Result()
	if err != nil && err != redis.Nil {
		c.Log.WithError(err).Warn("Failed to preload visited URLs")
	} else {
		for _, url := range visitedURLs {
			c.Seen.Store(url, struct{}{})
		}
		c.Log.Infof("Preloaded %d visited URLs into local cache", len(visitedURLs))
	}

	// Load pending URLs (sample to avoid memory issues)
	pendingURLs, err := c.RedisClient.SRandMemberN(ctx, "crawler:pending_urls_set", 5000).Result()
	if err != nil && err != redis.Nil {
		c.Log.WithError(err).Warn("Failed to preload pending URLs")
	} else {
		for _, url := range pendingURLs {
			c.Seen.Store(url, struct{}{})
			c.Pending.Store(url, struct{}{})
		}
		c.Log.Infof("Preloaded %d pending URLs into local cache", len(pendingURLs))
	}

	// Load requeued URLs (all of them since there should be fewer)
	requeuedURLs, err := c.RedisClient.SMembers(ctx, "crawler:requeued_urls").Result()
	if err != nil && err != redis.Nil {
		c.Log.WithError(err).Warn("Failed to preload requeued URLs")
	} else {
		for _, url := range requeuedURLs {
			c.Seen.Store(url, struct{}{})
			c.Requeued.Store(url, struct{}{})
		}
		c.Log.Infof("Preloaded %d requeued URLs into local cache", len(requeuedURLs))
	}

	c.Log.Info("Local cache preloading completed")
}
