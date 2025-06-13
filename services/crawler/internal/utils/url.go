package utils

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizeURL removes fragments and query parameters for deduplication purposes,
// and converts scheme/host to lowercase.
func NormalizeURL(rawURL string) (string, error) {
	// Parse the URL to ensure it is well-formed
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL %q: %w", rawURL, err)
	}

	// Remove fragments and query parameters
	parsed.Fragment = "" // remove fragments
	parsed.RawQuery = "" // remove query parameters

	// Normalize scheme and host to lowercase
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)

	// Remove trailing slash except root path
	if parsed.Path != "/" {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}

	return parsed.String(), nil
}
