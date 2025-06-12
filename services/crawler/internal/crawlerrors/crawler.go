package crawlerrors

import (
	"fmt"
	"time"
)

// CrawlError represents a structured error type for crawling operations
type CrawlError struct {
	URL       string
	Operation string
	Err       error
	Retry     bool
	Timestamp time.Time
}

// Error implements the error interface for CrawlError
func (err *CrawlError) Error() string {
	return fmt.Sprintf("CrawlError: %s operation failed for URL %s at %s: %v (Retry: %v)",
		err.Operation, err.URL, err.Timestamp.Format(time.RFC3339), err.Err, err.Retry)
}
