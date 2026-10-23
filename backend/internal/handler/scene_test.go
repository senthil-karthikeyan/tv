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

func newSeededSceneHandler() *handler.SceneHandler {
	return handler.NewSceneHandler(store.NewSceneStore(), nil)
}

func newEmptySceneHandler() *handler.SceneHandler {
	return handler.NewSceneHandler(store.NewEmptySceneStore(), nil)
}

// toJSON marshals v to a *bytes.Buffer or fails the test.
func toJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return bytes.NewBuffer(b)
}

// mustDecodeScene decodes a {"data": Scene} envelope and returns the inner Scene.
func mustDecodeScene(t *testing.T, rr *httptest.ResponseRecorder) model.Scene {
	t.Helper()
	var resp struct {
		Data model.Scene `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode scene: %v — body: %s", err, rr.Body.String())
	}
	return resp.Data
}

// createSceneViaHandler POSTs req to the handler and returns the created scene.
func createSceneViaHandler(t *testing.T, h *handler.SceneHandler, req model.CreateSceneRequest) model.Scene {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/scenes", toJSON(t, req))
	rr := httptest.NewRecorder()
	h.Create(rr, r)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Create: expected 201, got %d — %s", rr.Code, rr.Body.String())
	}
	return mustDecodeScene(t, rr)
}

func validSceneReq() model.CreateSceneRequest {
	return model.CreateSceneRequest{
		Name:    "Test Scene",
		Label:   "test",
		Icon:    "star",
		Route:   "/test",
		Order:   10,
		Visible: true,
	}
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestSceneHandler_List_Returns200AndSeeds(t *testing.T) {
	h := newSeededSceneHandler()
	rr := httptest.NewRecorder()
	h.List(rr, httptest.NewRequest(http.MethodGet, "/api/v1/scenes", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp struct {
		Data []model.Scene `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Meta.Total != 3 {
		t.Errorf("expected 3 seed scenes, got %d", resp.Meta.Total)
	}
	if len(resp.Data) != 3 {
		t.Errorf("expected 3 items in data array, got %d", len(resp.Data))
	}
}

func TestSceneHandler_List_SortedByOrder(t *testing.T) {
	h := newSeededSceneHandler()
	rr := httptest.NewRecorder()
	h.List(rr, httptest.NewRequest(http.MethodGet, "/api/v1/scenes", nil))

	var resp struct {
		Data []model.Scene `json:"data"`
	}
	json.NewDecoder(rr.Body).Decode(&resp)

	for i := 1; i < len(resp.Data); i++ {
		if resp.Data[i].Order < resp.Data[i-1].Order {
			t.Errorf("not sorted at index %d: order %d < %d", i, resp.Data[i].Order, resp.Data[i-1].Order)
		}
	}
}

func TestSceneHandler_List_ContentType(t *testing.T) {
	rr := httptest.NewRecorder()
	newSeededSceneHandler().List(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestSceneHandler_GetByID_Found(t *testing.T) {
	h := newEmptySceneHandler()
	created := createSceneViaHandler(t, h, validSceneReq())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenes/"+created.ID, nil)
	req.SetPathValue("id", created.ID)
	rr := httptest.NewRecorder()
	h.GetByID(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	sc := mustDecodeScene(t, rr)
	if sc.ID != created.ID {
		t.Errorf("ID mismatch: want %s, got %s", created.ID, sc.ID)
	}
}

func TestSceneHandler_GetByID_NotFound(t *testing.T) {
	h := newEmptySceneHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenes/missing", nil)
	req.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()
	h.GetByID(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
	assertJSONErrorCode(t, rr, "NOT_FOUND")
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestSceneHandler_Create_Returns201WithScene(t *testing.T) {
	h := newEmptySceneHandler()
	sc := createSceneViaHandler(t, h, validSceneReq())

	if sc.ID == "" {
		t.Error("expected non-empty ID")
	}
	if sc.Name != validSceneReq().Name {
		t.Errorf("expected name %q, got %q", validSceneReq().Name, sc.Name)
	}
	if sc.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be populated")
	}
}

func TestSceneHandler_Create_MissingName_Returns422(t *testing.T) {
	h := newEmptySceneHandler()
	req := validSceneReq()
	req.Name = ""

	r := httptest.NewRequest(http.MethodPost, "/api/v1/scenes", toJSON(t, req))
	rr := httptest.NewRecorder()
	h.Create(rr, r)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

func TestSceneHandler_Create_MissingLabel_Returns422(t *testing.T) {
	h := newEmptySceneHandler()
	req := validSceneReq()
	req.Label = ""

	r := httptest.NewRequest(http.MethodPost, "/api/v1/scenes", toJSON(t, req))
	rr := httptest.NewRecorder()
	h.Create(rr, r)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

func TestSceneHandler_Create_RouteWithoutLeadingSlash_Returns422(t *testing.T) {
	h := newEmptySceneHandler()
	req := validSceneReq()
	req.Route = "no-leading-slash"

	r := httptest.NewRequest(http.MethodPost, "/api/v1/scenes", toJSON(t, req))
	rr := httptest.NewRecorder()
	h.Create(rr, r)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

func TestSceneHandler_Create_NegativeOrder_Returns422(t *testing.T) {
	h := newEmptySceneHandler()
	req := validSceneReq()
	req.Order = -1

	r := httptest.NewRequest(http.MethodPost, "/api/v1/scenes", toJSON(t, req))
	rr := httptest.NewRecorder()
	h.Create(rr, r)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

func TestSceneHandler_Create_InvalidJSON_Returns400(t *testing.T) {
	h := newEmptySceneHandler()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/scenes", bytes.NewBufferString("{bad json"))
	rr := httptest.NewRecorder()
	h.Create(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestSceneHandler_Update_UpdatesNameOnly(t *testing.T) {
	h := newEmptySceneHandler()
	created := createSceneViaHandler(t, h, validSceneReq())

	newName := "Renamed Scene"
	updateReq := model.UpdateSceneRequest{Name: &newName}
	r := httptest.NewRequest(http.MethodPut, "/api/v1/scenes/"+created.ID, toJSON(t, updateReq))
	r.SetPathValue("id", created.ID)
	rr := httptest.NewRecorder()
	h.Update(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", rr.Code, rr.Body.String())
	}
	updated := mustDecodeScene(t, rr)
	if updated.Name != newName {
		t.Errorf("expected name %q, got %q", newName, updated.Name)
	}
	// Other fields must be unchanged.
	if updated.Route != created.Route {
		t.Errorf("route should not have changed: got %q", updated.Route)
	}
}

func TestSceneHandler_Update_NotFound(t *testing.T) {
	h := newEmptySceneHandler()
	newName := "X"
	r := httptest.NewRequest(http.MethodPut, "/api/v1/scenes/missing", toJSON(t, model.UpdateSceneRequest{Name: &newName}))
	r.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()
	h.Update(rr, r)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestSceneHandler_Update_InvalidJSON_Returns400(t *testing.T) {
	h := newEmptySceneHandler()
	created := createSceneViaHandler(t, h, validSceneReq())

	r := httptest.NewRequest(http.MethodPut, "/api/v1/scenes/"+created.ID, bytes.NewBufferString("notjson"))
	r.SetPathValue("id", created.ID)
	rr := httptest.NewRecorder()
	h.Update(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestSceneHandler_Update_InvalidRoute_Returns422(t *testing.T) {
	h := newEmptySceneHandler()
	created := createSceneViaHandler(t, h, validSceneReq())

	badRoute := "no-slash"
	r := httptest.NewRequest(http.MethodPut, "/api/v1/scenes/"+created.ID, toJSON(t, model.UpdateSceneRequest{Route: &badRoute}))
	r.SetPathValue("id", created.ID)
	rr := httptest.NewRecorder()
	h.Update(rr, r)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestSceneHandler_Delete_Returns204(t *testing.T) {
	h := newEmptySceneHandler()
	created := createSceneViaHandler(t, h, validSceneReq())

	r := httptest.NewRequest(http.MethodDelete, "/api/v1/scenes/"+created.ID, nil)
	r.SetPathValue("id", created.ID)
	rr := httptest.NewRecorder()
	h.Delete(rr, r)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestSceneHandler_Delete_SceneIsGoneAfterDelete(t *testing.T) {
	h := newEmptySceneHandler()
	created := createSceneViaHandler(t, h, validSceneReq())

	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/scenes/"+created.ID, nil)
	delReq.SetPathValue("id", created.ID)
	h.Delete(httptest.NewRecorder(), delReq)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/scenes/"+created.ID, nil)
	getReq.SetPathValue("id", created.ID)
	rr := httptest.NewRecorder()
	h.GetByID(rr, getReq)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", rr.Code)
	}
}

func TestSceneHandler_Delete_NotFound(t *testing.T) {
	h := newEmptySceneHandler()
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/scenes/missing", nil)
	r.SetPathValue("id", "missing")
	rr := httptest.NewRecorder()
	h.Delete(rr, r)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ── helper ────────────────────────────────────────────────────────────────────

func assertJSONErrorCode(t *testing.T, rr *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if resp.Error.Code != wantCode {
		t.Errorf("expected error code %q, got %q", wantCode, resp.Error.Code)
	}
}

// ── Broadcaster spy ───────────────────────────────────────────────────────────

type spyBroadcaster struct {
	events []ws.Event
}

func (s *spyBroadcaster) Broadcast(event ws.Event) {
	s.events = append(s.events, event)
}

func TestSceneHandler_BroadcastsEventsOnMutations(t *testing.T) {
	s := store.NewEmptySceneStore()
	spy := &spyBroadcaster{}
	h := handler.NewSceneHandler(s, spy)

	// 1. Create -> expect scene.created
	req := validSceneReq()
	created := createSceneViaHandler(t, h, req)

	if len(spy.events) != 1 {
		t.Fatalf("expected 1 event after create, got %d", len(spy.events))
	}
	if spy.events[0].Type != ws.EventSceneCreated || spy.events[0].ResourceID != created.ID || spy.events[0].Operation != ws.OpCreate {
		t.Errorf("unexpected create event: %+v", spy.events[0])
	}

	// 2. Update -> expect scene.updated
	newName := "Updated Name"
	updateReq := model.UpdateSceneRequest{Name: &newName}
	updateR := httptest.NewRequest(http.MethodPut, "/api/v1/scenes/"+created.ID, toJSON(t, updateReq))
	updateR.SetPathValue("id", created.ID)
	updateRR := httptest.NewRecorder()
	h.Update(updateRR, updateR)
	if updateRR.Code != http.StatusOK {
		t.Fatalf("Update: expected 200, got %d", updateRR.Code)
	}

	if len(spy.events) != 2 {
		t.Fatalf("expected 2 events after update, got %d", len(spy.events))
	}
	if spy.events[1].Type != ws.EventSceneUpdated || spy.events[1].ResourceID != created.ID || spy.events[1].Operation != ws.OpUpdate {
		t.Errorf("unexpected update event: %+v", spy.events[1])
	}

	// 3. Delete -> expect scene.deleted
	delR := httptest.NewRequest(http.MethodDelete, "/api/v1/scenes/"+created.ID, nil)
	delR.SetPathValue("id", created.ID)
	delRR := httptest.NewRecorder()
	h.Delete(delRR, delR)
	if delRR.Code != http.StatusNoContent {
		t.Fatalf("Delete: expected 204, got %d", delRR.Code)
	}

	if len(spy.events) != 3 {
		t.Fatalf("expected 3 events after delete, got %d", len(spy.events))
	}
	if spy.events[2].Type != ws.EventSceneDeleted || spy.events[2].ResourceID != created.ID || spy.events[2].Operation != ws.OpDelete {
		t.Errorf("unexpected delete event: %+v", spy.events[2])
	}
}
