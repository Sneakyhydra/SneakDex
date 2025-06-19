package crawler

import (
	// StdLib
	"path"
	"strings"
	"sync/atomic"

	// Third-party
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/sirupsen/logrus"
)

// setupCollyCollector initializes a Colly collector with the necessary handlers and configurations.
func (c *Crawler) setupCollyCollector() *colly.Collector {
	collector := c.createBaseCollector()
	c.setRequestHandler(collector)
	c.setHTMLHandler(collector)
	c.setLinkHandler(collector)
	c.setErrorHandler(collector)
	c.setResponseHandler(collector)
	c.setResponseLogger(collector)
	return collector
}

// createBaseCollector initializes and returns a collector with the config options applied.
func (c *Crawler) createBaseCollector() *colly.Collector {
	options := []colly.CollectorOption{
		colly.MaxDepth(c.Cfg.CrawlDepth),
		colly.Async(true),
		colly.UserAgent(c.Cfg.UserAgent),
		colly.ParseHTTPErrorResponse(),
		colly.DetectCharset(),
	}

	if len(c.Blacklist) > 0 {
		options = append(options, colly.DisallowedDomains(c.Blacklist...))
	}
	if len(c.Whitelist) > 0 {
		options = append(options, colly.AllowedDomains(c.Whitelist...))
	}
	if c.Cfg.EnableDebug {
		options = append(options, colly.Debugger(&debug.LogDebugger{}))
	}

	collector := colly.NewCollector(options...)
	_ = collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: c.Cfg.MaxConcurrency,
		Delay:       c.Cfg.RequestDelay,
	})

	collector.SetRequestTimeout(c.Cfg.RequestTimeout)
	return collector
}

// setRequestHandler allows us to apply headers before the request is made.
func (c *Crawler) setRequestHandler(collector *colly.Collector) {
	var skipExts = map[string]struct{}{
		".pdf": {}, ".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".css": {}, ".js": {}, ".ico": {},
		".svg": {}, ".woff": {}, ".ttf": {}, ".mp4": {}, ".mp3": {}, ".zip": {}, ".exe": {},
	}

	collector.OnRequest(func(r *colly.Request) {
		select {
		case <-c.Ctx.Done():
			c.Log.WithFields(logrus.Fields{"url": r.URL.String()}).Debug("Request aborted due to shutdown")
			r.Abort()
			return
		default:
			if atomic.LoadInt64(&c.Stats.PagesProcessed) >= c.Cfg.MaxPages {
				c.Log.WithFields(logrus.Fields{"url": r.URL.String()}).Debug("Skipping due to MaxPages limit")
				select {
				case <-c.Ctx.Done():
					c.Log.Debug("Context done, stopping further requests")
				default:
					c.CtxCancel() // Cancel the context to stop further requests
					c.Log.Debug("Context cancelled due to MaxPages limit")
				}
				r.Abort()
				return
			}

			ext := strings.ToLower(path.Ext(r.URL.Path))
			if _, skip := skipExts[ext]; skip {
				c.Log.WithFields(logrus.Fields{"url": r.URL.String(), "ext": ext}).Debug("Skipping URL due to file extension")
				r.Abort()
				return
			}

			r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
			r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
			r.Headers.Set("Accept-Encoding", "gzip, deflate")
			r.Headers.Set("DNT", "1")
			r.Headers.Set("Connection", "keep-alive")
			r.Headers.Set("Upgrade-Insecure-Requests", "1")

			c.IncrementInFlightPages()
			c.Log.WithFields(logrus.Fields{"url": r.URL.String()}).Debug("Visiting URL")
		}

	})
}

