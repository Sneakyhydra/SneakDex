package validator

import "time"

// SetDNSCacheTimeout sets how long DNS results should be cached.
func (v *URLValidator) SetDNSCacheTimeout(timeout time.Duration) {
	v.dnsCacheTimeout = timeout
}

// SetAllowPrivateIPs controls whether URLs resolving to private IPs are considered valid.
func (v *URLValidator) SetAllowPrivateIPs(allow bool) {
	v.allowPrivateIPs = allow
}

// SetAllowLoopback controls whether loopback addresses (e.g., 127.0.0.1) are allowed.
func (v *URLValidator) SetAllowLoopback(allow bool) {
	v.allowLoopback = allow
}

// SetSkipDNSCheck toggles whether to bypass DNS validation entirely.
func (v *URLValidator) SetSkipDNSCheck(skip bool) {
	v.skipDNSCheck = skip
}
