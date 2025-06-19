package validator

import (
	// Stdlib
	"net"
	"time"

	// Third-party
	"github.com/sirupsen/logrus"
)

// DNSResult represents the result of a DNS resolution
type DNSResult struct {
	IPs       []net.IP  // Resolved IPs
	Timestamp time.Time // When the DNS resolution occurred
	Valid     bool      // Whether the DNS resolution succeeded
}

// isIPValid resolves a host to IPs (with caching) and checks if they are allowed.
// It also supports direct IP input (bypassing DNS lookup).
func (uv *URLValidator) isIPValid(host string) bool {
	// Case 1: Host is a raw IP address
	if ip := net.ParseIP(host); ip != nil {
		return uv.isIPAllowed(ip)
	}

	// Case 2: Check DNS cache for host
	if cached, exists := uv.dnsCache.Load(host); exists {
		if result, ok := cached.(DNSResult); ok {
			if time.Since(result.Timestamp) < uv.dnsCacheTimeout {
				if !result.Valid {
					uv.log.WithFields(logrus.Fields{
						"host": host,
					}).Debug("Using cached failed DNS result")
					return false
				}
				// Validate cached IPs
				return uv.areIPsAllowed(result.IPs)
			}
			// Expired cache entry will be overwritten below
		} else {
			// Defensive programming: remove bad cache data
			uv.log.WithFields(logrus.Fields{
				"host": host,
			}).Warn("Invalid DNS cache entry; ignoring and refreshing")
			uv.dnsCache.Delete(host)
		}
	}

	// Case 3: DNS lookup required
	ips, err := net.LookupIP(host)

	// Cache the result regardless of success
	dnsResult := DNSResult{
		IPs:       ips,
		Timestamp: time.Now(),
		Valid:     err == nil,
	}
	uv.dnsCache.Store(host, dnsResult)

	if err != nil {
		uv.log.WithFields(logrus.Fields{
			"host":  host,
			"error": err,
		}).Debug("DNS resolution failed")
		return false
	}

	return uv.areIPsAllowed(ips)
}

// areIPsAllowed returns true if any of the resolved IPs are allowed.
func (uv *URLValidator) areIPsAllowed(ips []net.IP) bool {
	for _, ip := range ips {
		if uv.isIPAllowed(ip) {
			return true
		}
	}
	return false
}

// isIPAllowed checks a single IP against configured rules for loopback and private ranges.
func (uv *URLValidator) isIPAllowed(ip net.IP) bool {
	if ip == nil {
		return false
	}

	if !uv.allowLoopback && ip.IsLoopback() {
		uv.log.WithFields(logrus.Fields{
			"ip": ip.String(),
		}).Debug("Blocked loopback IP")
		return false
	}

	if !uv.allowPrivateIPs && ip.IsPrivate() {
		uv.log.WithFields(logrus.Fields{
			"ip": ip.String(),
		}).Debug("Blocked private IP")
		return false
	}

	return true
}
