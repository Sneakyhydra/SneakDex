// Package logger provides structured logging configuration for the Sneakdex crawler.
// It configures logrus with JSON formatting for structured log output suitable
// for production environments and log aggregation systems.
//
// Features:
//   - Configurable log levels (trace, debug, info, warn, error, fatal, panic)
//   - JSON structured output for machine parsing
//   - Standard output targeting for container environments
//   - Input validation and error handling
//
// The logger is designed to be initialized once at application startup and
// shared across all components for consistent logging behavior.
//
// Example usage:
//
//	logger, err := NewLogger("info")
//	if err != nil {
//	    log.Fatalf("Failed to initialize logger: %v", err)
//	}
//	logger.Info("Application started")
package logger

import (
	// Stdlib
	"fmt"
	"os"

	// Third-party
	"github.com/sirupsen/logrus"
)

// NewLogger sets up the global logrus logger with config-defined log level and JSON formatting.
func NewLogger(logLevel string) (*logrus.Logger, error) {
	log := logrus.New()

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid log level '%s': %w", logLevel, err)
	}

	log.SetLevel(level)
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)
	return log, nil
}
