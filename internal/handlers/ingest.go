package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Durga1534/go-analytics-ingestor/internal/models"
	"github.com/redis/go-redis/v9"
)

// IngestHandler handles POST requests to ingest analytics events
func IngestHandler(rdb *redis.Client, streamName string, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var e models.Event
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			logger.Warn("Invalid JSON received", "error", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Write to Redis stream
		payload, _ := json.Marshal(e)
		err := rdb.XAdd(context.Background(), &redis.XAddArgs{
			Stream: streamName,
			Values: map[string]interface{}{"payload": string(payload)},
		}).Err()

		if err != nil {
			logger.Error("Failed to stream event", "error", err)
			http.Error(w, "Ingestion failure", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
