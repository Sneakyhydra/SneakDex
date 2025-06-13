package utils

import (
	"fmt"
	"net/url"
	"strings"
)

// URLError represents URL-specific errors with additional context
type URLError struct {
	URL    string
	Op     string
	Reason string
	Err    error
}

func (e *URLError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("URL %s failed during %s: %s (%v)", e.URL, e.Op, e.Reason, e.Err)
	}
	return fmt.Sprintf("URL %s failed during %s: %s", e.URL, e.Op, e.Reason)
}

func (e *URLError) Unwrap() error {
	return e.Err
}

// NormalizeURL removes fragments and query parameters for deduplication purposes,
// converts scheme/host to lowercase, and applies consistent path normalization.
// This function is essential for web crawlers to avoid crawling duplicate pages
// that differ only in fragments, query parameters, or case variations.
func NormalizeURL(rawURL string) (string, error) {
	// Validate input
	if strings.TrimSpace(rawURL) == "" {
		return "", &URLError{
			URL:    rawURL,
			Op:     "normalize",
			Reason: "URL cannot be empty or whitespace",
		}
	}

	// Parse the URL to ensure it is well-formed
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", &URLError{
			URL:    rawURL,
			Op:     "normalize",
			Reason: "malformed URL structure",
			Err:    err,
		}
	}

	// Ensure we have a valid scheme for web crawling
	if parsed.Scheme == "" {
		return "", &URLError{
			URL:    rawURL,
			Op:     "normalize",
			Reason: "missing URL scheme (http/https required)",
		}
	}

	// Only allow HTTP/HTTPS schemes for web crawling
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", &URLError{
			URL:    rawURL,
			Op:     "normalize",
			Reason: fmt.Sprintf("unsupported scheme '%s' (only http/https allowed)", scheme),
		}
	}

	// Ensure we have a host
	if parsed.Host == "" {
		return "", &URLError{
			URL:    rawURL,
			Op:     "normalize",
			Reason: "missing hostname",
		}
	}

	// Apply normalizations for deduplication
	parsed.Scheme = scheme                     // Already lowercase from validation
	parsed.Host = strings.ToLower(parsed.Host) // Normalize host to lowercase
	parsed.Fragment = ""                       // Remove fragments (client-side navigation)
	parsed.RawQuery = ""                       // Remove query parameters (session IDs, tracking, etc.)

	// Normalize path for consistent URLs
	parsed.Path = normalizePath(parsed.Path)

	// Return the cleaned, normalized URL
	normalizedURL := parsed.String()

	// Final validation to ensure the normalized URL is still valid
	if _, err := url.Parse(normalizedURL); err != nil {
		return "", &URLError{
			URL:    rawURL,
			Op:     "normalize",
			Reason: "normalization produced invalid URL",
			Err:    err,
		}
	}

	return normalizedURL, nil
}

// normalizePath applies consistent path normalization rules
func normalizePath(path string) string {
	// Handle empty path
	if path == "" {
		return "/"
	}

	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Remove trailing slash except for root path
	if path != "/" && strings.HasSuffix(path, "/") {
		path = strings.TrimRight(path, "/")
	}

	// Handle multiple consecutive slashes
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	return path
}
