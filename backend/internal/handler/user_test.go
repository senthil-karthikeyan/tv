package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tv/internal/handler"
	"tv/internal/model"
	"tv/internal/store"
	"tv/internal/ws"
)

// ── test helpers ──────────────────────────────────────────────────────────────

func newSeededUserHandler() *handler.UserHandler {
	return handler.NewUserHandler(store.NewUserStore(), nil)
}

func newEmptyUserHandler() *handler.UserHandler {
	return handler.NewUserHandler(store.NewEmptyUserStore(), nil)
}

func mustDecodeUser(t *testing.T, rr *httptest.ResponseRecorder) model.User {
	t.Helper()
	var resp struct {
		Data model.User `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode user: %v — body: %s", err, rr.Body.String())
	}
	return resp.Data
}

func createUserViaHandler(t *testing.T, h *handler.UserHandler, req model.CreateUserRequest) model.User {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/users", toJSON(t, req))
	rr := httptest.NewRecorder()
	h.Create(rr, r)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Create: expected 201, got %d — %s", rr.Code, rr.Body.String())
	}
	return mustDecodeUser(t, rr)
}

func validUserReq() model.CreateUserRequest {
	return model.CreateUserRequest{
		Name:   "Alex Smith",
		Email:  "alex.smith@example.com",
		Role:   "viewer",
		Status: "active",
		Avatar: "avatar-alex",
	}
}

// ── GET /api/v1/users ─────────────────────────────────────────────────────────

func TestUserHandler_List_Returns200AndSeeds(t *testing.T) {
	h := newSeededUserHandler()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()

	h.List(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Data []model.User `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Meta.Total != 3 {
		t.Errorf("expected meta.total=3, got %d", resp.Meta.Total)
	}
	if len(resp.Data) != 3 {
		t.Errorf("expected 3 users, got %d", len(resp.Data))
	}
}

// ── GET /api/v1/users/{id} ───────────────────────────────────────────────────

func TestUserHandler_GetByID_Found(t *testing.T) {
	h := newEmptyUserHandler()
	created := createUserViaHandler(t, h, validUserReq())

	r := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+created.ID, nil)
	r.SetPathValue("id", created.ID)
	rr := httptest.NewRecorder()

	h.GetByID(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	got := mustDecodeUser(t, rr)
	if got.ID != created.ID || got.Email != created.Email {
		t.Errorf("retrieved user mismatch: got %+v, want %+v", got, created)
	}
}

func TestUserHandler_GetByID_NotFound(t *testing.T) {
	h := newEmptyUserHandler()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/users/nonexistent", nil)
	r.SetPathValue("id", "nonexistent")
	rr := httptest.NewRecorder()

	h.GetByID(rr, r)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	assertJSONErrorCode(t, rr, "NOT_FOUND")
}

// ── POST /api/v1/users ────────────────────────────────────────────────────────

func TestUserHandler_Create_Success(t *testing.T) {
	h := newEmptyUserHandler()
	req := validUserReq()

	r := httptest.NewRequest(http.MethodPost, "/api/v1/users", toJSON(t, req))
	rr := httptest.NewRecorder()

	h.Create(rr, r)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — %s", rr.Code, rr.Body.String())
	}

	created := mustDecodeUser(t, rr)
	if created.Name != req.Name {
		t.Errorf("name mismatch: got %q, want %q", created.Name, req.Name)
	}
	if created.Email != "alex.smith@example.com" {
		t.Errorf("email mismatch: got %q, want alex.smith@example.com", created.Email)
	}
	if created.Role != "viewer" {
		t.Errorf("expected default role 'viewer', got %q", created.Role)
	}
	if created.Status != "active" {
		t.Errorf("expected default status 'active', got %q", created.Status)
	}
}

func TestUserHandler_Create_ValidationFailures(t *testing.T) {
	h := newEmptyUserHandler()

	tests := []struct {
		name string
		req  model.CreateUserRequest
		want string
	}{
		{
			name: "missing name",
			req:  model.CreateUserRequest{Email: "test@example.com"},
			want: "VALIDATION_ERROR",
		},
		{
			name: "missing email",
			req:  model.CreateUserRequest{Name: "Test"},
			want: "VALIDATION_ERROR",
		},
		{
			name: "invalid email format",
			req:  model.CreateUserRequest{Name: "Test", Email: "not-an-email"},
			want: "VALIDATION_ERROR",
		},
		{
			name: "invalid role",
			req:  model.CreateUserRequest{Name: "Test", Email: "test@example.com", Role: "superhero"},
			want: "VALIDATION_ERROR",
		},
		{
			name: "invalid status",
			req:  model.CreateUserRequest{Name: "Test", Email: "test@example.com", Status: "banned"},
			want: "VALIDATION_ERROR",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/v1/users", toJSON(t, tc.req))
			rr := httptest.NewRecorder()
			h.Create(rr, r)

			if rr.Code != http.StatusUnprocessableEntity {
				t.Errorf("expected 422, got %d — %s", rr.Code, rr.Body.String())
			}
			assertJSONErrorCode(t, rr, tc.want)
		})
	}
}

func TestUserHandler_Create_DuplicateEmail_Returns409(t *testing.T) {
	h := newEmptyUserHandler()
	req := validUserReq()
	_ = createUserViaHandler(t, h, req)

	// Attempt duplicate create
	r := httptest.NewRequest(http.MethodPost, "/api/v1/users", toJSON(t, req))
	rr := httptest.NewRecorder()
	h.Create(rr, r)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d — %s", rr.Code, rr.Body.String())
	}
	assertJSONErrorCode(t, rr, "DUPLICATE_EMAIL")
}

