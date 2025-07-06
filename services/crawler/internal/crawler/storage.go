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

// initializeRedis sets up the Redis client and attempts connection with exponential backoff.
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

// checkRedisSet checks whether a URL exists in the specified Redis set with retry logic.
func (c *Crawler) checkRedisSet(setKey, url string) (bool, error) {
	var lastErr error
	for attempt := 1; attempt <= c.Cfg.RedisRetryMax; attempt++ {
		var err error
		var exists bool

		if setKey == "crawler:visited_urls" {
			exists, err = c.IsVisited(url)
		} else if setKey == "crawler:pending_urls_set" {
			exists, err = c.IsPending(url)
		} else if setKey == "crawler:requeued_urls" {
			exists, err = c.IsRequeued(url)
		}

		if err == nil {
			if exists {
				c.Stats.IncrementRedisSuccessful()
			}
			c.Stats.IncrementRedisFailed()
			return exists, nil
		}

		lastErr = err
		c.Log.Warnf("Redis SIsMember error (set: %s, url: %s, attempt: %d/%d): %v",
			setKey, url, attempt, c.Cfg.RedisRetryMax, err)

		if attempt < c.Cfg.RedisRetryMax {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return false, fmt.Errorf("checkRedisSet failed after %d attempts for set %s: %w", c.Cfg.RedisRetryMax, setKey, lastErr)
}

func (c *Crawler) isURLSeen(url string) (bool, error) {
	// Check comprehensive local cache first - this should catch 95%+ of lookups
	if _, exists := c.SeenLocal.Load(url); exists {
		return true, nil
	}
	
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
		// On Redis error, mark as seen locally and assume it's new
		c.SeenLocal.Store(url, struct{}{})
		return false, nil
	}
	
	// Mark as seen locally to prevent future Redis checks
	c.SeenLocal.Store(url, struct{}{})
	
	visited, err := visitedCmd.Result()
	if err == nil && visited {
		c.AddToVisitedLocal(url)
		c.Stats.IncrementRedisSuccessful()
		return true, nil
	}
	
	pending, err := pendingCmd.Result()
	if err == nil && pending {
		c.Stats.IncrementRedisSuccessful()
		return true, nil
	}
	
	c.Stats.IncrementRedisSuccessful()
	return false, nil
}

func (c *Crawler) isURLRequeued(url string) (bool, error) {
	// Check in-memory local cache first
	if _, exists := c.Requeued.Load(url); exists {
		c.Log.WithField("url", url).Trace("URL found in local cache")
		return true, nil
	}
	// Check requeued set in Redis
	if visited, err := c.checkRedisSet("crawler:requeued_urls", url); err != nil {
		return false, fmt.Errorf("failed checking requeued_urls in Redis: %w", err)
	} else if visited {
		c.AddToRequeuedLocal(url)
		return true, nil
	}
	c.Log.WithField("url", url).Trace("URL not found in Redis; treated as new")
	return false, nil
}

func (c *Crawler) MarkVisited(url string) {
	c.AddToVisited(url)
	c.AddToVisitedLocal(url)
	c.SeenLocal.Store(url, struct{}{}) // Mark as seen locally
}

func (c *Crawler) AddToVisitedLocal(url string) {
	c.Visited.Store(url, struct{}{})
}

func (c *Crawler) AddToVisited(url string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	if err := c.RedisClient.SAdd(ctx, "crawler:visited_urls", url).Err(); err != nil {
		c.Log.WithField("url", url).WithError(err).Error("Failed to add to visited_urls")
		c.Stats.IncrementRedisErrored()
	}
}

func (c *Crawler) RemoveFromVisited(url string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	if err := c.RedisClient.SRem(ctx, "crawler:visited_urls", url).Err(); err != nil {
		c.Log.WithField("url", url).WithError(err).Error("Failed to remove from visited_urls")
		c.Stats.IncrementRedisErrored()
	}
}

func (c *Crawler) IsVisited(url string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	exists, err := c.RedisClient.SIsMember(ctx, "crawler:visited_urls", url).Result()
	if err != nil {
		c.Log.WithField("url", url).WithError(err).Error("Failed to check visited_urls")
		c.Stats.IncrementRedisErrored()
		return false, err
	}
	return exists, nil
}

func (c *Crawler) AddToRequeuedLocal(url string) {
	c.Requeued.Store(url, struct{}{})
}

func (c *Crawler) AddToRequeued(url string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	if err := c.RedisClient.SAdd(ctx, "crawler:requeued_urls", url).Err(); err != nil {
		c.Log.WithField("url", url).WithError(err).Error("Failed to add to requeued_urls")
		c.Stats.IncrementRedisErrored()
	}
}

func (c *Crawler) RemoveFromRequeuedLocal(url string) {
	c.Requeued.Delete(url)
}

func (c *Crawler) RemoveFromRequeued(url string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	if err := c.RedisClient.SRem(ctx, "crawler:requeued_urls", url).Err(); err != nil {
		c.Log.WithField("url", url).WithError(err).Error("Failed to remove from requeued_urls")
		c.Stats.IncrementRedisErrored()
	}
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
		return // Skip if already in pending
	}
	
	// Check if already visited locally
	if _, exists := c.Visited.Load(url); exists {
		return // Skip if already visited
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

func (c *Crawler) IsPending(url string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	exists, err := c.RedisClient.SIsMember(ctx, "crawler:pending_urls_set", url).Result()
	if err != nil {
		c.Log.WithField("url", url).WithError(err).Error("Failed to check pending_urls_set")
		c.Stats.IncrementRedisErrored()
		return false, err
	}
	return exists, nil
}

// preloadLocalCaches loads existing URLs from Redis into local caches to minimize future Redis calls
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
			c.SeenLocal.Store(url, struct{}{})
			c.Visited.Store(url, struct{}{})
		}
		c.Log.Infof("Preloaded %d visited URLs into local cache", len(visitedURLs))
	}
	
	// Load pending URLs (sample to avoid memory issues)
	pendingURLs, err := c.RedisClient.SRandMemberN(ctx, "crawler:pending_urls_set", 5000).Result()
	if err != nil && err != redis.Nil {
		c.Log.WithError(err).Warn("Failed to preload pending URLs")
	} else {
		for _, url := range pendingURLs {
			c.SeenLocal.Store(url, struct{}{})
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
			c.SeenLocal.Store(url, struct{}{})
			c.Requeued.Store(url, struct{}{})
		}
		c.Log.Infof("Preloaded %d requeued URLs into local cache", len(requeuedURLs))
	}
	
	c.Log.Info("Local cache preloading completed")
}
