package validator

import (
	"strings"

	"github.com/sirupsen/logrus"
)

// isDomainAllowed determines whether a given host is allowed based on blacklist/whitelist rules.
// It uses a cache for performance and avoids repeated rule evaluations.
func (urlValidator *URLValidator) isDomainAllowed(host string) bool {
	normalizedHost := normalizeDomain(host)

	// Check cache
	if cached, exists := urlValidator.domainCache.Load(normalizedHost); exists {
		if valid, ok := cached.(bool); ok {
			return valid
		}
		// Fallback: invalid cache type, remove it
		urlValidator.log.WithFields(logrus.Fields{
			"host":  normalizedHost,
			"error": "Invalid cache type",
		}).Warn("Corrupted cache entry detected; ignoring")
		urlValidator.domainCache.Delete(normalizedHost)
	}

	allowed := urlValidator.checkDomainRules(normalizedHost)

	// Cache the decision
	urlValidator.domainCache.Store(normalizedHost, allowed)
	return allowed
}

// checkDomainRules applies blacklist and whitelist filtering to the host.
// Blacklist blocks override whitelist, and whitelist denies non-listed domains if defined.
func (urlValidator *URLValidator) checkDomainRules(host string) bool {
	// --- Blacklist Evaluation (Fail-Fast) ---
	for _, blocked := range urlValidator.blacklist {
		blocked = normalizeDomain(blocked)
		if urlValidator.matchesDomain(host, blocked) {
			urlValidator.log.WithFields(logrus.Fields{
				"host":        host,
				"blacklisted": blocked,
			}).Debug("Host denied by blacklist")
			return false
		}
	}

	// --- Whitelist Evaluation ---
	if len(urlValidator.whitelist) > 0 {
		for _, allowed := range urlValidator.whitelist {
			allowed = normalizeDomain(allowed)
			if urlValidator.matchesDomain(host, allowed) {
				return true
			}
		}
		// Host not in whitelist
		urlValidator.log.WithFields(logrus.Fields{
			"host": host,
		}).Debug("Host denied; not in whitelist")
		return false
	}

	// If no whitelist exists, allow by default
	return true
}

// matchesDomain checks if the host matches a domain pattern.
// Supports:
//   - exact domain match
//   - subdomain match (e.g., a.example.com -> example.com)
//   - wildcard domain match (e.g., *.example.com)
func (urlValidator *URLValidator) matchesDomain(host, domain string) bool {
	// Handle wildcard match (e.g., *.example.com)
	if strings.HasPrefix(domain, "*.") {
		suffix := domain[1:] // Remove '*'
		return strings.HasSuffix(host, suffix)
	}

	// Exact match
	if host == domain {
		return true
	}

	// Subdomain match
	return strings.HasSuffix(host, "."+domain)
}

// normalizeDomain lowercases and strips trailing dots from a domain.
// This ensures consistent comparisons.
func normalizeDomain(domain string) string {
	return strings.TrimSuffix(strings.ToLower(domain), ".")
}
