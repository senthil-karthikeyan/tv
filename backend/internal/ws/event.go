// Package ws implements the WebSocket hub and connection handler for the OTT Dashboard backend.
// It provides real-time event delivery to connected frontend clients whenever backend data changes.
package ws

import "time"

// ── Event types ───────────────────────────────────────────────────────────────

// Scene event types emitted when scene data changes.
const (
	EventSceneCreated = "scene.created"
	EventSceneUpdated = "scene.updated"
	EventSceneDeleted = "scene.deleted"
)

// User event types emitted when user data changes.
const (
	EventUserCreated = "user.created"
	EventUserUpdated = "user.updated"
	EventUserDeleted = "user.deleted"
)

// Settings event types emitted when settings change.
const (
	EventSettingsUpdated = "settings.updated"
)

// Telemetry event types emitted when telemetry is recorded.
const (
	EventTelemetryRecorded = "telemetry.recorded"
)

// ── Operation identifiers ─────────────────────────────────────────────────────

// Operation describes the kind of change that triggered an event.
const (
	OpCreate = "create"
	OpUpdate = "update"
	OpDelete = "delete"
)

// ── Event ─────────────────────────────────────────────────────────────────────

// Event is the JSON payload sent over WebSocket to all connected clients when
// backend data changes. The Data field carries the full updated resource on
// create/update, and is omitted on delete.
type Event struct {
	// Type is a dot-separated resource.operation string, e.g. "scene.updated".
	Type string `json:"type"`
	// Resource identifies the kind of resource affected, e.g. "scene".
	Resource string `json:"resource"`
	// ResourceID is the unique identifier of the affected record.
	ResourceID string `json:"resource_id,omitempty"`
	// Operation is the action performed: "create", "update", or "delete".
	Operation string `json:"operation"`
	// Data is the full resource body after the mutation. Omitted for deletes.
	Data any `json:"data,omitempty"`
	// Timestamp is the UTC time the event was created on the server.
	Timestamp time.Time `json:"timestamp"`
}
