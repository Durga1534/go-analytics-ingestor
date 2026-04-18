package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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

var (
	rdb    *redis.Client
	dbPool *pgxpool.Pool
	ctx    = context.Background()
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// 1. Redis Setup
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("REDIS_URL is not set")
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	rdb = redis.NewClient(opt)

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	fmt.Println("✅ Connected to Redis (Upstash)")

	// 2. PostgreSQL Connection Pool
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		var dberr error
		dbPool, dberr = pgxpool.New(ctx, dbURL)
		if dberr != nil {
			log.Fatalf("Unable to connect to database: %v", dberr)
		}
		defer dbPool.Close()
		fmt.Println("✅ Connected to PostgreSQL Pool")
	}

	// 3. Setup Routes
	http.HandleFunc("/ingest", ingestHandler)

	// 4. Start Background Worker
	go startWorker()

	// 5. Dynamic Port Logic & Server Start
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default if not specified in .env
	}

	go func() {
		fmt.Printf("🚀 Server starting on :%s...\n", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %s", err)
		}
	}()

	// 6. Graceful Shutdown Listener
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop // Block here until Ctrl+C
	fmt.Println("\nShutting down gracefully...")
}

func ingestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var e Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	payload, _ := json.Marshal(e)
	if err := rdb.LPush(ctx, "analytics_queue", payload).Err(); err != nil {
		log.Printf("Redis Error: %v", err)
		http.Error(w, "Queue failure", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Event accepted and queued"))
}

func startWorker() {
	fmt.Println("👷 Worker active: Batching 10 events or 10s")

	const batchSize = 10
	var batch []Event

	eventChan := make(chan Event)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			result, err := rdb.BLPop(ctx, 0, "analytics_queue").Result()
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}

			var e Event
			if err := json.Unmarshal([]byte(result[1]), &e); err == nil {
				eventChan <- e
			}
		}
	}()

	for {
		select {
		case e := <-eventChan:
			batch = append(batch, e)
			if len(batch) >= batchSize {
				flush(batch)
				batch = nil
			}

		case <-ticker.C:
			if len(batch) > 0 {
				fmt.Print("(Timer Flush) ")
				flush(batch)
				batch = nil
			}
		}
	}
}

func flush(batch []Event) {
	if len(batch) == 0 {
		return
	}

	// High performance batch insert: 1 trip to the DB
	query := `INSERT INTO analytics (id, type, payload, timestamp) VALUES `
	values := []interface{}{}

	for i, e := range batch {
		p := i * 4
		// Fixed: Added missing comma between first and second placeholder
		query += fmt.Sprintf("($%d, $%d, $%d, $%d),", p+1, p+2, p+3, p+4)
		values = append(values, e.ID, e.Type, e.Payload, e.Timestamp)
	}
	query = query[:len(query)-1] // Remove trailing comma

	_, err := dbPool.Exec(ctx, query, values...)
	if err != nil {
		log.Printf("Database Batch Insert Error: %v", err)
	} else {
		fmt.Printf("Successfully persisted batch of %d events to PostgreSQL\n", len(batch))
	}
}
