package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/Durga1534/go-analytics-ingestor/internal/models"
	"github.com/Durga1534/go-analytics-ingestor/internal/persistence"
	"github.com/redis/go-redis/v9"
)

// DistributedWorker manages async event processing from Redis stream
type DistributedWorker struct {
	Redis          *redis.Client
	DB             *persistence.Database
	Logger         *slog.Logger
	ProcessedCount *int64
	StreamName     string
	ConsumerGroup  string
	DLQName        string
}

// New creates a new distributed worker
func New(rdb *redis.Client, db *persistence.Database, logger *slog.Logger, processedCount *int64) *DistributedWorker {
	return &DistributedWorker{
		Redis:          rdb,
		DB:             db,
		Logger:         logger,
		ProcessedCount: processedCount,
		StreamName:     "analytics_stream",
		ConsumerGroup:  "ingestor_group",
		DLQName:        "analytics_dlq",
	}
}

// Start begins the distributed worker consuming events from Redis stream
func (w *DistributedWorker) Start(ctx context.Context, name string) {
	const batchSize = 10
	ticker := time.NewTicker(10 * time.Second)

	var batch []models.Event
	var messageIDs []string

	w.Logger.Info("Distributed worker ready", "consumer_name", name)

	for {
		select {
		case <-ctx.Done():
			w.Logger.Info("Worker shutting down")
			if len(batch) > 0 {
				w.Logger.Info("Flushing remaining batch on shutdown", "count", len(batch))
				w.flushAndAck(batch, messageIDs)
			}
			return

		case <-ticker.C:
			if len(batch) > 0 {
				w.Logger.Info("Timer-based flush", "count", len(batch))
				if w.flushAndAck(batch, messageIDs) {
					batch = nil
					messageIDs = nil
				}
			}

		default:
			// Read from consumer group
			// ">" means: "New messages never delivered to others"
			entries, err := w.Redis.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    w.ConsumerGroup,
				Consumer: name,
				Streams:  []string{w.StreamName, ">"},
				Count:    batchSize,
				Block:    2 * time.Second,
			}).Result()

			if err != nil {
				continue
			}

			for _, stream := range entries {
				for _, msg := range stream.Messages {
					var e models.Event
					data := msg.Values["payload"].(string)

					if err := json.Unmarshal([]byte(data), &e); err != nil {
						// DEAD LETTER QUEUE (DLQ)
						w.Logger.Error("Poison pill detected. Moving to DLQ", "data", data)
						w.Redis.LPush(ctx, w.DLQName, data)
						w.Redis.XAck(ctx, w.StreamName, w.ConsumerGroup, msg.ID)
						continue
					}

					batch = append(batch, e)
					messageIDs = append(messageIDs, msg.ID)
				}
			}

			if len(batch) >= batchSize {
				w.Logger.Info("Capacity-based flush", "count", len(batch))
				if w.flushAndAck(batch, messageIDs) {
					batch = nil
					messageIDs = nil
				}
			}
		}
	}
}

// flushAndAck persists batch to database and acknowledges consumption
func (w *DistributedWorker) flushAndAck(batch []models.Event, ids []string) bool {
	ctx := context.Background()

	// Persist batch to database
	if err := w.DB.BatchInsertEvents(ctx, batch); err != nil {
		return false
	}

	// Acknowledge consumption only after successful DB write
	err := w.Redis.XAck(ctx, w.StreamName, w.ConsumerGroup, ids...).Err()
	if err != nil {
		w.Logger.Error("Failed to ACK messages", "error", err)
		return false
	}

	atomic.AddInt64(w.ProcessedCount, int64(len(batch)))
	return true
}
