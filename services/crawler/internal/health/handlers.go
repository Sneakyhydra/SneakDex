package health

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func checkHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Also check Redis and Kafka health here for a more robust health check
	redisStatus := "ok"
	kafkaStatus := "ok"

	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second) // Use request context, short timeout
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		redisStatus = fmt.Sprintf("error: %v", err)
	}

	// Kafka health check:
	// A non-invasive check is just to ensure the producer is initialized (not nil).
	// Sarama's SyncProducer handles internal connection management and retries.
	if producer == nil {
		kafkaStatus = "error: Kafka producer uninitialized"
	}
	// For more advanced Kafka health, you might consider:
	// 1. Trying to send a trivial message to a dedicated health-check topic.
	// 2. Accessing the underlying Sarama Client's metadata (if exposed or managed separately).
	// For this setup, a non-nil producer indicates it's attempting to work.

	status := "healthy"
	if strings.HasPrefix(redisStatus, "error") || strings.HasPrefix(kafkaStatus, "error") {
		status = "unhealthy"
	}

	fmt.Fprintf(w, `{"status":"%s","timestamp":"%s","dependencies":{"redis":"%s","kafka":"%s"}}`,
		status, time.Now().Format(time.RFC3339), redisStatus, kafkaStatus)
}

func checkMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := stats.GetStats()
	fmt.Fprintf(w, `{
            "pages_processed": %d,
            "pages_successful": %d,
            "pages_failed": %d,
            "kafka_successful": %d,
            "kafka_failed": %d,
            "redis_successful": %d,
            "redis_failed": %d,
            "uptime_seconds": %.2f
        }`,
		stats["pages_processed"],
		stats["pages_successful"],
		stats["pages_failed"],
		stats["kafka_successful"],
		stats["kafka_failed"],
		stats["redis_successful"],
		stats["redis_failed"],
		stats["uptime_seconds"].(float64), // Ensure type assertion for float
	)
}
