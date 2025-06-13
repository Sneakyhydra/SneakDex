package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"

	"github.com/sneakyhydra/sneakdex/crawler/internal/config"
	"github.com/sneakyhydra/sneakdex/crawler/internal/logger"
	"github.com/sneakyhydra/sneakdex/crawler/internal/metrics"
)

type healthServer struct {
	httpServer    *http.Server
	redisClient   *redis.Client
	kafkaProducer sarama.AsyncProducer
	stats         *metrics.Metrics
}

// Start launches the health server and gracefully shuts down on signal.
func Start(wg *sync.WaitGroup, shutdown chan struct{}, redisClient *redis.Client, kafkaProducer sarama.AsyncProducer, stats *metrics.Metrics) {
	log := logger.GetLogger()
	cfg := config.GetConfig()

	mux := http.NewServeMux()
	hs := &healthServer{
		redisClient:   redisClient,
		kafkaProducer: kafkaProducer,
		stats:         stats,
	}

	mux.HandleFunc("/health", hs.handleHealth)
	mux.HandleFunc("/metrics", hs.handleMetrics)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HealthCheckPort),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	hs.httpServer = server

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Infof("Health check server starting on port %d", cfg.HealthCheckPort)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Errorf("Health check server error: %v", err)
		}
	}()

	// Graceful shutdown
	go func() {
		<-shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Errorf("Health check server shutdown error: %v", err)
		} else {
			log.Info("Health check server shut down gracefully")
		}
	}()
}
