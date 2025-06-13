package crawlerrors

import (
	"fmt"
	"time"
)

// CrawlError represents a structured error with crawl-specific metadata.
type CrawlError struct {
	URL       string    // The URL being crawled
	Operation string    // The operation being performed (e.g. "Fetch", "Parse", "Store")
	Err       error     // Underlying error
	Retry     bool      // Whether the operation should be retried
	Timestamp time.Time // When the error occurred
}

// Error implements the error interface for CrawlError.
func (e *CrawlError) Error() string {
	return fmt.Sprintf(
		"CrawlError: [%s] operation failed for URL: %s at %s | Retry: %v | Err: %v",
		e.Operation,
		e.URL,
		e.Timestamp.Format(time.RFC3339),
		e.Retry,
		e.Err,
	)
}

// Unwrap returns the underlying error to support errors.Unwrap and errors.Is.
func (e *CrawlError) Unwrap() error {
	return e.Err
}
