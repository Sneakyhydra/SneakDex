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
	c.setLinkHandler(collector)
	c.setErrorHandler(collector)
	c.setResponseHandler(collector)
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
			r.Headers.Set("Keep-Alive", "timeout=30, max=100")
			r.Headers.Set("Upgrade-Insecure-Requests", "1")

			c.IncrementInFlightPages()
			c.Log.WithFields(logrus.Fields{"url": r.URL.String()}).Debug("Visiting URL")
		}
	})
}

// setLinkHandler allows us to find links in an html page.
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
			// Fast rejection filters
			if link == "" || len(link) > 2000 {
				return
			}
			// Check first character for quick rejections
			switch link[0] {
			case '#', '?': // Fragment or query only
				return
			case 'j': // javascript:
				if strings.HasPrefix(link, "javascript:") {
					return
				}
			case 'm': // mailto:
				if strings.HasPrefix(link, "mailto:") {
					return
				}
			case 't': // tel:
				if strings.HasPrefix(link, "tel:") {
					return
				}
			}

			normalizedURL, valid := c.UrlValidator.IsValidURL(link)
			if !valid {
				return
			}

			visited, err := c.isURLSeen(normalizedURL)
			if err != nil || visited {
				return // Skip error logging for performance
			}

			parentDepthAny := e.Request.Ctx.GetAny("depth")
			parentDepth, ok := parentDepthAny.(int)
			if !ok {
				parentDepth = 1 // fallback if missing
			}

			c.AddToPending(QueueItem{
				URL:   normalizedURL,
				Depth: parentDepth + 1,
			})
		}
	})
}

// setErrorHandler allows us to handle errors gracefully.
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

// setResponseHandler allows us to process the response of a request and send it to kafka.
func (c *Crawler) setResponseHandler(collector *colly.Collector) {
	collector.OnResponse(func(r *colly.Response) {
		defer c.DecrementInFlightPages()
		// extract depth from context
		depthAny := r.Request.Ctx.GetAny("depth")
		depth, ok := depthAny.(int)
		if !ok {
			depth = 1
		}

		if c.Cfg.EnableDebug {
			c.Log.WithFields(logrus.Fields{
				"url":          r.Request.URL.String(),
				"method":       r.Request.Method,
				"status_code":  r.StatusCode,
				"content_type": r.Headers.Get("Content-Type"),
				"size":         len(r.Body),
				"depth":        depth,
			}).Debug("Response received")
		}

		if !strings.Contains(r.Headers.Get("Content-Type"), "text/html") {
			c.Log.WithFields(logrus.Fields{"url": r.Request.URL.String(), "depth": depth}).Debug("Non-HTML content received, skipping")
			c.MarkVisited(r.Request.URL.String())
			r.Request.Abort()
			return
		}

		c.Stats.IncrementPagesProcessed()
		url := r.Request.URL.String()
		html := string(r.Body)

		// Send the HTML content to Kafka
		if retry, err := c.sendToKafka(QueueItem{URL: url, Depth: depth}, html); err != nil {
			if retry {
				if exists, err := c.isURLRequeued(url); exists {
					c.Log.WithFields(logrus.Fields{"url": url}).Trace("URL already requeued once. Will be marked as visited")
					c.RemoveFromRequeued(url)
				} else {
					// Re-queue URL instead of marking as visited
					c.Log.WithFields(logrus.Fields{"url": url, "error": err}).Warn("Retriable error occurred, requeuing URL")

					c.AddToPending(QueueItem{URL: url, Depth: depth})
					c.AddToRequeued(url)
					return
				}
			}

			c.Log.WithFields(logrus.Fields{"url": url, "error": err}).Error("Failed to send to Kafka")
			c.Stats.IncrementPagesFailed()
			c.MarkVisited(url)
			return
		}

		c.Stats.IncrementPagesSuccessful()
		c.Log.WithFields(logrus.Fields{"url": url, "content_size": len(html)}).Debug("Page processed successfully and enqueued to Kafka.")
		c.MarkVisited(url)
	})
}
