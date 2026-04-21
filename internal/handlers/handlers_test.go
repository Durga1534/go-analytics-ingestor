package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Durga1534/go-analytics-ingestor/internal/models"
	"github.com/redis/go-redis/v9"
)

func TestIngestHandler(t *testing.T) {
	// Mock Redis client (you'd use testcontainers or miniredis for integration tests)
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	tests := []struct {
		name           string
		method         string
		body           interface{}
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Valid event ingestion",
			method:         http.MethodPost,
			body:           models.Event{ID: "evt_123", Type: "page_view", Payload: "test", Timestamp: time.Now()},
			expectedStatus: http.StatusAccepted,
			expectError:    false,
		},
		{
			name:           "Invalid HTTP method",
			method:         http.MethodGet,
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				body, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, "/ingest", bytes.NewReader(body))
			} else {
				req = httptest.NewRequest(tt.method, "/ingest", nil)
			}

			w := httptest.NewRecorder()
			handler := IngestHandler(rdb, "test_stream", nil)
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
