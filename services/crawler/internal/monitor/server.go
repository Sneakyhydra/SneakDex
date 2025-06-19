package monitor

import (
	// Stdlib
	"context"
	"fmt"
	"net/http"
	"time"

	// Internal modules
	"github.com/sneakyhydra/sneakdex/crawler/internal/crawler"
)

type monitorServer struct {
	port       int          // Port for the monitor server
	httpServer *http.Server // HTTP server instance
	crawler    *crawler.Crawler
}

// Initialize the monitor server configuration
func InitializeMonitorServer(crawler *crawler.Crawler) *monitorServer {
	ms := &monitorServer{
		port:       crawler.Cfg.MonitorPort,
		httpServer: nil, // Will be set in Start function
		crawler:    crawler,
	}

	return ms
}

// Start launches the monitor server and gracefully shuts down on signal.
func (ms *monitorServer) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", ms.handleHealth)
	mux.HandleFunc("/metrics", ms.handleMetrics)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", ms.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	ms.httpServer = server

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ms.crawler.Stats.SyncPrometheusMetrics() // Sync metrics periodically
			case <-ms.crawler.CShutdown:
				ms.crawler.Stats.SyncPrometheusMetrics() // Final sync on shutdown
				return                                   // Exit the goroutine on shutdown signal
			}
		}
	}()

	ms.crawler.Wg.Add(1)
	go func() {
		defer ms.crawler.Wg.Done()
		ms.crawler.Log.Infof("Monitor server starting on port %d", ms.port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			ms.crawler.Log.Errorf("Monitor server error: %v", err)
		}
	}()

	// Graceful shutdown
	go func() {
		<-ms.crawler.CShutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			ms.crawler.Log.Errorf("Monitor server shutdown error: %v", err)
		} else {
			ms.crawler.Log.Info("Monitor server shut down gracefully")
		}
	}()
}
