package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"tv/internal/model"
	"tv/internal/store"
	"tv/internal/ws"
)

// SettingsHandler handles HTTP requests for the /api/v1/settings resource.
type SettingsHandler struct {
	store       *store.SettingsStore
	broadcaster ws.Broadcaster
}

// NewSettingsHandler creates a SettingsHandler backed by the provided store and broadcaster.
// If broadcaster is nil, real-time event broadcasts are skipped.
func NewSettingsHandler(s *store.SettingsStore, b ws.Broadcaster) *SettingsHandler {
	return &SettingsHandler{
		store:       s,
		broadcaster: b,
	}
}

// Get handles GET /api/v1/settings.
// Returns the current application settings.
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	st := h.store.Get()
	WriteData(w, http.StatusOK, st)
}

// Update handles PUT /api/v1/settings.
// Partially updates settings with non-null fields provided in the body.
// Requires API key authentication.
func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req model.UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "invalid JSON body")
		return
	}

	if msg := validateUpdateSettings(&req); msg != "" {
		WriteValidationError(w, msg)
		return
	}

	st := h.store.Update(&req)

	if h.broadcaster != nil {
		h.broadcaster.Broadcast(ws.Event{
			Type:      ws.EventSettingsUpdated,
			Resource:  "settings",
			Operation: ws.OpUpdate,
			Data:      st,
		})
	}

	WriteData(w, http.StatusOK, st)
}

// Reset handles POST /api/v1/settings/reset.
// Restores all settings to system defaults.
// Requires API key authentication.
func (h *SettingsHandler) Reset(w http.ResponseWriter, r *http.Request) {
	st := h.store.Reset()

	if h.broadcaster != nil {
		h.broadcaster.Broadcast(ws.Event{
			Type:      ws.EventSettingsUpdated,
			Resource:  "settings",
			Operation: ws.OpUpdate,
			Data:      st,
		})
	}

	WriteData(w, http.StatusOK, st)
}

// ── validation ────────────────────────────────────────────────────────────────

func validateUpdateSettings(req *model.UpdateSettingsRequest) string {
	if req.Theme != nil {
		switch *req.Theme {
		case "dark", "light", "system":
			// valid
		default:
			return "theme must be one of: dark, light, system"
		}
	}

	if req.Language != nil {
		lang := strings.TrimSpace(*req.Language)
		if len(lang) < 2 || len(lang) > 5 {
			return "language must be a valid 2 to 5 character code"
		}
	}

	if req.AutoRefreshInterval != nil {
		if *req.AutoRefreshInterval < 5 || *req.AutoRefreshInterval > 300 {
			return "auto_refresh_interval must be between 5 and 300 seconds"
		}
	}

	if req.MaxStreamQuality != nil {
		switch *req.MaxStreamQuality {
		case "4K", "1080p", "720p", "480p":
			// valid
		default:
			return "max_stream_quality must be one of: 4K, 1080p, 720p, 480p"
		}
	}

	return ""
}
