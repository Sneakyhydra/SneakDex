package validator

import (
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

type DNSResult struct {
	IPs       []net.IP
	Timestamp time.Time
	Valid     bool
}

// isIPValid checks if the host resolves to valid IPs with caching
func (urlValidator *URLValidator) isIPValid(host string) bool {
	// Check if it's already an IP address
	if ip := net.ParseIP(host); ip != nil {
		return urlValidator.isIPAllowed(ip)
	}

	// Check DNS cache first
	if cached, exists := urlValidator.dnsCache.Load(host); exists {
		if result, ok := cached.(DNSResult); ok {
			// Check if cache is still valid
			if time.Since(result.Timestamp) < urlValidator.dnsCacheTimeout {
				if !result.Valid {
					return false
				}
				// Check cached IPs
				return urlValidator.areIPsAllowed(result.IPs)
			}
		}
	}

	// Perform DNS lookup
	ips, err := net.LookupIP(host)
	dnsResult := DNSResult{
		IPs:       ips,
		Timestamp: time.Now(),
		Valid:     err == nil,
	}

	// Cache the result
	urlValidator.dnsCache.Store(host, dnsResult)

	if err != nil {
		urlValidator.log.WithFields(logrus.Fields{"host": host, "error": err}).Debug("DNS lookup failed")
		return false
	}

	return urlValidator.areIPsAllowed(ips)
}

// areIPsAllowed checks if any of the IPs are allowed
func (urlValidator *URLValidator) areIPsAllowed(ips []net.IP) bool {
	for _, ip := range ips {
		if urlValidator.isIPAllowed(ip) {
			return true
		}
	}
	return false
}

// isIPAllowed checks if a single IP is allowed based on configuration
func (urlValidator *URLValidator) isIPAllowed(ip net.IP) bool {
	if !urlValidator.allowLoopback && ip.IsLoopback() {
		urlValidator.log.WithFields(logrus.Fields{"ip": ip.String()}).Debug("Blocked loopback IP")
		return false
	}

	if !urlValidator.allowPrivateIPs && ip.IsPrivate() {
		urlValidator.log.WithFields(logrus.Fields{"ip": ip.String()}).Debug("Blocked private IP")
		return false
	}

	return true
}
