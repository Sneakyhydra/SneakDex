package validator

import (
	"strings"

	"github.com/sirupsen/logrus"
)

// isDomainAllowed checks whitelist and blacklist rules with caching
func (urlValidator *URLValidator) isDomainAllowed(host string) bool {
	// Check cache first
	if cached, exists := urlValidator.domainCache.Load(host); exists {
		if valid, ok := cached.(bool); ok {
			return valid
		}
	}

	allowed := urlValidator.checkDomainRules(host)

	// Cache the result
	urlValidator.domainCache.Store(host, allowed)

	return allowed
}

// checkDomainRules performs the actual domain filtering logic
func (urlValidator *URLValidator) checkDomainRules(host string) bool {
	// Check blacklist first (fail fast)
	for _, blocked := range urlValidator.blacklist {
		blocked = strings.ToLower(blocked)
		if urlValidator.matchesDomain(host, blocked) {
			urlValidator.log.WithFields(logrus.Fields{"host": host, "blocked_by": blocked}).Debug("Host blocked by blacklist")
			return false
		}
	}

	// Check whitelist if configured
	if len(urlValidator.whitelist) > 0 {
		for _, allowedDomain := range urlValidator.whitelist {
			allowedDomain = strings.ToLower(allowedDomain)
			if urlValidator.matchesDomain(host, allowedDomain) {
				return true
			}
		}
		// If whitelist is configured but no match found, deny
		urlValidator.log.WithFields(logrus.Fields{"host": host}).Debug("Host not in whitelist")
		return false
	}

	return true
}

// matchesDomain checks if host matches domain pattern
func (urlValidator *URLValidator) matchesDomain(host, domain string) bool {
	// Exact match
	if host == domain {
		return true
	}

	// Subdomain match
	if strings.HasSuffix(host, "."+domain) {
		return true
	}

	return false
}
