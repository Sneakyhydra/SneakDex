// Package main implements the entry point for the SneakDex web crawler service.
// This application provides a production-ready web crawler that discovers and processes
// web content for the SneakDex system, with comprehensive error handling, graceful
// shutdown, and operational monitoring.
//
// The crawler service integrates multiple components:
//   - Configuration management with environment variable support
//   - Structured JSON logging with configurable levels
//   - Distributed crawling using Redis for URL queue management
//   - Kafka integration for downstream content processing
//   - HTTP monitoring endpoints for health checks and metrics
//   - Graceful shutdown handling with configurable timeouts
//
// The application follows production best practices including:
//   - Proper exit codes for different failure scenarios
//   - Signal-based shutdown handling (SIGTERM, SIGINT, SIGQUIT)
//   - Comprehensive error logging and monitoring
//   - Resource cleanup and graceful component shutdown
//
// Usage:
//
//	Set environment variables for configuration (see config package)
//	Run: ./crawler
//	Monitor: curl http://localhost:8080/health
package main

import (
	// StdLib
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Third-party
	"github.com/sirupsen/logrus"

	// Internal modules
	"github.com/sneakyhydra/sneakdex/crawler/internal/config"
	"github.com/sneakyhydra/sneakdex/crawler/internal/crawler"
	"github.com/sneakyhydra/sneakdex/crawler/internal/logger"
	"github.com/sneakyhydra/sneakdex/crawler/internal/monitor"
)

const (
	// ShutdownTimeout defines how long to wait for graceful shutdown before forcing exit
	ShutdownTimeout = 30 * time.Second
)

// exitCode represents different exit conditions
type exitCode int

const (
	exitSuccess exitCode = iota
	exitConfigError
	exitLoggerError
	exitCrawlerCreationError
	exitCrawlerRuntimeError
	exitShutdownError
)

func main() {
	os.Exit(int(run()))
}

// run contains the main application logic and returns an exit code
func run() exitCode {
	logrus.Info("SneakDex Crawler Starting...")

	var cfg *config.Config
	var log *logrus.Logger
	var err error

	// Initialize configuration
	if cfg, err = config.InitializeConfig(); err != nil {
		logrus.WithError(err).Fatal("Failed to initialize configuration")
		return exitConfigError
	}

	// Initialize logger early to enable structured logging for subsequent operations
	if log, err = logger.NewLogger(cfg.LogLevel); err != nil {
		logrus.WithError(err).Fatal("Failed to initialize logger")
		return exitLoggerError
	}

	if log == nil {
		logrus.Fatal("Logger is not initialized")
		return exitLoggerError
	}
	logrus.Info("Configuration and logging initialized successfully")

	// Create crawler instance with proper error context
	crawlerInstance, err := crawler.New(cfg, log)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create crawler instance")
		return exitCrawlerCreationError
	}

	logrus.Info("Crawler instance created successfully")

	// Start the monitor server in a new goroutine.
	// This server provides insights into the crawler's operational status.
	monitorServer := monitor.InitializeMonitorServer(crawlerInstance)
	monitorServer.Start()

	// Start crawler and handle shutdown signals
	if code := runCrawlerWithShutdown(crawlerInstance); code != exitSuccess {
		return code
	}

	logrus.Info("Application shutdown completed successfully")
	return exitSuccess
}

// runCrawlerWithShutdown handles the main crawler execution and graceful shutdown
func runCrawlerWithShutdown(crawlerInstance *crawler.Crawler) exitCode {
	// Set up OS signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(sigChan)

	// Channel to capture crawler completion and any runtime errors
	crawlerDone := make(chan error, 1)

	// Start the crawler in a separate goroutine
	go func() {
		logrus.Info("Starting crawler main process")
		crawlerDone <- crawlerInstance.Start()
	}()

	// Wait for either completion or shutdown signal
	var shutdownReason string
	var crawlerErr error

	select {
	case sig := <-sigChan:
		shutdownReason = fmt.Sprintf("received OS signal: %v", sig)
		logrus.WithField("signal", sig.String()).Warn("Shutdown signal received")

		// Use select with timeout to avoid indefinite block
		select {
		case <-crawlerDone:
		case <-time.After(ShutdownTimeout / 3):
			logrus.Warn("Timeout draining crawlerDone; proceeding to shutdown")
		}

	case crawlerErr = <-crawlerDone:
		if crawlerErr != nil {
			shutdownReason = "crawler encountered fatal error"
			logrus.WithError(crawlerErr).Error("Crawler terminated with error")
		} else {
			shutdownReason = "crawler completed successfully"
			logrus.Info("Crawler completed all tasks successfully")
		}
	}

	// Perform graceful shutdown
	logrus.WithField("reason", shutdownReason).Info("Initiating graceful shutdown")

	if err := performGracefulShutdown(crawlerInstance); err != nil {
		logrus.WithError(err).Error("Graceful shutdown encountered errors")
		return exitShutdownError
	}

	// Return appropriate exit code based on crawler result
	if crawlerErr != nil {
		return exitCrawlerRuntimeError
	}

	return exitSuccess
}

// performGracefulShutdown handles the shutdown process with timeout
func performGracefulShutdown(crawlerInstance *crawler.Crawler) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	shutdownDone := make(chan struct{}, 1)

	// Perform shutdown in goroutine to respect timeout
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.WithField("panic", r).Error("Panic occurred during shutdown")
			}
		}()

		crawlerInstance.Shutdown()
		shutdownDone <- struct{}{}
	}()

	select {
	case <-shutdownDone:
		logrus.Info("Graceful shutdown completed successfully")
		return nil

	case <-shutdownCtx.Done():
		logrus.Error("Shutdown timeout exceeded, forcing exit")
		return fmt.Errorf("shutdown timeout exceeded (%v)", ShutdownTimeout)
	}
}
