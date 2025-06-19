package validator

import (
	// Stdlib
	"sync"
)

// ClearCache resets both DNS and domain validation caches to prevent stale lookups.
func (uv *URLValidator) ClearCache() {
	uv.dnsCache = sync.Map{}       // Clears DNS resolution results
	uv.domainCache = sync.Map{}    // Clears domain status checks
	uv.normalizedURLs = sync.Map{} // Clears normalized URL cache
}
