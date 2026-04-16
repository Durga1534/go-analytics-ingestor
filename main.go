package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	http.HandleFunc("/ingest", ingestHandler)

	fmt.Println("Server starting on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server failed: %s\n", err)
	}
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

func processEvent(e Event) {
	fmt.Printf("[%s] Logged: %s (ID: %s)\n", e.Timestamp.Format(time.RFC3339), e.Type, e.ID)
}
