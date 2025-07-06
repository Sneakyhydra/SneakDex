package validator

import (
	// StdLib
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	// Third-party
	"github.com/sirupsen/logrus"
)

// URLValidator provides methods to validate URLs against a whitelist/blacklist,
// perform DNS and domain filtering, and cache expensive operations for performance.
type URLValidator struct {
	whitelist []string
	blacklist []string
	log       *logrus.Logger

	// Normalized URLs cache
	normalizedURLs sync.Map

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

// NewURLValidator initializes and returns a URLValidator with the given options
func NewURLValidator(whitelist, blacklist []string, log *logrus.Logger) *URLValidator {
	if log == nil {
		log = logrus.New()
		log.Warn("No logger provided; using default logger")
	}

	newUrlValidator := &URLValidator{
		whitelist:       whitelist,
		blacklist:       blacklist,
		log:             log,
		dnsCacheTimeout: 5 * time.Minute,
		allowPrivateIPs: false,
		allowLoopback:   false,
		skipDNSCheck:    true,
		maxURLLength:    2048, // Default max URL length
	}

	return newUrlValidator
}

// IsValidURL checks if a given URL string passes validation criteria:
// - Well-formed
// - HTTP(S) scheme
// - Passes domain allow/block logic
// - Passes IP validation from DNS resolution (if enabled)
func (uv *URLValidator) IsValidURL(rawURL string) (string, bool) {
	// Trim whitespace from the input URL
	trimmedURL := strings.TrimSpace(rawURL)

	// --- URL Length Check ---
	if trimmedURL == "" || len(trimmedURL) > uv.maxURLLength {
		uv.log.WithFields(logrus.Fields{
			"url":    trimmedURL,
			"length": len(trimmedURL),
		}).Debug("Invalid URL length")
		return trimmedURL, false
	}

	// --- Cache Check ---
	if cachedURL, ok := uv.normalizedURLs.Load(trimmedURL); ok {
		// If the URL is already normalized, return it directly
		return cachedURL.(string), true
	}

	// --- Parse URL ---
	parsedURL, err := url.Parse(trimmedURL)
	if err != nil {
		uv.log.WithFields(logrus.Fields{
			"url":   trimmedURL,
			"error": err,
		}).Debug("Failed to parse URL")
		return trimmedURL, false
	}

	// --- Scheme Check ---
	if parsedURL.Scheme == "" {
		uv.log.WithField("url", trimmedURL).Debug("Missing URL scheme")
		return trimmedURL, false
	}

	// Only allow HTTP/HTTPS schemes for web crawling
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		uv.log.WithFields(logrus.Fields{
			"url":    trimmedURL,
			"scheme": scheme,
		}).Debug("Invalid URL scheme")
		return trimmedURL, false
	}

	// --- Hostname Check ---
	// Lowercase the host for consistency
	host := strings.ToLower(parsedURL.Host)

	// Remove default ports from the host for normalization
	if (parsedURL.Scheme == "http" && strings.HasSuffix(host, ":80")) ||
		(parsedURL.Scheme == "https" && strings.HasSuffix(host, ":443")) {
		host = strings.Split(host, ":")[0]
	}

	// Validate that the cleaned host is present and well-formed
	if host == "" || parsedURL.Hostname() == "" {
		uv.log.WithField("url", trimmedURL).Debug("Missing or invalid hostname")
		return trimmedURL, false
	}

	// --- Domain Filtering ---
	if !uv.isDomainAllowed(host) {
		uv.log.WithFields(logrus.Fields{
			"url":  trimmedURL,
			"host": host,
		}).Debug("Domain not allowed")
		return trimmedURL, false
	}

	// --- Optional DNS/IP Filtering ---
	if !uv.skipDNSCheck && !uv.isIPValid(host) {
		uv.log.WithFields(logrus.Fields{
			"url":  trimmedURL,
			"host": host,
		}).Debug("DNS/IP validation failed")
		return trimmedURL, false
	}

	// --- URL Normalization ---
	// Apply normalizations for deduplication
	parsedURL.Scheme = scheme // Already lowercase from validation
	parsedURL.Host = host     // Already lowercase and cleaned
	parsedURL.Fragment = ""   // Remove fragments (client-side navigation)
	parsedURL.RawQuery = ""   // Remove query parameters (session IDs, tracking, etc.)

	// Normalize path for consistent URLs
	parsedURL.Path = uv.normalizePath(parsedURL.Path)

	// Return the cleaned, normalized URL
	normalizedURL := parsedURL.String()

	// Final validation to ensure the normalized URL is still valid
	if _, err := url.Parse(normalizedURL); err != nil {
		uv.log.WithFields(logrus.Fields{
			"url":   trimmedURL,
			"error": err,
		}).Debug("Normalization produced invalid URL")
		return trimmedURL, false
	}

	// Cache the normalized URL to avoid redundant processing in the future
	uv.normalizedURLs.Store(trimmedURL, normalizedURL)

	return normalizedURL, true
}

// normalizePath applies consistent and safe URL path normalization
func (uv *URLValidator) normalizePath(p string) string {
	clean := path.Clean("/" + strings.TrimSpace(p))
	if clean == "." {
		return "/"
	}
	return clean
}
