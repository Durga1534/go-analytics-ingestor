package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog" // Structured Logging
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic" // For thread-safe metrics
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

// Event matches your JSON analytics payload
type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// SystemMetrics tracks the "Health" of your application
type SystemMetrics struct {
	EventsProcessed int64  `json:"events_processed"`
	QueueLength     int64  `json:"queue_length"`
	MemoryUsageMB   uint64 `json:"memory_usage_mb"`
	UptimeSeconds   int64  `json:"uptime_seconds"`
}

var (
	rdb       *redis.Client
	dbPool    *pgxpool.Pool
	ctx       = context.Background()
	startTime = time.Now()
	logger    *slog.Logger

	// Atomic counters for thread-safe metrics
	processedCount int64
)

func main() {
	// 1. Initialize Structured Logger
	// In production, you'd use slog.NewJSONHandler(os.Stdout, nil)
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found, using system environment variables")
	}

	// 2. Redis Setup
	redisURL := os.Getenv("REDIS_URL")
	opt, _ := redis.ParseURL(redisURL)
	rdb = redis.NewClient(opt)
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		logger.Error("Could not connect to Redis", "error", err)
		os.Exit(1)
	}
	logger.Info("Connected to Redis", "provider", "Upstash")

	// 3. PostgreSQL Setup
	var dberr error
	dbPool, dberr = pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if dberr != nil {
		logger.Error("Database connection failed", "error", dberr)
		os.Exit(1)
	}
	defer dbPool.Close()
	logger.Info("Connected to PostgreSQL", "pool", "initialized")

	// 4. Routes
	http.HandleFunc("/ingest", ingestHandler)
	http.HandleFunc("/metrics", metricsHandler)

	// 5. Worker & Server Start
	workerDone := make(chan bool)
	go startWorker(workerDone)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		logger.Info("Server starting", "port", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			logger.Error("Server crashed", "error", err)
		}
	}()

	// 6. Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	logger.Info("Shutdown signal received. Cleaning up...")
	// In a full production app, you'd trigger a final flush here
	logger.Info("System offline")
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	qLen, _ := rdb.LLen(ctx, "analytics_queue").Result()

	currentMetrics := SystemMetrics{
		EventsProcessed: atomic.LoadInt64(&processedCount),
		QueueLength:     qLen,
		MemoryUsageMB:   m.Alloc / 1024 / 1024,
		UptimeSeconds:   int64(time.Since(startTime).Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currentMetrics)
}

func ingestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var e Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		logger.Warn("Invalid JSON payload received")
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	payload, _ := json.Marshal(e)
	if err := rdb.LPush(ctx, "analytics_queue", payload).Err(); err != nil {
		logger.Error("Failed to queue event", "id", e.ID, "error", err)
		http.Error(w, "Queue failure", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func startWorker(done chan bool) {
	const batchSize = 10
	var batch []Event
	ticker := time.NewTicker(10 * time.Second)

	logger.Info("Worker started", "batch_limit", batchSize, "timer", "10s")

	for {
		select {
		case <-ticker.C:
			if len(batch) > 0 {
				logger.Info("Timer-based flush triggered", "count", len(batch))
				flush(batch)
				batch = nil
			}
		default:
			// Non-blocking fetch from Redis
			result, err := rdb.BLPop(ctx, 1*time.Second, "analytics_queue").Result()
			if err != nil {
				continue
			}

			var e Event
			if err := json.Unmarshal([]byte(result[1]), &e); err == nil {
				batch = append(batch, e)
				if len(batch) >= batchSize {
					logger.Info("Capacity-based flush triggered", "count", len(batch))
					flush(batch)
					batch = nil
				}
			}
		}
	}
}

func flush(batch []Event) {
	query := `INSERT INTO analytics (id, type, payload, timestamp) VALUES `
	values := []interface{}{}

	for i, e := range batch {
		p := i * 4
		query += fmt.Sprintf("($%d, $%d, $%d, $%d),", p+1, p+2, p+3, p+4)
		values = append(values, e.ID, e.Type, e.Payload, e.Timestamp)
	}
	query = query[:len(query)-1]

	start := time.Now()
	_, err := dbPool.Exec(ctx, query, values...)
	duration := time.Since(start)

	if err != nil {
		logger.Error("Database batch insert failed", "error", err, "batch_size", len(batch))
	} else {
		atomic.AddInt64(&processedCount, int64(len(batch)))
		logger.Info("Batch persisted successfully",
			"count", len(batch),
			"latency_ms", duration.Milliseconds(),
			"total_processed", atomic.LoadInt64(&processedCount))
	}
}
