package crawler

import (
	// StdLib
	"context"
	"path"
	"strings"
	"sync/atomic"

	// Third-party
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/sirupsen/logrus"

	// Internal modules
	"github.com/sneakyhydra/sneakdex/crawler/internal/config"
	"github.com/sneakyhydra/sneakdex/crawler/internal/logger"
	"github.com/sneakyhydra/sneakdex/crawler/internal/utils"
	"github.com/sneakyhydra/sneakdex/crawler/internal/validator"
)

// isContextDone checks if the context is done without blocking
func isContextDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// setupCollyCollector initializes a Colly collector with the necessary handlers and configurations
func (crawler *Crawler) setupCollyCollector(ctx context.Context) *colly.Collector {
	collector := crawler.createBaseCollector()
	crawler.setRequestHandler(collector, ctx)
	crawler.setHTMLHandler(collector, ctx)
	crawler.setLinkHandler(collector, ctx)
	crawler.setErrorHandler(collector, ctx)
	crawler.setResponseHandler(collector, ctx)
	crawler.setResponseLogger(collector)
	return collector
}

func (crawler *Crawler) createBaseCollector() *colly.Collector {
	cfg := config.GetConfig()

	options := []colly.CollectorOption{
		colly.MaxDepth(cfg.CrawlDepth),
		colly.Async(true),
		colly.UserAgent(cfg.UserAgent),
		colly.ParseHTTPErrorResponse(),
		colly.DetectCharset(),
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

	collector := colly.NewCollector(options...)
	_ = collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: cfg.MaxConcurrency,
		Delay:       cfg.RequestDelay,
		RandomDelay: cfg.RequestDelay / 2,
	})

	collector.SetRequestTimeout(cfg.RequestTimeout)
	return collector
}

func (crawler *Crawler) setRequestHandler(collector *colly.Collector, ctx context.Context) {
	log := logger.GetLogger()
	cfg := config.GetConfig()

	var skipExts = map[string]struct{}{
		".pdf": {}, ".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".css": {}, ".js": {}, ".ico": {},
		".svg": {}, ".woff": {}, ".ttf": {}, ".mp4": {}, ".mp3": {}, ".zip": {}, ".exe": {},
	}

	collector.OnRequest(func(r *colly.Request) {
		if isContextDone(ctx) {
			log.WithFields(logrus.Fields{"url": r.URL.String()}).Debug("Request aborted due to shutdown")
			r.Abort()
			return
		}

		if atomic.LoadInt64(&crawler.stats.PagesProcessed) >= cfg.MaxPages {
			log.WithFields(logrus.Fields{"url": r.URL.String()}).Debug("Skipping due to MaxPages limit")
			r.Abort()
			return
		}

		ext := strings.ToLower(path.Ext(r.URL.Path))
		if _, skip := skipExts[ext]; skip {
			log.WithFields(logrus.Fields{"url": r.URL.String(), "ext": ext}).Debug("Skipping URL due to file extension")
			r.Abort()
			return
		}

		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Accept-Encoding", "gzip, deflate")
		r.Headers.Set("DNT", "1")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")

		crawler.IncrementInFlightPages()
		log.WithFields(logrus.Fields{"url": r.URL.String()}).Debug("Visiting URL")
	})
}

func (crawler *Crawler) setHTMLHandler(collector *colly.Collector, ctx context.Context) {
	log := logger.GetLogger()
	cfg := config.GetConfig()

	collector.OnHTML("html", func(e *colly.HTMLElement) {
		defer crawler.DecrementInFlightPages() // Ensure this is handled correctly for overall page count
		if isContextDone(ctx) {
			log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("HTML processing skipped due to shutdown")
			return
		}

		if atomic.LoadInt64(&crawler.stats.PagesProcessed) >= cfg.MaxPages {
			log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("Max pages limit reached, stopping HTML processing")
			// Note: If you want to stop the entire crawl immediately when MaxPages is reached,
			// you might want to consider cancelling the main context here as well,
			// which will signal Colly and other goroutines to stop.
			// For now, it just stops processing new HTML events.
			return
		}

		crawler.stats.IncrementPagesProcessed()
		// This should ideally decrement when a page is truly done (processed + kafka acked)
		// but for now, it's consistent with your original.
		url := e.Request.URL.String()
		html, err := e.DOM.Html()
		if err != nil {
			log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to extract HTML")
			crawler.stats.IncrementPagesFailed()
			crawler.markVisited(ctx, url) // Mark as visited even if HTML extraction failed, so we don't retry immediately
			return
		}

		// Call publishPageData. It no longer returns an error for asynchronous Kafka.
		// Success or failure for Kafka will be reported via handleKafkaSuccesses/Errors.
		crawler.publishPageData([]byte(html), url)

		// The logic for re-queueing based on Kafka publish error
		// needs to be moved to the handleKafkaErrors goroutine if you still want it.
		// When publishPageData returns, it just means the message was *enqueued*,
		// not necessarily sent to Kafka successfully.
		// For now, we'll assume enqueuing means it's on its way.

		crawler.stats.IncrementPagesSuccessful() // This counts successful *processing* by crawler, not necessarily successful Kafka delivery.
		// You might want to adjust this metric based on Kafka acks.
		log.WithFields(logrus.Fields{"url": url, "content_size": len(html)}).Debug("Page processed successfully and enqueued to Kafka.")
		crawler.markVisited(ctx, url)
	})
}

