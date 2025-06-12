package validator

import "sync"

// ClearCache clears all cached results
func (urlValidator *URLValidator) ClearCache() {
	urlValidator.dnsCache = sync.Map{}
	urlValidator.domainCache = sync.Map{}
}
