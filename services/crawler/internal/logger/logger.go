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
