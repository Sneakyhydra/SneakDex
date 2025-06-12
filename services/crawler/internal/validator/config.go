package validator

import "time"

// SetDNSCacheTimeout sets the DNS cache timeout duration
func (urlValidator *URLValidator) SetDNSCacheTimeout(timeout time.Duration) {
	urlValidator.dnsCacheTimeout = timeout
}

// SetAllowPrivateIPs configures whether private IPs are allowed
func (urlValidator *URLValidator) SetAllowPrivateIPs(allow bool) {
	urlValidator.allowPrivateIPs = allow
}

// SetAllowLoopback configures whether loopback are allowed
func (urlValidator *URLValidator) SetAllowLoopback(allow bool) {
	urlValidator.allowPrivateIPs = allow
}

// SetSkipDNSCheck configures whether to skip DNS validation
func (urlValidator *URLValidator) SetSkipDNSCheck(skip bool) {
	urlValidator.skipDNSCheck = skip
}
