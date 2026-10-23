package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"tv/internal/model"
	"tv/internal/store"
	"tv/internal/ws"
)

// UserHandler handles HTTP requests for the /api/v1/users resource.
type UserHandler struct {
	store       *store.UserStore
	broadcaster ws.Broadcaster
}

// NewUserHandler creates a UserHandler backed by the provided store and broadcaster.
// If broadcaster is nil, real-time broadcasts are skipped.
func NewUserHandler(s *store.UserStore, b ws.Broadcaster) *UserHandler {
	return &UserHandler{
		store:       s,
		broadcaster: b,
	}
}

// List handles GET /api/v1/users.
// Returns all users ordered by CreatedAt ascending wrapped in the collection envelope.
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	users := h.store.List()
	WriteCollection(w, users, len(users))
}

// GetByID handles GET /api/v1/users/{id}.
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	u, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			WriteNotFound(w, "user")
			return
		}
		WriteInternalError(w)
		return
	}
	WriteData(w, http.StatusOK, u)
}

// Create handles POST /api/v1/users.
// Requires a valid JSON body and API key authentication.
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "invalid JSON body")
		return
	}

	if msg := validateCreateUser(&req); msg != "" {
		WriteValidationError(w, msg)
		return
	}

	u, err := h.store.Create(&req)
	if err != nil {
		if errors.Is(err, store.ErrDuplicateEmail) {
			WriteConflict(w, "DUPLICATE_EMAIL", "email already in use")
			return
		}
		WriteInternalError(w)
		return
	}

	if h.broadcaster != nil {
		h.broadcaster.Broadcast(ws.Event{
			Type:       ws.EventUserCreated,
			Resource:   "user",
			ResourceID: u.ID,
			Operation:  ws.OpCreate,
			Data:       u,
		})
	}

	WriteData(w, http.StatusCreated, u)
}

// Update handles PUT /api/v1/users/{id}.
// Only non-null fields in the request body are applied; omitted fields are unchanged.
// Requires API key authentication.
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req model.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "invalid JSON body")
		return
	}

	if msg := validateUpdateUser(&req); msg != "" {
		WriteValidationError(w, msg)
		return
	}

	u, err := h.store.Update(id, &req)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			WriteNotFound(w, "user")
			return
		}
		if errors.Is(err, store.ErrDuplicateEmail) {
			WriteConflict(w, "DUPLICATE_EMAIL", "email already in use")
			return
		}
		WriteInternalError(w)
		return
	}

	if h.broadcaster != nil {
		h.broadcaster.Broadcast(ws.Event{
			Type:       ws.EventUserUpdated,
			Resource:   "user",
			ResourceID: u.ID,
			Operation:  ws.OpUpdate,
			Data:       u,
		})
	}

	WriteData(w, http.StatusOK, u)
}

// Delete handles DELETE /api/v1/users/{id}.
// Returns 204 No Content on success.
// Requires API key authentication.
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.Delete(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			WriteNotFound(w, "user")
			return
		}
		WriteInternalError(w)
		return
	}

	if h.broadcaster != nil {
		h.broadcaster.Broadcast(ws.Event{
			Type:       ws.EventUserDeleted,
			Resource:   "user",
			ResourceID: id,
			Operation:  ws.OpDelete,
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── validation ────────────────────────────────────────────────────────────────

func validateCreateUser(req *model.CreateUserRequest) string {
	if strings.TrimSpace(req.Name) == "" {
		return "name is required"
	}
	if len(req.Name) > 100 {
		return "name must be 100 characters or fewer"
	}
	if strings.TrimSpace(req.Email) == "" {
		return "email is required"
	}
	if len(req.Email) > 255 {
		return "email must be 255 characters or fewer"
	}
	if !isValidEmail(req.Email) {
		return "email must be a valid email address"
	}
	if req.Role != "" && !isValidRole(req.Role) {
		return "role must be one of: admin, editor, viewer"
	}
	if req.Status != "" && !isValidStatus(req.Status) {
		return "status must be one of: active, inactive, suspended"
	}
	if len(req.Avatar) > 255 {
		return "avatar must be 255 characters or fewer"
	}
	return ""
}

func validateUpdateUser(req *model.UpdateUserRequest) string {
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return "name must not be empty"
		}
		if len(*req.Name) > 100 {
			return "name must be 100 characters or fewer"
		}
	}
	if req.Email != nil {
		if strings.TrimSpace(*req.Email) == "" {
			return "email must not be empty"
		}
		if len(*req.Email) > 255 {
			return "email must be 255 characters or fewer"
		}
		if !isValidEmail(*req.Email) {
			return "email must be a valid email address"
		}
	}
	if req.Role != nil && !isValidRole(*req.Role) {
		return "role must be one of: admin, editor, viewer"
	}
	if req.Status != nil && !isValidStatus(*req.Status) {
		return "status must be one of: active, inactive, suspended"
	}
	if req.Avatar != nil && len(*req.Avatar) > 255 {
		return "avatar must be 255 characters or fewer"
	}
	return ""
}

func isValidRole(r string) bool {
	switch r {
	case "admin", "editor", "viewer":
		return true
	default:
		return false
	}
}

func isValidStatus(s string) bool {
	switch s {
	case "active", "inactive", "suspended":
		return true
	default:
		return false
	}
}

func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if len(email) < 3 || len(email) > 255 {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	return strings.Contains(parts[1], ".")
}
