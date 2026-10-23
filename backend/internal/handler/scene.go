package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"tv/internal/model"
	"tv/internal/store"
	"tv/internal/ws"
)

const maxBodyBytes = 1 << 20 // 1 MiB — maximum accepted request body size

// SceneHandler handles HTTP requests for the /api/v1/scenes resource.
type SceneHandler struct {
	store       *store.SceneStore
	broadcaster ws.Broadcaster
}

// NewSceneHandler creates a SceneHandler backed by the provided store and broadcaster.
// If broadcaster is nil, real-time broadcasts are skipped.
func NewSceneHandler(s *store.SceneStore, b ws.Broadcaster) *SceneHandler {
	return &SceneHandler{
		store:       s,
		broadcaster: b,
	}
}

// List handles GET /api/v1/scenes.
// Returns all scenes ordered by Order ascending wrapped in the collection envelope.
func (h *SceneHandler) List(w http.ResponseWriter, r *http.Request) {
	scenes := h.store.List()
	WriteCollection(w, scenes, len(scenes))
}

// GetByID handles GET /api/v1/scenes/{id}.
func (h *SceneHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sc, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			WriteNotFound(w, "scene")
			return
		}
		WriteInternalError(w)
		return
	}
	WriteData(w, http.StatusOK, sc)
}

// Create handles POST /api/v1/scenes.
// Requires a valid JSON body and API key authentication (enforced by middleware).
func (h *SceneHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req model.CreateSceneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "invalid JSON body")
		return
	}

	if msg := validateCreate(&req); msg != "" {
		WriteValidationError(w, msg)
		return
	}

	sc, err := h.store.Create(&req)
	if err != nil {
		WriteInternalError(w)
		return
	}
	if h.broadcaster != nil {
		h.broadcaster.Broadcast(ws.Event{
			Type:       ws.EventSceneCreated,
			Resource:   "scene",
			ResourceID: sc.ID,
			Operation:  ws.OpCreate,
			Data:       sc,
		})
	}
	WriteData(w, http.StatusCreated, sc)
}

// Update handles PUT /api/v1/scenes/{id}.
// Only non-null fields in the request body are applied; omitted fields are unchanged.
// Requires API key authentication (enforced by middleware).
func (h *SceneHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req model.UpdateSceneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "invalid JSON body")
		return
	}

	if msg := validateUpdate(&req); msg != "" {
		WriteValidationError(w, msg)
		return
	}

	sc, err := h.store.Update(id, &req)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			WriteNotFound(w, "scene")
			return
		}
		WriteInternalError(w)
		return
	}
	if h.broadcaster != nil {
		h.broadcaster.Broadcast(ws.Event{
			Type:       ws.EventSceneUpdated,
			Resource:   "scene",
			ResourceID: sc.ID,
			Operation:  ws.OpUpdate,
			Data:       sc,
		})
	}
	WriteData(w, http.StatusOK, sc)
}

// Delete handles DELETE /api/v1/scenes/{id}.
// Returns 204 No Content on success.
// Requires API key authentication (enforced by middleware).
func (h *SceneHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.Delete(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			WriteNotFound(w, "scene")
			return
		}
		WriteInternalError(w)
		return
	}
	if h.broadcaster != nil {
		h.broadcaster.Broadcast(ws.Event{
			Type:       ws.EventSceneDeleted,
			Resource:   "scene",
			ResourceID: id,
			Operation:  ws.OpDelete,
		})
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── validation ────────────────────────────────────────────────────────────────

func validateCreate(req *model.CreateSceneRequest) string {
	if strings.TrimSpace(req.Name) == "" {
		return "name is required"
	}
	if len(req.Name) > 100 {
		return "name must be 100 characters or fewer"
	}
	if strings.TrimSpace(req.Label) == "" {
		return "label is required"
	}
	if len(req.Label) > 50 {
		return "label must be 50 characters or fewer"
	}
	if strings.TrimSpace(req.Route) == "" {
		return "route is required"
	}
	if !strings.HasPrefix(req.Route, "/") {
		return "route must start with /"
	}
	if req.Order < 0 {
		return "order must be 0 or greater"
	}
	if len(req.Icon) > 50 {
		return "icon must be 50 characters or fewer"
	}
	return ""
}

func validateUpdate(req *model.UpdateSceneRequest) string {
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return "name must not be empty"
		}
		if len(*req.Name) > 100 {
			return "name must be 100 characters or fewer"
		}
	}
	if req.Label != nil {
		if strings.TrimSpace(*req.Label) == "" {
			return "label must not be empty"
		}
		if len(*req.Label) > 50 {
			return "label must be 50 characters or fewer"
		}
	}
	if req.Route != nil {
		if strings.TrimSpace(*req.Route) == "" {
			return "route must not be empty"
		}
		if !strings.HasPrefix(*req.Route, "/") {
			return "route must start with /"
		}
	}
	if req.Order != nil && *req.Order < 0 {
		return "order must be 0 or greater"
	}
	if req.Icon != nil && len(*req.Icon) > 50 {
		return "icon must be 50 characters or fewer"
	}
	return ""
}
