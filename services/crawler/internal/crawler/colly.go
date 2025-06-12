package crawler

import (
	// StdLib
	"context"
	"errors"
	"path"
	"strings"
	"sync/atomic"

	// Third-party
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/sirupsen/logrus"

	// Internal modules
	"github.com/sneakyhydra/sneakdex/crawler/internal/config"
	"github.com/sneakyhydra/sneakdex/crawler/internal/crawlerrors"
	"github.com/sneakyhydra/sneakdex/crawler/internal/logger"
	"github.com/sneakyhydra/sneakdex/crawler/internal/utils"
	"github.com/sneakyhydra/sneakdex/crawler/internal/validator"
)

// setupCollyCollector configures the colly collector with callbacks for crawling logic.
func (crawler *Crawler) setupCollyCollector(ctx context.Context) *colly.Collector {
	log := logger.GetLogger()
	cfg := config.GetConfig()
	urlValidator := validator.GetURLValidator()

	// Setup Colly collector options
	options := []colly.CollectorOption{
		colly.MaxDepth(cfg.CrawlDepth),
		colly.Async(true), // Enable asynchronous requests, managing concurrency internally
		colly.UserAgent(cfg.UserAgent),
		colly.ParseHTTPErrorResponse(), // Parse 4xx/5xx responses for better error handling
		colly.DetectCharset(),          // Auto-detect and convert character encoding
	}

	if len(crawler.blacklist) > 0 {
		options = append(options, colly.DisallowedDomains(crawler.blacklist...))
	}
	if len(crawler.whitelist) > 0 {
		options = append(options, colly.AllowedDomains(crawler.whitelist...))
	}
	if cfg.EnableDebug {
		options = append(options, colly.Debugger(&debug.LogDebugger{}))
	}

	// Create the collector
	collector := colly.NewCollector(options...)

	// Apply rate limits for all domains
	err := collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: cfg.MaxConcurrency,
		Delay:       cfg.RequestDelay,
		RandomDelay: cfg.RequestDelay / 2,
	})
	if err != nil {
		log.WithFields(logrus.Fields{"error": err}).Error("Failed to set rate limit")
		// Continue
	}

	collector.SetRequestTimeout(cfg.RequestTimeout)

	var skipExts = map[string]struct{}{
		".pdf": {}, ".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".css": {}, ".js": {}, ".ico": {},
		".svg": {}, ".woff": {}, ".ttf": {}, ".mp4": {}, ".mp3": {}, ".zip": {}, ".exe": {},
	}

	// Request filtering and header setting
	collector.OnRequest(func(r *colly.Request) {
		// Check for context cancellation before proceeding with the request
		select {
		case <-ctx.Done():
			log.WithFields(logrus.Fields{"url": r.URL.String()}).Debug("Request aborted due to shutdown")
			r.Abort()
			return
		default:
			// Continue
		}

		urlStr := r.URL.String()
		// Skip common file extensions
		ext := strings.ToLower(path.Ext(r.URL.Path))
		if _, skip := skipExts[ext]; skip {
			log.WithFields(logrus.Fields{"url": urlStr, "ext": ext}).Debug("Skipping URL due to file extension")
			r.Abort()
			return
		}

		// Add common headers to appear more like a real browser
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Accept-Encoding", "gzip, deflate")
		r.Headers.Set("DNT", "1")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")

		crawler.IncrementInFlightPages()
		log.WithFields(logrus.Fields{"url": urlStr}).Debug("Visiting URL")
	})

	// Handle HTML pages
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("HTML processing skipped due to shutdown")
			return
		default:
			// Continue
		}

		// Stop processing if max pages limit is reached
		if atomic.LoadInt64(&crawler.stats.PagesProcessed) >= cfg.MaxPages {
			log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("Max pages limit reached, stopping HTML processing")
			return
		}

		crawler.stats.IncrementPagesProcessed()
		defer crawler.DecrementInFlightPages()
		url := e.Request.URL.String()

		html, err := e.DOM.Html()
		if err != nil {
			log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to extract HTML")
			crawler.stats.IncrementPagesFailed()
			// IMPORTANT: Mark as visited even if HTML extraction fails
			_, redisErr := crawler.redisClient.SAdd(ctx, "crawler:visited_urls", url).Result()
			if redisErr != nil {
				log.WithFields(logrus.Fields{"url": url, "error": redisErr}).Error("Failed to add URL to visited_urls Redis set after HTML extraction failure")
				crawler.stats.IncrementRedisFailed()
			}
			return
		}

		if err := crawler.sendToKafka(url, html); err != nil {
			var crawlErr *crawlerrors.CrawlError
			if errors.As(err, &crawlErr) && crawlErr.Retry {
				if _, ok := crawler.requeued.Load(url); ok {
					log.WithFields(logrus.Fields{"url": url}).Trace("URL already requeued once. Will be marked as visited")
					crawler.requeued.Delete(url)
				} else {
					// Re-queue URL instead of marking as visited
					log.WithFields(logrus.Fields{"url": url, "error": err}).Warn("Retriable error occurred, requeuing URL")

					_, redisErr := crawler.redisClient.SAdd(ctx, "crawler:pending_urls", url).Result()
					if redisErr != nil {
						log.WithFields(logrus.Fields{"url": url, "error": redisErr}).Error("Failed to requeue URL after Kafka retryable error")
						crawler.stats.IncrementRedisFailed()
					}
					crawler.requeued.Store(url, struct{}{})
					return
				}
			}

			log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to send to Kafka")
			crawler.stats.IncrementPagesFailed()
			// IMPORTANT: Mark as visited even if sending to Kafka fails
			_, redisErr := crawler.redisClient.SAdd(ctx, "crawler:visited_urls", url).Result()
			if redisErr != nil {
				log.WithFields(logrus.Fields{"url": url, "error": redisErr}).Error("Failed to add URL to visited_urls Redis set after Kafka failure")
				crawler.stats.IncrementRedisFailed()
			}
			return
		}

		crawler.stats.IncrementPagesSuccessful()
		log.WithFields(logrus.Fields{"url": url, "content_size": len(html)}).Debug("Page processed successfully")

		// IMPORTANT: Mark as visited in Redis after successful processing
		_, err = crawler.redisClient.SAdd(ctx, "crawler:visited_urls", url).Result()
		if err != nil {
			log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to add URL to visited_urls Redis set")
			crawler.stats.IncrementRedisFailed()
		}
	})

	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("Link extraction skipped due to shutdown")
			return
		default:
			// Continue
		}

		// Stop processing if max pages limit is reached
		if atomic.LoadInt64(&crawler.stats.PagesProcessed) >= cfg.MaxPages {
			log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("Max pages limit reached, stopping link extraction")
			return
		}

		link := e.Attr("href")
		if link == "" {
			return
		}

		// Skip common non-HTTP/HTTPS links or internal page anchors
		if strings.HasPrefix(link, "javascript:") ||
			strings.HasPrefix(link, "mailto:") ||
			strings.HasPrefix(link, "tel:") ||
			strings.Contains(link, "#") { // Links with fragments are typically internal page anchors
			return
		}

		absoluteURL := e.Request.AbsoluteURL(link)
		if !urlValidator.IsValidURL(absoluteURL) {
			return
		}

		normalized, err := utils.NormalizeURL(absoluteURL)
		if err != nil {
			log.WithFields(logrus.Fields{"url": absoluteURL, "error": err}).Debug("Failed to normalize URL")
			return
		}

		// Use the enhanced isURLVisited to check both 'visited' and 'pending' sets
		visitedOrPending, err := crawler.isURLVisited(normalized)
		if err != nil {
			log.WithFields(logrus.Fields{"url": normalized, "error": err}).Error("Failed to check URL visited/pending status")
			crawler.stats.IncrementRedisFailed() // Increment failed metrics if Redis check fails
			return
		}

		if !visitedOrPending {
			log.WithFields(logrus.Fields{"url": normalized}).Debug("Adding new URL to Redis pending queue")
			// Add to Redis 'pending_urls' set instead of calling Colly's Visit directly
			_, err := crawler.redisClient.SAdd(ctx, "crawler:pending_urls", normalized).Result()
			if err != nil {
				log.WithFields(logrus.Fields{"url": normalized, "error": err}).Error("Failed to add URL to Redis pending queue")
				crawler.stats.IncrementRedisFailed()
			}
		}
	})

	// Error handler for HTTP requests
	collector.OnError(func(r *colly.Response, err error) {
		// Suppress logging common timeout/connection errors if debug is not enabled,
		// as these can be noisy but are expected in network operations.
		isNetworkError := strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "no such host")

		if !isNetworkError || cfg.EnableDebug {
			log.WithFields(logrus.Fields{
				"url":         r.Request.URL.String(),
				"status_code": r.StatusCode,
				"error":       err,
			}).Warn("Request failed")
		} else {
			log.WithFields(logrus.Fields{
				"url":         r.Request.URL.String(),
				"status_code": r.StatusCode,
				"error":       err,
			}).Debug("Suppressed network error") // Log as debug if suppressed
		}
		crawler.stats.IncrementPagesFailed()

		// IMPORTANT: Mark as visited in Redis, even if the request failed, to prevent re-attempts for this URL.
		url := r.Request.URL.String()
		_, redisErr := crawler.redisClient.SAdd(ctx, "crawler:visited_urls", url).Result()
		if redisErr != nil {
			log.WithFields(logrus.Fields{"url": url, "error": redisErr}).Error("Failed to add URL to visited_urls Redis set on error")
			crawler.stats.IncrementRedisFailed()
		}
		crawler.DecrementInFlightPages()
	})

	// Debug logging on successful response
	if cfg.EnableDebug {
		collector.OnResponse(func(r *colly.Response) {
			log.WithFields(logrus.Fields{
				"url":          r.Request.URL.String(),
				"status_code":  r.StatusCode,
				"content_type": r.Headers.Get("Content-Type"),
				"size":         len(r.Body),
			}).Debug("Response received")
		})
	}

	return collector
}
