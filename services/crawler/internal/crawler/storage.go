package crawler

import (
	// Stdlib
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	// Third-party
	"github.com/redis/go-redis/v9"
)

type QueueItem struct {
	URL   string `json:"url"`
	Depth int    `json:"depth"`
}

// getQueueKey returns the Redis key for a specific depth level
func (c *Crawler) getQueueKey(depth int) string {
	return fmt.Sprintf("crawler:pending_urls:depth_%d", depth)
}

// AddToPending adds an item to the appropriate depth-based queue
func (c *Crawler) AddToPending(item QueueItem) {
	// Check if already in pending locally
	if _, exists := c.Pending.Load(item.URL); exists {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	// Add to Redis SET to deduplicate
	added, err := c.RedisClient.SAdd(ctx, "crawler:pending_urls_set", item.URL).Result()
	if err != nil {
		c.Log.WithField("url", item.URL).WithError(err).Error("Failed to add to pending_urls_set")
		c.Stats.IncrementRedisErrored()
		return
	}

	// Mark as pending locally
	c.Pending.Store(item.URL, struct{}{})

	// Only push to queue if it wasn't already in the set
	if added == 1 {
		data, err := json.Marshal(item)
		if err != nil {
			c.Log.WithField("url", item.URL).WithError(err).Error("Failed to marshal QueueItem")
			c.Stats.IncrementRedisErrored()
			return
		}

		ctx2, cancel2 := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
		defer cancel2()

		// Push to depth-specific queue
		queueKey := c.getQueueKey(item.Depth)
		if err := c.RedisClient.RPush(ctx2, queueKey, data).Err(); err != nil {
			c.Log.WithField("url", item.URL).WithField("depth", item.Depth).WithError(err).Error("Failed to enqueue to depth-specific queue")
			c.Stats.IncrementRedisErrored()
			// Optionally clean up set
			ctx3, cancel3 := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
			defer cancel3()
			_ = c.RedisClient.SRem(ctx3, "crawler:pending_urls_set", item.URL).Err()
			c.Pending.Delete(item.URL)
			return
		}
	}
}

// RemoveFromPending removes an item from the priority queue, starting with the lowest depth
func (c *Crawler) RemoveFromPending() (*QueueItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	// Try each depth level starting from 0
	maxDepth := c.Cfg.CrawlDepth

	for depth := 0; depth <= maxDepth; depth++ {
		queueKey := c.getQueueKey(depth)

		data, err := c.RedisClient.LPop(ctx, queueKey).Result()
		if err == redis.Nil {
			// No items at this depth, try next depth
			continue
		}
		if err != nil {
			c.Log.WithField("depth", depth).WithError(err).Error("Failed to dequeue from depth-specific queue")
			c.Stats.IncrementRedisErrored()
			return nil, err
		}

		var item QueueItem
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			c.Log.WithError(err).Error("Failed to unmarshal QueueItem from Redis")
			c.Stats.IncrementRedisErrored()
			return nil, err
		}

		// Remove from set
		ctx2, cancel2 := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
		defer cancel2()
		if err := c.RedisClient.SRem(ctx2, "crawler:pending_urls_set", item.URL).Err(); err != nil {
			c.Log.WithField("url", item.URL).WithError(err).Error("Failed to remove from pending_urls_set")
			c.Stats.IncrementRedisErrored()
		}

		// Remove from local pending
		c.Pending.Delete(item.URL)

		return &item, nil
	}

	// No items found at any depth level
	return nil, redis.Nil
}

// GetQueueStats returns statistics about queue depth distribution
func (c *Crawler) GetQueueStats() map[int]int64 {
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	stats := make(map[int]int64)
	maxDepth := c.Cfg.CrawlDepth // Should match the maxDepth in RemoveFromPending

	for depth := 0; depth <= maxDepth; depth++ {
		queueKey := c.getQueueKey(depth)
		length, err := c.RedisClient.LLen(ctx, queueKey).Result()
		if err != nil {
			c.Log.WithField("depth", depth).WithError(err).Warn("Failed to get queue length")
			continue
		}
		if length > 0 {
			stats[depth] = length
		}
	}

	return stats
}

// CleanupEmptyQueues removes empty depth-based queues (optional maintenance)
func (c *Crawler) CleanupEmptyQueues() {
	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	maxDepth := c.Cfg.CrawlDepth // Should match the maxDepth in RemoveFromPending

	for depth := 0; depth <= maxDepth; depth++ {
		queueKey := c.getQueueKey(depth)
		length, err := c.RedisClient.LLen(ctx, queueKey).Result()
		if err != nil {
			continue
		}
		if length == 0 {
			// Queue is empty, we can delete it
			c.RedisClient.Del(ctx, queueKey)
		}
	}
}

// preloadLocalCaches - updated to handle multiple depth queues
func (c *Crawler) preloadLocalCaches() {
	c.Log.Info("Preloading local caches from Redis to minimize future Redis calls...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

func (c *Crawler) isURLSeen(url string) (bool, error) {
	// Check local cache first
	if _, exists := c.Seen.Load(url); exists {
		return true, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Cfg.RedisTimeout)
	defer cancel()

	key := fmt.Sprintf("crawler:visited:%s", url)

	exists, err := c.RedisClient.Exists(ctx, key).Result()
	if err != nil {
		c.Log.WithField("url", url).WithError(err).Error("Failed to check visited key")
		c.Stats.IncrementRedisErrored()
		return false, err
	}

	if exists > 0 {
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

	key := fmt.Sprintf("crawler:visited:%s", url)
	ttl := 24 * time.Hour // or make configurable

	if err := c.RedisClient.Set(ctx, key, "1", ttl).Err(); err != nil {
		c.Log.WithField("url", url).WithError(err).Error("Failed to mark visited with TTL")
		c.Stats.IncrementRedisErrored()
		return
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
