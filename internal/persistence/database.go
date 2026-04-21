package persistence

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Durga1534/go-analytics-ingestor/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Database wraps the connection pool and provides database operations
type Database struct {
	Pool   *pgxpool.Pool
	Logger *slog.Logger
}

// New creates and initializes a database connection pool
func New(ctx context.Context, databaseURL string, logger *slog.Logger) (*Database, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	logger.Info("✅ Connected to PostgreSQL Pool")

	return &Database{
		Pool:   pool,
		Logger: logger,
	}, nil
}

// BatchInsertEvents inserts a batch of events into the analytics table
func (db *Database) BatchInsertEvents(ctx context.Context, batch []models.Event) error {
	if len(batch) == 0 {
		return nil
	}

	// Build batch insert query
	query := `INSERT INTO analytics (id, type, payload, timestamp) VALUES `
	values := []interface{}{}

	for i, e := range batch {
		p := i * 4
		query += fmt.Sprintf("($%d, $%d, $%d, $%d),", p+1, p+2, p+3, p+4)
		values = append(values, e.ID, e.Type, e.Payload, e.Timestamp)
	}
	query = query[:len(query)-1]

	// Execute batch insert
	_, err := db.Pool.Exec(ctx, query, values...)
	if err != nil {
		db.Logger.Error("Database persistence failed", "error", err)
		return err
	}

	return nil
}

// Close closes the database connection pool
func (db *Database) Close() {
	db.Pool.Close()
}
