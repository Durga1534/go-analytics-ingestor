package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

// --- MODELS ---

type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

type SystemMetrics struct {
	EventsProcessed int64  `json:"events_processed"`
	StreamPending   int64  `json:"stream_pending"`
	MemoryUsageMB   uint64 `json:"memory_usage_mb"`
	UptimeSeconds   int64  `json:"uptime_seconds"`
}

// --- GLOBALS ---

var (
	rdb            *redis.Client
	dbPool         *pgxpool.Pool
	ctx            = context.Background()
	startTime      = time.Now()
	logger         *slog.Logger
	processedCount int64

	// Phase 6 Constants
	StreamName    = "analytics_stream"
	ConsumerGroup = "ingestor_group"
	DLQName       = "analytics_dlq"
)

func main() {
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found, using system environment variables")
	}

	// 1. Redis Setup
	opt, _ := redis.ParseURL(os.Getenv("REDIS_URL"))
	rdb = redis.NewClient(opt)

	// Phase 6: Initialize Consumer Group
	// We use MKSTREAM to create the stream if it doesn't exist
	err := rdb.XGroupCreateMkStream(ctx, StreamName, ConsumerGroup, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		logger.Error("Failed to create consumer group", "error", err)
	}

	// 2. Postgres Setup
	dbPool, err = pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.Error("Database connection failed", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// 3. Worker Configuration (Scaleable via hostname)
	workerName, _ := os.Hostname()
	if workerName == "" {
		workerName = "local-worker"
	}

	// Start Distributed Worker
	go startDistributedWorker(workerName)

	// 4. API Routes
	http.HandleFunc("/ingest", ingestHandler)
	http.HandleFunc("/metrics", metricsHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{Addr: ":" + port}
	go func() {
		logger.Info("Distributed Engine Starting", "port", port, "worker", workerName)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server crashed", "error", err)
		}
	}()

	// 5. Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("Shutdown signal received. Closing connections...")
	server.Shutdown(ctx)
	logger.Info("System offline")
}

// --- HANDLERS ---

func ingestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var e Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Phase 6: Write to Stream instead of List
	// This allows multiple consumers to read the same data if needed
	payload, _ := json.Marshal(e)
	err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamName,
		Values: map[string]interface{}{"payload": string(payload)},
	}).Err()

	if err != nil {
		logger.Error("Failed to stream event", "error", err)
		http.Error(w, "Ingestion failure", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check Stream Pending count
	groups, _ := rdb.XInfoGroups(ctx, StreamName).Result()
	var pending int64
	if len(groups) > 0 {
		pending = groups[0].Pending
	}

	json.NewEncoder(w).Encode(SystemMetrics{
		EventsProcessed: atomic.LoadInt64(&processedCount),
		StreamPending:   pending,
		MemoryUsageMB:   m.Alloc / 1024 / 1024,
		UptimeSeconds:   int64(time.Since(startTime).Seconds()),
	})
}

// --- DISTRIBUTED WORKER LOGIC ---

func startDistributedWorker(name string) {
	const batchSize = 10
	ticker := time.NewTicker(10 * time.Second)

	var batch []Event
	var messageIDs []string

	logger.Info("Distributed worker ready", "consumer_name", name)

	for {
		select {
		case <-ticker.C:
			if len(batch) > 0 {
				logger.Info("Timer-based flush", "count", len(batch))
				if flushAndAck(batch, messageIDs) {
					batch = nil
					messageIDs = nil
				}
			}
		default:
			// Read from the group
			// ">" means: "New messages never delivered to others"
			entries, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    ConsumerGroup,
				Consumer: name,
				Streams:  []string{StreamName, ">"},
				Count:    batchSize,
				Block:    2 * time.Second,
			}).Result()

			if err != nil {
				continue
			}

			for _, stream := range entries {
				for _, msg := range stream.Messages {
					var e Event
					data := msg.Values["payload"].(string)

					if err := json.Unmarshal([]byte(data), &e); err != nil {
						// DEAD LETTER QUEUE (DLQ)
						logger.Error("Poison pill detected. Moving to DLQ", "data", data)
						rdb.LPush(ctx, DLQName, data)
						rdb.XAck(ctx, StreamName, ConsumerGroup, msg.ID) // Ack to remove from stream
						continue
					}

					batch = append(batch, e)
					messageIDs = append(messageIDs, msg.ID)
				}
			}

			if len(batch) >= batchSize {
				logger.Info("Capacity-based flush", "count", len(batch))
				if flushAndAck(batch, messageIDs) {
					batch = nil
					messageIDs = nil
				}
			}
		}
	}
}

func flushAndAck(batch []Event, ids []string) bool {
	// Build Batch Insert Query
	query := `INSERT INTO analytics (id, type, payload, timestamp) VALUES `
	values := []interface{}{}

	for i, e := range batch {
		p := i * 4
		query += fmt.Sprintf("($%d, $%d, $%d, $%d),", p+1, p+2, p+3, p+4)
		values = append(values, e.ID, e.Type, e.Payload, e.Timestamp)
	}
	query = query[:len(query)-1]

	// Execute Persistence
	_, err := dbPool.Exec(ctx, query, values...)
	if err != nil {
		logger.Error("Database persistence failed", "error", err)
		return false
	}

	// ACKNOWLEDGE - The most important part of Phase 6
	// Only remove from Redis if DB write succeeded
	err = rdb.XAck(ctx, StreamName, ConsumerGroup, ids...).Err()
	if err != nil {
		logger.Error("Failed to ACK messages", "error", err)
		return false
	}

	atomic.AddInt64(&processedCount, int64(len(batch)))
	return true
}
