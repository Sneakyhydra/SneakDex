package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func (hs *healthServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Basic Redis check
	if err := hs.redisClient.Ping(ctx).Err(); err != nil {
		http.Error(w, fmt.Sprintf("Redis unhealthy: %v", err), http.StatusServiceUnavailable)
		return
	}

	// Basic Kafka check
	if hs.kafkaProducer == nil {
		http.Error(w, "Kafka producer not initialized", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (hs *healthServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := hs.stats.GetStats()

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
	}
}