func TestUserHandler_Create_InvalidJSON_Returns400(t *testing.T) {
	h := newEmptyUserHandler()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	h.Create(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	assertJSONErrorCode(t, rr, "BAD_REQUEST")
}

// ── PUT /api/v1/users/{id} ───────────────────────────────────────────────────

func TestUserHandler_Update_Success(t *testing.T) {
	h := newEmptyUserHandler()
	created := createUserViaHandler(t, h, validUserReq())

	newName := "Updated Name"
	newRole := "admin"
	req := model.UpdateUserRequest{
		Name: &newName,
		Role: &newRole,
	}

	r := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+created.ID, toJSON(t, req))
	r.SetPathValue("id", created.ID)
	rr := httptest.NewRecorder()

	h.Update(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", rr.Code, rr.Body.String())
	}

	updated := mustDecodeUser(t, rr)
	if updated.Name != newName || updated.Role != newRole {
		t.Errorf("unexpected updated user: %+v", updated)
	}
	if updated.Email != created.Email {
		t.Errorf("email modified unexpectedly: %q", updated.Email)
	}
}

func TestUserHandler_Update_DuplicateEmail_Returns409(t *testing.T) {
	h := newEmptyUserHandler()
	_ = createUserViaHandler(t, h, model.CreateUserRequest{Name: "User 1", Email: "one@example.com"})
	u2 := createUserViaHandler(t, h, model.CreateUserRequest{Name: "User 2", Email: "two@example.com"})

	conflictEmail := "one@example.com"
	req := model.UpdateUserRequest{Email: &conflictEmail}

	r := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+u2.ID, toJSON(t, req))
	r.SetPathValue("id", u2.ID)
	rr := httptest.NewRecorder()

	h.Update(rr, r)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", rr.Code)
	}
	assertJSONErrorCode(t, rr, "DUPLICATE_EMAIL")
}

func TestUserHandler_Update_NotFound(t *testing.T) {
	h := newEmptyUserHandler()
	newName := "Doesn't Matter"
	req := model.UpdateUserRequest{Name: &newName}

	r := httptest.NewRequest(http.MethodPut, "/api/v1/users/missing", toJSON(t, req))
	r.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()

	h.Update(rr, r)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	assertJSONErrorCode(t, rr, "NOT_FOUND")
}

// ── DELETE /api/v1/users/{id} ────────────────────────────────────────────────

func TestUserHandler_Delete_Success(t *testing.T) {
	h := newEmptyUserHandler()
	created := createUserViaHandler(t, h, validUserReq())

	r := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+created.ID, nil)
	r.SetPathValue("id", created.ID)
	rr := httptest.NewRecorder()

	h.Delete(rr, r)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", rr.Code)
	}

	// Verify user is gone
	getR := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+created.ID, nil)
	getR.SetPathValue("id", created.ID)
	getRR := httptest.NewRecorder()
	h.GetByID(getRR, getR)

	if getRR.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", getRR.Code)
	}
}

func TestUserHandler_Delete_NotFound(t *testing.T) {
	h := newEmptyUserHandler()
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/users/missing", nil)
	r.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()

	h.Delete(rr, r)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	assertJSONErrorCode(t, rr, "NOT_FOUND")
}

// ── WebSocket Broadcasts ─────────────────────────────────────────────────────

func TestUserHandler_BroadcastsEventsOnMutations(t *testing.T) {
	s := store.NewEmptyUserStore()
	spy := &spyBroadcaster{}
	h := handler.NewUserHandler(s, spy)

	// 1. Create -> expect user.created
	created := createUserViaHandler(t, h, validUserReq())

	if len(spy.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(spy.events))
	}
	if spy.events[0].Type != ws.EventUserCreated || spy.events[0].Resource != "user" || spy.events[0].ResourceID != created.ID {
		t.Errorf("unexpected create event: %+v", spy.events[0])
	}

	// 2. Update -> expect user.updated
	newName := "New Name"
	updateReq := model.UpdateUserRequest{Name: &newName}
	updateR := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+created.ID, toJSON(t, updateReq))
	updateR.SetPathValue("id", created.ID)
	updateRR := httptest.NewRecorder()
	h.Update(updateRR, updateR)
	if updateRR.Code != http.StatusOK {
		t.Fatalf("Update: expected 200, got %d", updateRR.Code)
	}

	if len(spy.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(spy.events))
	}
	if spy.events[1].Type != ws.EventUserUpdated || spy.events[1].Resource != "user" || spy.events[1].ResourceID != created.ID {
		t.Errorf("unexpected update event: %+v", spy.events[1])
	}

	// 3. Delete -> expect user.deleted
	delR := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+created.ID, nil)
	delR.SetPathValue("id", created.ID)
	delRR := httptest.NewRecorder()
	h.Delete(delRR, delR)
	if delRR.Code != http.StatusNoContent {
		t.Fatalf("Delete: expected 204, got %d", delRR.Code)
	}

	if len(spy.events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(spy.events))
	}
	if spy.events[2].Type != ws.EventUserDeleted || spy.events[2].Resource != "user" || spy.events[2].ResourceID != created.ID {
		t.Errorf("unexpected delete event: %+v", spy.events[2])
	}
}
