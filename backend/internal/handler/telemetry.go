package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"tv/internal/model"
	"tv/internal/store"
	"tv/internal/ws"
)

// TelemetryHandler handles ingestion and querying of telemetry and client events.
type TelemetryHandler struct {
	telemetryStore *store.TelemetryStore
	settingsStore  *store.SettingsStore
	broadcaster    ws.Broadcaster
}

// NewTelemetryHandler constructs a TelemetryHandler backed by stores and broadcaster.
func NewTelemetryHandler(ts *store.TelemetryStore, ss *store.SettingsStore, b ws.Broadcaster) *TelemetryHandler {
	return &TelemetryHandler{
		telemetryStore: ts,
		settingsStore:  ss,
		broadcaster:    b,
	}
}

// Ingest handles POST /api/v1/telemetry/events.
// Accepts either a single JSON event object or a JSON array of event objects.
// Returns 202 Accepted.
func (h *TelemetryHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	data, err := io.ReadAll(r.Body)
	if err != nil {
		WriteBadRequest(w, "failed to read request body")
		return
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		WriteBadRequest(w, "request body cannot be empty")
		return
	}

	var reqs []model.IngestEventRequest
	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &reqs); err != nil {
			WriteBadRequest(w, "invalid JSON array")
			return
		}
	} else if trimmed[0] == '{' {
		var single model.IngestEventRequest
		if err := json.Unmarshal(trimmed, &single); err != nil {
			WriteBadRequest(w, "invalid JSON object")
			return
		}
		reqs = []model.IngestEventRequest{single}
	} else {
		WriteBadRequest(w, "invalid JSON payload")
		return
	}

	if len(reqs) == 0 {
		WriteBadRequest(w, "event batch cannot be empty")
		return
	}

	// Validate all events before processing
	for i := range reqs {
		if msg := validateIngestEvent(&reqs[i]); msg != "" {
			WriteValidationError(w, msg)
			return
		}
	}

	// Check if telemetry tracking is enabled in application settings
	if h.settingsStore != nil && !h.settingsStore.Get().TelemetryEnabled {
		WriteData(w, http.StatusAccepted, map[string]any{
			"accepted": 0,
			"status":   "disabled",
		})
		return
	}

	recorded, err := h.telemetryStore.RecordBatch(reqs)
	if err != nil {
		WriteInternalError(w)
		return
	}

	// Broadcast recorded events to connected WebSocket clients for real-time monitoring
	if h.broadcaster != nil {
		for _, evt := range recorded {
			h.broadcaster.Broadcast(ws.Event{
				Type:       ws.EventTelemetryRecorded,
				Resource:   "telemetry",
				ResourceID: evt.ID,
				Operation:  ws.OpCreate,
				Data:       evt,
			})
		}
	}

	WriteData(w, http.StatusAccepted, map[string]any{
		"accepted": len(recorded),
	})
}

// List handles GET /api/v1/telemetry/events.
// Returns stored telemetry events, newest first, with optional type, source, and limit query filters.
// Requires API key authentication.
func (h *TelemetryHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	eventType := query.Get("type")
	source := query.Get("source")
	limit := 50

	if lStr := query.Get("limit"); lStr != "" {
		if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	events := h.telemetryStore.List(eventType, source, limit)
	WriteCollection(w, events, len(events))
}

// Summary handles GET /api/v1/telemetry/summary.
// Returns aggregate statistics of recorded telemetry events.
// Requires API key authentication.
func (h *TelemetryHandler) Summary(w http.ResponseWriter, r *http.Request) {
	summary := h.telemetryStore.Summary()
	WriteData(w, http.StatusOK, summary)
}

// ── validation ────────────────────────────────────────────────────────────────

func validateIngestEvent(req *model.IngestEventRequest) string {
	if strings.TrimSpace(req.Type) == "" {
		return "type is required"
	}
	if len(req.Type) > 50 {
		return "type must be 50 characters or fewer"
	}
	if strings.TrimSpace(req.Source) == "" {
		return "source is required"
	}
	if len(req.Source) > 50 {
		return "source must be 50 characters or fewer"
	}
	if len(req.UserID) > 100 {
		return "user_id must be 100 characters or fewer"
	}
	if len(req.SessionID) > 100 {
		return "session_id must be 100 characters or fewer"
	}
	return ""
}
