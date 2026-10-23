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

func newSettingsHandler() *handler.SettingsHandler {
	return handler.NewSettingsHandler(store.NewSettingsStore(), nil)
}

func mustDecodeSettings(t *testing.T, rr *httptest.ResponseRecorder) model.Settings {
	t.Helper()
	var resp struct {
		Data model.Settings `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode settings: %v — body: %s", err, rr.Body.String())
	}
	return resp.Data
}

// ── GET /api/v1/settings ──────────────────────────────────────────────────────

func TestSettingsHandler_Get_Returns200AndDefaults(t *testing.T) {
	h := newSettingsHandler()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	rr := httptest.NewRecorder()

	h.Get(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	st := mustDecodeSettings(t, rr)
	if st.Theme != "dark" || st.Language != "en" || st.AutoRefreshInterval != 30 {
		t.Errorf("unexpected default settings: %+v", st)
	}
}

// ── PUT /api/v1/settings ──────────────────────────────────────────────────────

func TestSettingsHandler_Update_Success(t *testing.T) {
	h := newSettingsHandler()

	newTheme := "light"
	newInterval := 60
	quality := "4K"
	maintenance := true

	req := model.UpdateSettingsRequest{
		Theme:               &newTheme,
		AutoRefreshInterval: &newInterval,
		MaxStreamQuality:    &quality,
		MaintenanceMode:     &maintenance,
	}

	r := httptest.NewRequest(http.MethodPut, "/api/v1/settings", toJSON(t, req))
	rr := httptest.NewRecorder()

	h.Update(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", rr.Code, rr.Body.String())
	}

	st := mustDecodeSettings(t, rr)
	if st.Theme != "light" || st.AutoRefreshInterval != 60 || st.MaxStreamQuality != "4K" || !st.MaintenanceMode {
		t.Errorf("unexpected updated settings: %+v", st)
	}
	if st.Language != "en" {
		t.Errorf("omitted language modified: %q", st.Language)
	}
}

func TestSettingsHandler_Update_ValidationFailures(t *testing.T) {
	h := newSettingsHandler()

	badTheme := "neon-pink"
	badLang := "x"
	tooShortInterval := 2
	tooLongInterval := 500
	badQuality := "8K"

	tests := []struct {
		name string
		req  model.UpdateSettingsRequest
	}{
		{name: "invalid theme", req: model.UpdateSettingsRequest{Theme: &badTheme}},
		{name: "too short language code", req: model.UpdateSettingsRequest{Language: &badLang}},
		{name: "interval below minimum", req: model.UpdateSettingsRequest{AutoRefreshInterval: &tooShortInterval}},
		{name: "interval above maximum", req: model.UpdateSettingsRequest{AutoRefreshInterval: &tooLongInterval}},
		{name: "invalid max stream quality", req: model.UpdateSettingsRequest{MaxStreamQuality: &badQuality}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPut, "/api/v1/settings", toJSON(t, tc.req))
			rr := httptest.NewRecorder()

			h.Update(rr, r)

			if rr.Code != http.StatusUnprocessableEntity {
				t.Errorf("expected 422, got %d — %s", rr.Code, rr.Body.String())
			}
			assertJSONErrorCode(t, rr, "VALIDATION_ERROR")
		})
	}
}

func TestSettingsHandler_Update_InvalidJSON_Returns400(t *testing.T) {
	h := newSettingsHandler()
	r := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBufferString("not valid json"))
	rr := httptest.NewRecorder()

	h.Update(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	assertJSONErrorCode(t, rr, "BAD_REQUEST")
}

// ── POST /api/v1/settings/reset ───────────────────────────────────────────────

func TestSettingsHandler_Reset_Success(t *testing.T) {
	h := newSettingsHandler()

	// Mutate first
	newTheme := "system"
	h.Update(httptest.NewRecorder(), httptest.NewRequest(http.MethodPut, "/api/v1/settings", toJSON(t, model.UpdateSettingsRequest{Theme: &newTheme})))

	// Reset
	r := httptest.NewRequest(http.MethodPost, "/api/v1/settings/reset", nil)
	rr := httptest.NewRecorder()

	h.Reset(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	st := mustDecodeSettings(t, rr)
	if st.Theme != "dark" {
		t.Errorf("expected theme reset to 'dark', got %q", st.Theme)
	}
}

// ── WebSocket Broadcasts ─────────────────────────────────────────────────────

func TestSettingsHandler_BroadcastsEventsOnUpdateAndReset(t *testing.T) {
	s := store.NewSettingsStore()
	spy := &spyBroadcaster{}
	h := handler.NewSettingsHandler(s, spy)

	// 1. Update -> expect settings.updated
	newTheme := "light"
	r := httptest.NewRequest(http.MethodPut, "/api/v1/settings", toJSON(t, model.UpdateSettingsRequest{Theme: &newTheme}))
	rr := httptest.NewRecorder()
	h.Update(rr, r)

	if len(spy.events) != 1 {
		t.Fatalf("expected 1 event after update, got %d", len(spy.events))
	}
	if spy.events[0].Type != ws.EventSettingsUpdated || spy.events[0].Resource != "settings" || spy.events[0].Operation != ws.OpUpdate {
		t.Errorf("unexpected update event: %+v", spy.events[0])
	}

	// 2. Reset -> expect settings.updated
	resetR := httptest.NewRequest(http.MethodPost, "/api/v1/settings/reset", nil)
	resetRR := httptest.NewRecorder()
	h.Reset(resetRR, resetR)

	if len(spy.events) != 2 {
		t.Fatalf("expected 2 events after reset, got %d", len(spy.events))
	}
	if spy.events[1].Type != ws.EventSettingsUpdated || spy.events[1].Resource != "settings" || spy.events[1].Operation != ws.OpUpdate {
		t.Errorf("unexpected reset event: %+v", spy.events[1])
	}
}
