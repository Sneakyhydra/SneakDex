package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	// Third-party
	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"

	"github.com/sneakyhydra/sneakdex/crawler/internal/config"
	"github.com/sneakyhydra/sneakdex/crawler/internal/logger"
	"github.com/sneakyhydra/sneakdex/crawler/internal/metrics"
)

var redisClient *redis.Client
var producer *sarama.SyncProducer
var stats *metrics.Metrics

// StartHealthCheck starts HTTP health check and metrics server.
func StartHealthCheck(wg *sync.WaitGroup, shutdown chan struct{}, newRedisClient *redis.Client, newProducer *sarama.SyncProducer, newStats *metrics.Metrics) {
	log := logger.GetLogger()
	cfg := config.GetConfig()

	redisClient = newRedisClient
	producer = newProducer
	stats = newStats

	mux := http.NewServeMux()

	mux.HandleFunc("/health", checkHealth)

	mux.HandleFunc("/metrics", checkMetrics)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HealthCheckPort),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Infof("Health check server starting on port %d", cfg.HealthCheckPort)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Errorf("Health check server error: %v", err)
		}
	}()

	// Graceful shutdown for health check server
	go func() {
		<-shutdown // Wait for the main shutdown signal
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Errorf("Health check server shutdown error: %v", err)
		} else {
			log.Info("Health check server shut down gracefully")
		}
	}()
}
