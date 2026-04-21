package cache

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// Redis wraps the Redis client with application-specific methods
type Redis struct {
	Client *redis.Client
	Logger *slog.Logger
}

// New creates and initializes a Redis connection
func New(ctx context.Context, redisURL string, logger *slog.Logger) (*Redis, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	// Test connection
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}

	logger.Info("✅ Connected to Redis")

	return &Redis{
		Client: client,
		Logger: logger,
	}, nil
}

// InitializeStream sets up the Redis stream and consumer group
func (r *Redis) InitializeStream(ctx context.Context, streamName, consumerGroup string) error {
	// MKSTREAM creates the stream if it doesn't exist
	err := r.Client.XGroupCreateMkStream(ctx, streamName, consumerGroup, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

// Close closes the Redis connection
func (r *Redis) Close() error {
	return r.Client.Close()
}
