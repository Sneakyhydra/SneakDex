package main

import (
	// StdLib
	"fmt"
	"os"
	"os/signal"
	"syscall"

	// Internal modules
	"github.com/sneakyhydra/sneakdex/crawler/internal/config"
	"github.com/sneakyhydra/sneakdex/crawler/internal/crawler"
	"github.com/sneakyhydra/sneakdex/crawler/internal/logger"
)

func main() {
	fmt.Println("👋 main.go reached")

	if err := config.InitializeConfig(); err != nil {
		fmt.Printf("Failed to initialize config: %v\n", err)
		return
	}

	if err := logger.InitializeLogger(); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return
	}

	log := logger.GetLogger()

	err := crawler.New()
	if err != nil {
		log.Fatalf("Failed to create crawler: %v", err)
	}

	newCrawler := crawler.GetCrawler()

	// Set up OS signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Channel to signal when the main crawler Start() routine completes
	done := make(chan struct{})

	// Start the main crawling process in a goroutine
	go func() {
		if err := newCrawler.Start(); err != nil {
			log.Errorf("Crawler encountered a critical error: %v", err)
		}
		close(done) // Signal that the crawling process has finished
	}()

	// Wait for either an OS signal or the crawler to complete naturally
	select {
	case <-sigChan:
		log.Info("Received OS shutdown signal. Initiating graceful shutdown...")
	case <-done:
		log.Info("Crawler completed all tasks naturally. Initiating graceful shutdown...")
	}

	// Always ensure shutdown is called
	newCrawler.Shutdown()
}
