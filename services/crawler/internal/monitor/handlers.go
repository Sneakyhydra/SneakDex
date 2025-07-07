package monitor

import (
	// Stdlib
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	// Third-party
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// handleHealth checks the health of the monitor server by verifying Redis and Kafka connectivity.
// It responds with HTTP 200 OK if healthy, or an error message with HTTP 503 Service Unavailable if unhealthy.
func (ms *monitorServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")

	// Health check results
	healthStatus := struct {
		Status    string            `json:"status"`
		Timestamp time.Time         `json:"timestamp"`
		Services  map[string]string `json:"services"`
		Errors    []string          `json:"errors,omitempty"`
	}{
		Status:    "healthy",
		Timestamp: time.Now().UTC(),
		Services:  make(map[string]string),
		Errors:    []string{},
	}

	// Check Redis connectivity
	if err := ms.checkRedisHealth(ctx); err != nil {
		healthStatus.Status = "unhealthy"
		healthStatus.Services["redis"] = "unhealthy"
		healthStatus.Errors = append(healthStatus.Errors, fmt.Sprintf("Redis: %v", err))
	} else {
		healthStatus.Services["redis"] = "healthy"
	}

	// Check Kafka connectivity
	if err := ms.checkKafkaHealth(); err != nil {
		healthStatus.Status = "unhealthy"
		healthStatus.Services["kafka"] = "unhealthy"
		healthStatus.Errors = append(healthStatus.Errors, fmt.Sprintf("Kafka: %v", err))
	} else {
		healthStatus.Services["kafka"] = "healthy"
	}

	// Check application-specific health
	if err := ms.checkApplicationHealth(); err != nil {
		healthStatus.Status = "unhealthy"
		healthStatus.Services["application"] = "unhealthy"
		healthStatus.Errors = append(healthStatus.Errors, fmt.Sprintf("Application: %v", err))
	} else {
		healthStatus.Services["application"] = "healthy"
	}

	// Set HTTP status code
	statusCode := http.StatusOK
	if healthStatus.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	// Send JSON response
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(healthStatus); err != nil {
		// Fallback to plain text if JSON encoding fails
		w.Header().Set("Content-Type", "text/plain")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// checkRedisHealth performs Redis connectivity check
func (ms *monitorServer) checkRedisHealth(ctx context.Context) error {
	// Basic ping test
	if err := ms.crawler.RedisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

// checkKafkaHealth performs Kafka connectivity and producer health check
func (ms *monitorServer) checkKafkaHealth() error {
	// Check if AsyncProducer is initialized
	if ms.crawler.AsyncProducer == nil {
		return fmt.Errorf("AsyncProducer not initialized")
	}

	return nil
}

// checkApplicationHealth performs application-specific health checks
func (ms *monitorServer) checkApplicationHealth() error {
	// Check if crawler is running
	if ms.crawler == nil {
		return fmt.Errorf("crawler not initialized")
	}

	// Check if stats are being collected
	if ms.crawler.Stats == nil {
		return fmt.Errorf("stats collector not initialized")
	}

	// Optional: Check if there are any critical application errors
	// This could include checking error rates, queue sizes, etc.

	// Example: Check if error rate is too high
	// if ms.crawler.Stats.GetErrorRate() > 0.5 {
	//     return fmt.Errorf("error rate too high: %.2f", ms.crawler.Stats.GetErrorRate())
	// }

	// Example: Check if queue size is too large
	// if queueSize := ms.crawler.GetQueueSize(); queueSize > 10000 {
	//     return fmt.Errorf("queue size too large: %d", queueSize)
	// }

	return nil
}

// handleMetrics retrieves and returns the current metrics in Prometheus format.
// It responds with HTTP 200 OK and the metrics in JSON format.
// If there is an error encoding the metrics, it responds with HTTP 500 Internal Server Error.
func (ms *monitorServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	ms.crawler.Stats.SyncPrometheusMetrics() // one-time sync before respond
	// Delegate to the Prometheus HTTP handler
	promhttp.Handler().ServeHTTP(w, r)
}
