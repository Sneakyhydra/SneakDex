package validator

import (
	// Stdlib
	"time"
)

// SetDNSCacheTimeout sets how long DNS results should be cached.
func (uv *URLValidator) SetDNSCacheTimeout(timeout time.Duration) {
	uv.dnsCacheTimeout = timeout
}

// SetAllowPrivateIPs controls whether URLs resolving to private IPs are considered valid.
func (uv *URLValidator) SetAllowPrivateIPs(allow bool) {
	uv.allowPrivateIPs = allow
}

// SetAllowLoopback controls whether loopback addresses (e.g., 127.0.0.1) are allowed.
func (uv *URLValidator) SetAllowLoopback(allow bool) {
	uv.allowLoopback = allow
}

// SetSkipDNSCheck toggles whether to bypass DNS validation entirely.
func (uv *URLValidator) SetSkipDNSCheck(skip bool) {
	uv.skipDNSCheck = skip
}
