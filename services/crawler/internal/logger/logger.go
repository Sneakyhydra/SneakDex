package logger

import (
	"fmt"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/sneakyhydra/sneakdex/crawler/internal/config"
)

var (
	log      *logrus.Logger
	initOnce sync.Once
)

// InitializeLogger sets up the global logrus logger with config-defined log level and JSON formatting.
// It ensures logger is initialized only once, thread-safely.
func InitializeLogger() error {
	var initErr error
	initOnce.Do(func() {
		newlog := logrus.New()

		level, err := logrus.ParseLevel(config.GetConfig().LogLevel)
		if err != nil {
			initErr = fmt.Errorf("invalid log level: %w", err)
			return
		}

		newlog.SetLevel(level)
		newlog.SetFormatter(&logrus.JSONFormatter{})
		newlog.SetOutput(os.Stdout)

		log = newlog
	})
	return initErr
}

// GetLogger returns the initialized logger. Panics if not initialized.
func GetLogger() *logrus.Logger {
	if log == nil {
		panic("logger not initialized: call InitializeLogger() first")
	}
	return log
}
