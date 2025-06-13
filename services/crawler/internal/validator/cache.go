package validator

import "sync"

// ClearCache resets both DNS and domain validation caches to prevent stale lookups.
// This is safe for concurrent use.
func (v *URLValidator) ClearCache() {
	v.dnsCache = sync.Map{}    // Clears DNS resolution results
	v.domainCache = sync.Map{} // Clears domain status checks
}
