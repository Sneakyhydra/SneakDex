package logger

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/sneakyhydra/sneakdex/crawler/internal/config"
)

var log *logrus.Logger

// Initialize logger creates a new logrus logger
func InitializeLogger() error {
	newlog := logrus.New()
	level, err := logrus.ParseLevel(config.GetConfig().LogLevel)
	if err != nil {
		return fmt.Errorf("{invalid log level: %w}", err)
	}
	newlog.SetLevel(level)
	newlog.SetFormatter(&logrus.JSONFormatter{})
	log = newlog
	return nil
}

func GetLogger() *logrus.Logger {
	return log
}