func (crawler *Crawler) setLinkHandler(collector *colly.Collector, ctx context.Context) {
	log := logger.GetLogger()
	cfg := config.GetConfig()
	urlValidator := validator.GetURLValidator()

	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		if isContextDone(ctx) {
			return
		}

		if atomic.LoadInt64(&crawler.stats.PagesProcessed) >= cfg.MaxPages {
			return
		}

		link := e.Attr("href")
		if link == "" || strings.HasPrefix(link, "javascript:") ||
			strings.HasPrefix(link, "mailto:") || strings.HasPrefix(link, "tel:") || strings.Contains(link, "#") {
			return
		}

		absoluteURL := e.Request.AbsoluteURL(link)
		if !urlValidator.IsValidURL(absoluteURL) {
			return
		}

		normalized, err := utils.NormalizeURL(absoluteURL)
		if err != nil {
			return
		}

		visited, err := crawler.isURLVisited(normalized)
		if err != nil || visited {
			if err != nil {
				log.WithFields(logrus.Fields{"url": normalized, "error": err}).Error("Error checking if URL is visited")
				crawler.stats.IncrementRedisFailed()
			}
			return
		}

		_, err = crawler.redisClient.SAdd(ctx, "crawler:pending_urls", normalized).Result()
		if err != nil {
			log.WithFields(logrus.Fields{"url": normalized, "error": err}).Error("Failed to add URL to Redis pending queue")
			crawler.stats.IncrementRedisFailed()
		}
	})
}

func (crawler *Crawler) setErrorHandler(collector *colly.Collector, ctx context.Context) {
	log := logger.GetLogger()
	cfg := config.GetConfig()

	collector.OnError(func(r *colly.Response, err error) {
		defer crawler.DecrementInFlightPages()
		defer crawler.stats.IncrementPagesFailed()
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
			}).Debug("Suppressed network error")
		}

		crawler.markVisited(ctx, r.Request.URL.String())
	})
}

func (crawler *Crawler) setResponseHandler(collector *colly.Collector, ctx context.Context) {
	log := logger.GetLogger()
	collector.OnResponse(func(r *colly.Response) {
		if !strings.Contains(r.Headers.Get("Content-Type"), "text/html") {
			log.WithFields(logrus.Fields{"url": r.Request.URL.String()}).Debug("Non-HTML content received, skipping")
			r.Request.Abort()
			crawler.markVisited(ctx, r.Request.URL.String())
			crawler.DecrementInFlightPages()
			return
		}
	})
}

func (crawler *Crawler) setResponseLogger(collector *colly.Collector) {
	cfg := config.GetConfig()
	log := logger.GetLogger()
	if cfg.EnableDebug {
		collector.OnResponse(func(r *colly.Response) {
			log.WithFields(logrus.Fields{
				"url":          r.Request.URL.String(),
				"method":       r.Request.Method, // Add request method
				"status_code":  r.StatusCode,
				"content_type": r.Headers.Get("Content-Type"),
				"size":         len(r.Body),
			}).Debug("Response received")
		})
	}
}

func (crawler *Crawler) markVisited(ctx context.Context, url string) {
	log := logger.GetLogger()
	_, err := crawler.redisClient.SAdd(ctx, "crawler:visited_urls", url).Result()
	if err != nil {
		log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to add URL to visited_urls Redis set")
		crawler.stats.IncrementRedisFailed()
	}
}
