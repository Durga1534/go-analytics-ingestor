package models

import "time"

// Event represents an analytics event payload
type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// SystemMetrics contains runtime metrics for the system
type SystemMetrics struct {
	EventsProcessed int64  `json:"events_processed"`
	StreamPending   int64  `json:"stream_pending"`
	MemoryUsageMB   uint64 `json:"memory_usage_mb"`
	UptimeSeconds   int64  `json:"uptime_seconds"`
}
