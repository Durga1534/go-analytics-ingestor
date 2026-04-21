package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/Durga1534/go-analytics-ingestor/internal/models"
	"github.com/redis/go-redis/v9"
)

// MetricsHandler handles GET requests to fetch system metrics
func MetricsHandler(rdb *redis.Client, streamName string, processedCount *int64, startTime time.Time, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		// Check stream pending count
		groups, _ := rdb.XInfoGroups(context.Background(), streamName).Result()
		var pending int64
		if len(groups) > 0 {
			pending = groups[0].Pending
		}

		metrics := models.SystemMetrics{
			EventsProcessed: atomic.LoadInt64(processedCount),
			StreamPending:   pending,
			MemoryUsageMB:   m.Alloc / 1024 / 1024,
			UptimeSeconds:   int64(time.Since(startTime).Seconds()),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metrics)
	}
}