// setHTMLHandler allows us to process a HTML response
func (c *Crawler) setHTMLHandler(collector *colly.Collector) {
	collector.OnHTML("html", func(e *colly.HTMLElement) {
		defer c.DecrementInFlightPages()

		select {
		case <-c.Ctx.Done():
			c.Log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("HTML processing skipped due to shutdown")
			return

		default:
			if atomic.LoadInt64(&c.Stats.PagesProcessed) >= c.Cfg.MaxPages {
				c.Log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("Max pages limit reached, stopping HTML processing")
				select {
				case <-c.Ctx.Done():
					c.Log.Debug("Context done, stopping further processing")
				default:
					c.CtxCancel() // Cancel the context to stop further processing
					c.Log.Debug("Context cancelled due to MaxPages limit")
				}
				return
			}

			c.Stats.IncrementPagesProcessed()
			url := e.Request.URL.String()
			html, err := e.DOM.Html()
			if err != nil {
				c.Log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to extract HTML")
				c.Stats.IncrementPagesFailed()
				c.MarkVisited(url) // Mark as visited even if HTML extraction failed, so we don't retry immediately
				return
			}

			// Send the HTML content to Kafka
			if retry, err := c.sendToKafka(url, html); err != nil {
				if retry {
					if exists, err := c.isURLRequeued(url); exists {
						c.Log.WithFields(logrus.Fields{"url": url}).Trace("URL already requeued once. Will be marked as visited")
						c.RemoveFromRequeuedLocal(url)
						c.RemoveFromRequeued(url)
					} else {
						// Re-queue URL instead of marking as visited
						c.Log.WithFields(logrus.Fields{"url": url, "error": err}).Warn("Retriable error occurred, requeuing URL")

						c.AddToPending(url)
						c.AddToRequeued(url)
						c.AddToRequeuedLocal(url)
						return
					}
				}

				c.Log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to send to Kafka")
				c.Stats.IncrementPagesFailed()
				// IMPORTANT: Mark as visited even if sending to Kafka fails
				c.MarkVisited(url) // This prevents immediate retries of the same URL
				return
			}

			c.Stats.IncrementPagesSuccessful()
			c.Log.WithFields(logrus.Fields{"url": url, "content_size": len(html)}).Debug("Page processed successfully and enqueued to Kafka.")
			c.MarkVisited(url)
		}

	})
}

func (c *Crawler) setLinkHandler(collector *colly.Collector) {
	collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		select {
		case <-c.Ctx.Done():
			c.Log.WithFields(logrus.Fields{"url": e.Request.URL.String()}).Debug("Link processing skipped due to shutdown")
			return
		default:
			if atomic.LoadInt64(&c.Stats.PagesProcessed) >= c.Cfg.MaxPages {
				return
			}

			link := e.Attr("href")
			if link == "" || strings.HasPrefix(link, "javascript:") ||
				strings.HasPrefix(link, "mailto:") || strings.HasPrefix(link, "tel:") || strings.Contains(link, "#") {
				return
			}

			absoluteURL := e.Request.AbsoluteURL(link)
			normalizedURL, valid := c.UrlValidator.IsValidURL(absoluteURL)
			if !valid {
				return
			}

			visited, err := c.isURLSeen(normalizedURL)
			if err != nil || visited {
				if err != nil {
					c.Log.WithFields(logrus.Fields{"url": normalizedURL, "error": err}).Error("Error checking if URL is visited")
				}
				return
			}

			c.AddToPending(normalizedURL)
		}
	})
}

func (c *Crawler) setErrorHandler(collector *colly.Collector) {
	collector.OnError(func(r *colly.Response, err error) {
		defer c.DecrementInFlightPages()
		defer c.Stats.IncrementPagesFailed()
		isNetworkError := strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "no such host")

		if !isNetworkError || c.Cfg.EnableDebug {
			c.Log.WithFields(logrus.Fields{
				"url":         r.Request.URL.String(),
				"status_code": r.StatusCode,
				"error":       err,
			}).Warn("Request failed")
		} else {
			c.Log.WithFields(logrus.Fields{
				"url":         r.Request.URL.String(),
				"status_code": r.StatusCode,
				"error":       err,
			}).Debug("Suppressed network error")
		}

		c.MarkVisited(r.Request.URL.String())
	})
}

func (c *Crawler) setResponseHandler(collector *colly.Collector) {
	collector.OnResponse(func(r *colly.Response) {
		if !strings.Contains(r.Headers.Get("Content-Type"), "text/html") {
			c.Log.WithFields(logrus.Fields{"url": r.Request.URL.String()}).Debug("Non-HTML content received, skipping")
			c.MarkVisited(r.Request.URL.String())
			c.DecrementInFlightPages()
			r.Request.Abort()
			return
		}
	})
}

func (c *Crawler) setResponseLogger(collector *colly.Collector) {
	if c.Cfg.EnableDebug {
		collector.OnResponse(func(r *colly.Response) {
			c.Log.WithFields(logrus.Fields{
				"url":          r.Request.URL.String(),
				"method":       r.Request.Method, // Add request method
				"status_code":  r.StatusCode,
				"content_type": r.Headers.Get("Content-Type"),
				"size":         len(r.Body),
			}).Debug("Response received")
		})
	}
}
