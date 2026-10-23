package model

import "time"

// TelemetryEvent represents an event tracked from the client or runtime system.
type TelemetryEvent struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`                 // e.g. "video_start", "scene_selected", "error"
	Source    string         `json:"source"`               // e.g. "tv-launcher", "player", "web"
	UserID    string         `json:"user_id,omitempty"`    // optional user identifier
	SessionID string         `json:"session_id,omitempty"` // optional session identifier
	Payload   map[string]any `json:"payload,omitempty"`    // flexible event metadata
	Timestamp time.Time      `json:"timestamp"`
}

// IngestEventRequest holds the client-supplied fields for a single telemetry event.
type IngestEventRequest struct {
	Type      string         `json:"type"`
	Source    string         `json:"source"`
	UserID    string         `json:"user_id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	Timestamp *time.Time     `json:"timestamp,omitempty"`
}

// TelemetrySummary provides aggregate counts and breakdowns of recorded telemetry.
type TelemetrySummary struct {
	TotalRecorded int            `json:"total_recorded"`
	ByType        map[string]int `json:"by_type"`
	BySource      map[string]int `json:"by_source"`
	LastEventAt   *time.Time     `json:"last_event_at,omitempty"`
}
