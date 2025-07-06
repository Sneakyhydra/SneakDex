package monitor

import (
	// Stdlib
	"context"
	"fmt"
	"net/http"
	"time"

	// Third-party
	"github.com/IBM/sarama"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// handleHealth checks the health of the monitor server by verifying Redis and Kafka connectivity.
// It responds with HTTP 200 OK if healthy, or an error message with HTTP 503 Service Unavailable if unhealthy.
func (ms *monitorServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Basic Redis check
	if err := ms.crawler.RedisClient.Ping(ctx).Err(); err != nil {
		http.Error(w, fmt.Sprintf("Redis unhealthy: %v", err), http.StatusServiceUnavailable)
		return
	}

	// Basic Kafka check - verify AsyncProducer is running
	if ms.crawler.AsyncProducer == nil {
		http.Error(w, "Kafka AsyncProducer not initialized", http.StatusServiceUnavailable)
		return
	}
	
	// Try to send a health check message (non-blocking)
	select {
	case ms.crawler.AsyncProducer.Input() <- &sarama.ProducerMessage{
		Topic: "health-check",
		Value: sarama.StringEncoder("health-check"),
	}:
		// Message queued successfully
	case <-time.After(100 * time.Millisecond):
		// Channel is full or producer is not responsive
		http.Error(w, "Kafka AsyncProducer channel full or unresponsive", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// handleMetrics retrieves and returns the current metrics in Prometheus format.
// It responds with HTTP 200 OK and the metrics in JSON format.
// If there is an error encoding the metrics, it responds with HTTP 500 Internal Server Error.
func (ms *monitorServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	ms.crawler.Stats.SyncPrometheusMetrics() // one-time sync before respond
	// Delegate to the Prometheus HTTP handler
	promhttp.Handler().ServeHTTP(w, r)
}
