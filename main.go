package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	rdb *redis.Client
	ctx = context.Background() // Global Context for simple use cases
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, proceeding with system evv variables")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("REDIS_URL is not set in .env file")
	}
	//1 Safety: Intialize Redis and chexk connection
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	rdb = redis.NewClient(opt)

	// Verify connection (Safety First)
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("Could not connect to Upstash: %v", err)
	}
	fmt.Println("Successfully connected to Upstash Redis!")

	//2 Start the consumer (The Worker)
	go startWorker()

	http.HandleFunc("/ingest", ingestHandler)

	fmt.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func ingestHandler(w http.ResponseWriter, r *http.Request) {
	//1. Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//2. Efficently decode the body
	var e Event
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&e); err != nil {
		// The 'Fix': Handle bad JSON without panicking
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}
	payload, _ := json.Marshal(e)
	err := rdb.LPush(ctx, "analytics_queue", payload).Err()
	if err != nil {
		log.Printf("Redis Push Error: %v", err)
		http.Error(w, "Failed to queue event", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	//3. Send 202 Accepted immediately
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Event accepted"))

	//4. Fire and forget the background processing
	go func(event Event) {
		// Simulate background check
		processEvent(event)

	}(e)
}

func startWorker() {
	fmt.Println("Worker active: Batching enabled (Max 10 events or 10s)...")

	const batchSize = 10
	var batch []Event

	// Create a channel to pipe events from Redis to our select block
	eventChan := make(chan Event)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Helper Goroutine: Fetch from Redis and pipe to the channel
	go func() {
		for {
			result, err := rdb.BLPop(ctx, 0, "analytics_queue").Result()
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			var e Event
			if err := json.Unmarshal([]byte(result[1]), &e); err != nil {
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
				fmt.Print("(Timer Flush)")
				flush(batch)
				batch = nil
			}
		}
	}
}

func flush(b []Event) {
	fmt.Printf("Batch Processed: %d events\n", len(b))
}

func processEvent(e Event) {
	fmt.Printf("Worker Processed: %s | Type: %s\n", e.ID, e.Type)
}
