package validator

import "sync"

// UpdateWhitelist updates the whitelist and clears domain cache
func (urlValidator *URLValidator) UpdateWhitelist(whitelist []string) {
	urlValidator.whitelist = whitelist
	urlValidator.domainCache = sync.Map{} // Clear cache since rules changed
}

// UpdateBlacklist updates the blacklist and clears domain cache
func (urlValidator *URLValidator) UpdateBlacklist(blacklist []string) {
	urlValidator.blacklist = blacklist
	urlValidator.domainCache = sync.Map{} // Clear cache since rules changed
}
