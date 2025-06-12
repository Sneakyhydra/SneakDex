package validator

import (
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// URLValidator handles URL validation with caching and improved performance
type URLValidator struct {
	whitelist []string
	blacklist []string
	log       *logrus.Logger

	// Caching for DNS lookups
	dnsCache        sync.Map // map[string]DNSResult
	dnsCacheTimeout time.Duration

	// Caching for domain checks
	domainCache sync.Map // map[string]bool

	// Configuration
	allowPrivateIPs bool
	allowLoopback   bool
	skipDNSCheck    bool
	maxURLLength    int
}

var urlValidator *URLValidator

// NewURLValidator creates a new URL validator with default settings
func NewURLValidator(whitelist, blacklist []string, log *logrus.Logger) {
	newurlValidator := &URLValidator{
		whitelist:       whitelist,
		blacklist:       blacklist,
		log:             log,
		dnsCacheTimeout: 5 * time.Minute, // Cache DNS results for 5 minutes
		allowPrivateIPs: false,
		allowLoopback:   false,
		skipDNSCheck:    false,
		maxURLLength:    2048, // Reasonable URL length limit
	}
	urlValidator = newurlValidator
}

func GetURLValidator() *URLValidator {
	return urlValidator
}

// IsValidURL checks if a URL is valid and not blocked by blacklist or whitelist rules
func (urlValidator *URLValidator) IsValidURL(rawURL string) bool {
	// Quick checks first (cheapest operations)
	if rawURL == "" || len(rawURL) > urlValidator.maxURLLength {
		return false
	}

	// Parse the URL to ensure it is well-formed
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		urlValidator.log.WithFields(logrus.Fields{"url": rawURL, "error": err}).Debug("Invalid URL format")
		return false
	}

	// Check if the URL has a valid scheme and host
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}
	if parsedURL.Host == "" {
		return false
	}

	// Normalize the host to lowercase for consistent checks
	host := strings.ToLower(parsedURL.Hostname())

	// Check domain-based filtering first (before expensive DNS lookup)
	if !urlValidator.isDomainAllowed(host) {
		return false
	}

	// DNS validation (most expensive operation, do last)
	if !urlValidator.skipDNSCheck && !urlValidator.isIPValid(host) {
		return false
	}

	return true
}
