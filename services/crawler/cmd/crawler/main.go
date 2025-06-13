package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sneakyhydra/sneakdex/crawler/internal/config"
	"github.com/sneakyhydra/sneakdex/crawler/internal/crawler"
	"github.com/sneakyhydra/sneakdex/crawler/internal/logger"
)

const (
	// AppName is the application identifier used in logs and error messages
	AppName = "Project Sneakdex Crawler"
	// ShutdownTimeout defines how long to wait for graceful shutdown before forcing exit
	ShutdownTimeout = 30 * time.Second
)

// exitCode represents different exit conditions for better debugging
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
// This separation makes the code more testable and provides cleaner error handling
func run() exitCode {
	fmt.Printf("%s starting...\n", AppName)

	// Initialize configuration
	if err := initializeConfig(); err != nil {
		fmt.Printf("Fatal: Failed to initialize configuration: %v\n", err)
		return exitConfigError
	}

	// Initialize logger early to enable structured logging for subsequent operations
	if err := initializeLogger(); err != nil {
		fmt.Printf("Fatal: Failed to initialize logger: %v\n", err)
		return exitLoggerError
	}

	log := logger.GetLogger()
	log.Info("Configuration and logging initialized successfully")

	// Create crawler instance with proper error context
	crawlerInstance, err := createCrawler()
	if err != nil {
		log.WithError(err).Fatal("Failed to create crawler instance")
		return exitCrawlerCreationError
	}

	log.Info("Crawler instance created successfully")

	// Start crawler and handle shutdown signals
	if code := runCrawlerWithShutdown(crawlerInstance, log); code != exitSuccess {
		return code
	}

	log.Info("Application shutdown completed successfully")
	return exitSuccess
}

// initializeConfig wraps config initialization with better error context
func initializeConfig() error {
	if err := config.InitializeConfig(); err != nil {
		return fmt.Errorf("configuration initialization failed: %w", err)
	}
	return nil
}

// initializeLogger wraps logger initialization with better error context
func initializeLogger() error {
	if err := logger.InitializeLogger(); err != nil {
		return fmt.Errorf("logger initialization failed: %w", err)
	}
	return nil
}

// createCrawler creates and validates the crawler instance
func createCrawler() (*crawler.Crawler, error) {
	if err := crawler.New(); err != nil {
		return nil, fmt.Errorf("crawler instantiation failed: %w", err)
	}

	crawlerInstance := crawler.GetCrawler()
	if crawlerInstance == nil {
		return nil, fmt.Errorf("crawler instance is nil after creation")
	}

	return crawlerInstance, nil
}

// runCrawlerWithShutdown handles the main crawler execution and graceful shutdown
func runCrawlerWithShutdown(crawlerInstance *crawler.Crawler, log *logrus.Logger) exitCode {
	// Set up OS signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(sigChan)

	// Channel to capture crawler completion and any runtime errors
	crawlerDone := make(chan error, 1)

	// Start the crawler in a separate goroutine
	go func() {
		log.Info("Starting crawler main process")
		crawlerDone <- crawlerInstance.Start()
	}()

	// Wait for either completion or shutdown signal
	var shutdownReason string
	var crawlerErr error

	select {
	case sig := <-sigChan:
		shutdownReason = fmt.Sprintf("received OS signal: %v", sig)
		log.WithField("signal", sig.String()).Warn("Shutdown signal received")

	case crawlerErr = <-crawlerDone:
		if crawlerErr != nil {
			shutdownReason = "crawler encountered fatal error"
			log.WithError(crawlerErr).Error("Crawler terminated with error")
		} else {
			shutdownReason = "crawler completed successfully"
			log.Info("Crawler completed all tasks successfully")
		}
	}

	// Perform graceful shutdown
	log.WithField("reason", shutdownReason).Info("Initiating graceful shutdown")

	if err := performGracefulShutdown(crawlerInstance, log); err != nil {
		log.WithError(err).Error("Graceful shutdown encountered errors")
		return exitShutdownError
	}

	// Return appropriate exit code based on crawler result
	if crawlerErr != nil {
		return exitCrawlerRuntimeError
	}

	return exitSuccess
}

// performGracefulShutdown handles the shutdown process with timeout
func performGracefulShutdown(crawlerInstance *crawler.Crawler, log *logrus.Logger) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	shutdownDone := make(chan struct{}, 1)

	// Perform shutdown in goroutine to respect timeout
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.WithField("panic", r).Error("Panic occurred during shutdown")
			}
		}()

		crawlerInstance.Shutdown()
		shutdownDone <- struct{}{}
	}()

	select {
	case <-shutdownDone:
		log.Info("Graceful shutdown completed successfully")
		return nil

	case <-shutdownCtx.Done():
		log.Error("Shutdown timeout exceeded, forcing exit")
		return fmt.Errorf("shutdown timeout exceeded (%v)", ShutdownTimeout)
	}
}
