package validator

import (
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// URLValidator provides methods to validate URLs against a whitelist/blacklist,
// perform DNS and domain filtering, and cache expensive operations for performance.
type URLValidator struct {
	whitelist []string
	blacklist []string
	log       *logrus.Logger

	// DNS caching
	dnsCache        sync.Map // map[string]DNSResult
	dnsCacheTimeout time.Duration

	// Domain validation caching
	domainCache sync.Map // map[string]bool

	// Validation configuration
	allowPrivateIPs bool
	allowLoopback   bool
	skipDNSCheck    bool
	maxURLLength    int
}

// Global singleton instance (optional, for convenience)
var (
	urlValidator *URLValidator
	initOnce     sync.Once
)

// NewURLValidator creates and sets a global URLValidator instance with sane defaults.
// Use GetURLValidator() to retrieve it.
func NewURLValidator(whitelist, blacklist []string, logger *logrus.Logger) {
	if logger == nil {
		logger = logrus.New()
		logger.Warn("No logger provided; using default logger")
	}

	initOnce.Do(func() {
		newUrlValidator := &URLValidator{
			whitelist:       whitelist,
			blacklist:       blacklist,
			log:             logger,
			dnsCacheTimeout: 5 * time.Minute, // Configurable
			allowPrivateIPs: false,
			allowLoopback:   false,
			skipDNSCheck:    false,
			maxURLLength:    2048, // Adjustable upper bound
		}

		urlValidator = newUrlValidator
	})
}

// GetURLValidator returns the global singleton instance.
func GetURLValidator() *URLValidator {
	if urlValidator == nil {
		panic("URLValidator not initialized: call NewURLValidator() first")
	}
	return urlValidator
}

// IsValidURL checks if a given URL string passes validation criteria:
// - Well-formed
// - HTTP(S) scheme
// - Passes domain allow/block logic
// - (Optionally) Passes IP validation from DNS resolution
func (uv *URLValidator) IsValidURL(rawURL string) bool {
	// --- Pre-filter: empty or too long ---
	if rawURL == "" || len(rawURL) > uv.maxURLLength {
		uv.log.WithField("url", rawURL).Debug("Rejected due to empty or overlong URL")
		return false
	}

	// --- Parse URL ---
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		uv.log.WithFields(logrus.Fields{
			"url":   rawURL,
			"error": err,
		}).Debug("Failed to parse URL")
		return false
	}

	// --- Scheme Check ---
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		uv.log.WithFields(logrus.Fields{
			"url":    rawURL,
			"scheme": parsedURL.Scheme,
		}).Debug("Invalid URL scheme")
		return false
	}

	// --- Hostname Check ---
	host := parsedURL.Hostname()
	if host == "" {
		uv.log.WithField("url", rawURL).Debug("Missing host in URL")
		return false
	}
	host = strings.ToLower(host)

	// --- Domain Filtering ---
	if !uv.isDomainAllowed(host) {
		uv.log.WithFields(logrus.Fields{
			"url":  rawURL,
			"host": host,
		}).Debug("Domain not allowed")
		return false
	}

	// --- Optional DNS/IP Filtering ---
	if !uv.skipDNSCheck && !uv.isIPValid(host) {
		uv.log.WithFields(logrus.Fields{
			"url":  rawURL,
			"host": host,
		}).Debug("DNS/IP validation failed")
		return false
	}

	return true
}
