package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Durga1534/go-analytics-ingestor/internal/cache"
	"github.com/Durga1534/go-analytics-ingestor/internal/config"
	"github.com/Durga1534/go-analytics-ingestor/internal/handlers"
	"github.com/Durga1534/go-analytics-ingestor/internal/logger"
	"github.com/Durga1534/go-analytics-ingestor/internal/persistence"
	"github.com/Durga1534/go-analytics-ingestor/internal/worker"
)

func main() {
	// Initialize logger
	logger.Initialize()
	log := slog.Default()

	// Load configuration
	cfg := config.Load()

	// Validate required configuration
	if cfg.RedisURL == "" {
		log.Error("REDIS_URL is not set")
		os.Exit(1)
	}
	if cfg.DatabaseURL == "" {
		log.Error("DATABASE_URL is not set")
		os.Exit(1)
	}

	ctx := context.Background()

	// Initialize Redis
	rdb, err := cache.New(ctx, cfg.RedisURL, log)
	if err != nil {
		log.Error("Redis connection failed", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// Initialize consumer group
	if err := rdb.InitializeStream(ctx, "analytics_stream", "ingestor_group"); err != nil {
		log.Error("Failed to initialize consumer group", "error", err)
		os.Exit(1)
	}

	// Initialize Database
	db, err := persistence.New(ctx, cfg.DatabaseURL, log)
	if err != nil {
		log.Error("Database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Metrics tracking
	var processedCount int64
	startTime := time.Now()

	// Start Distributed Worker
	w := worker.New(rdb.Client, db, log, &processedCount)
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	go w.Start(workerCtx, cfg.WorkerName)

	// Setup HTTP routes
	http.HandleFunc("/ingest", handlers.IngestHandler(rdb.Client, "analytics_stream", log))
	http.HandleFunc("/metrics", handlers.MetricsHandler(rdb.Client, "analytics_stream", &processedCount, startTime, log))

	// Create HTTP server
	server := &http.Server{Addr: ":" + cfg.Port}

	// Start server in goroutine
	go func() {
		log.Info("Distributed Engine Starting", "port", cfg.Port, "worker", cfg.WorkerName)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server crashed", "error", err)
		}
	}()

	// Graceful shutdown handler
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info("Shutdown signal received. Closing connections...")
	cancelWorker()
	server.Shutdown(ctx)
	log.Info("System offline")
}
