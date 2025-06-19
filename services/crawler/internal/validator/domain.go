package validator

import (
	// Stdlib
	"strings"

	// Third-party
	"github.com/sirupsen/logrus"
)

// isDomainAllowed determines whether a given host is allowed based on blacklist/whitelist rules.
// It uses a cache for performance and avoids repeated rule evaluations.
func (uv *URLValidator) isDomainAllowed(host string) bool {
	// Check cache
	if cached, exists := uv.domainCache.Load(host); exists {
		if valid, ok := cached.(bool); ok {
			return valid
		}
		// Fallback: invalid cache type, remove it
		uv.log.WithFields(logrus.Fields{
			"host":  host,
			"error": "Invalid cache type",
		}).Warn("Corrupted cache entry detected; ignoring")
		uv.domainCache.Delete(host)
	}

	allowed := uv.checkDomainRules(host)

	// Cache the decision
	uv.domainCache.Store(host, allowed)
	return allowed
}

// checkDomainRules applies blacklist and whitelist filtering to the host.
// Blacklist blocks override whitelist, and whitelist denies non-listed domains if defined.
func (uv *URLValidator) checkDomainRules(host string) bool {
	// --- Blacklist Evaluation (Fail-Fast) ---
	for _, blocked := range uv.blacklist {
		if strings.Contains(host, blocked) {
			uv.log.WithFields(logrus.Fields{
				"host":    host,
				"blocked": blocked,
			}).Info("Domain blocked by blacklist rule")
			return false // Blocked by blacklist
		}
	}

	// --- Whitelist Evaluation ---
	if len(uv.whitelist) > 0 {
		for _, allowed := range uv.whitelist {
			if strings.Contains(host, allowed) {
				uv.log.WithFields(logrus.Fields{
					"host":    host,
					"allowed": allowed,
				}).Info("Domain allowed by whitelist rule")
				return true // Allowed by whitelist
			}
		}

		// If we reach here, the host is not in the whitelist
		uv.log.WithFields(logrus.Fields{
			"host": host,
		}).Info("Domain not allowed; not in whitelist")
		return false // Not allowed by whitelist
	}

	// If no whitelist exists, allow by default
	return true
}
