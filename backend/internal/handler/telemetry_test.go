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

func newTelemetryHandler() (*handler.TelemetryHandler, *store.TelemetryStore, *store.SettingsStore) {
	ts := store.NewTelemetryStore()
	ss := store.NewSettingsStore()
	h := handler.NewTelemetryHandler(ts, ss, nil)
	return h, ts, ss
}

// ── POST /api/v1/telemetry/events ─────────────────────────────────────────────

func TestTelemetryHandler_Ingest_SingleEvent(t *testing.T) {
	h, ts, _ := newTelemetryHandler()

	event := model.IngestEventRequest{
		Type:   "video_start",
		Source: "player",
		Payload: map[string]any{
			"bitrate": 5000,
		},
	}

	r := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/events", toJSON(t, event))
	rr := httptest.NewRecorder()

	h.Ingest(rr, r)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d — %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data struct {
			Accepted int `json:"accepted"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Data.Accepted != 1 {
		t.Errorf("expected accepted=1, got %d", resp.Data.Accepted)
	}

	events := ts.List("", "", 10)
	if len(events) != 1 {
		t.Fatalf("expected 1 stored event, got %d", len(events))
	}
	if events[0].Type != "video_start" {
		t.Errorf("expected stored event type 'video_start', got %q", events[0].Type)
	}
}

func TestTelemetryHandler_Ingest_BatchEvents(t *testing.T) {
	h, ts, _ := newTelemetryHandler()

	batch := []model.IngestEventRequest{
		{Type: "page_view", Source: "launcher"},
		{Type: "scene_selected", Source: "launcher"},
	}

	r := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/events", toJSON(t, batch))
	rr := httptest.NewRecorder()

	h.Ingest(rr, r)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d — %s", rr.Code, rr.Body.String())
	}

	events := ts.List("", "", 10)
	if len(events) != 2 {
		t.Fatalf("expected 2 stored events, got %d", len(events))
	}
}

func TestTelemetryHandler_Ingest_ValidationFailures(t *testing.T) {
	h, _, _ := newTelemetryHandler()

	tests := []struct {
		name string
		body any
	}{
		{
			name: "missing type",
			body: model.IngestEventRequest{Source: "player"},
		},
		{
			name: "missing source",
			body: model.IngestEventRequest{Type: "video_start"},
		},
		{
			name: "batch with invalid item",
			body: []model.IngestEventRequest{
				{Type: "valid", Source: "player"},
				{Type: "", Source: "player"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/events", toJSON(t, tc.body))
			rr := httptest.NewRecorder()

			h.Ingest(rr, r)

			if rr.Code != http.StatusUnprocessableEntity {
				t.Errorf("expected 422, got %d — %s", rr.Code, rr.Body.String())
			}
			assertJSONErrorCode(t, rr, "VALIDATION_ERROR")
		})
	}
}

func TestTelemetryHandler_Ingest_DisabledInSettings(t *testing.T) {
	h, ts, ss := newTelemetryHandler()

	// Disable telemetry in application settings
	disabled := false
	ss.Update(&model.UpdateSettingsRequest{
		TelemetryEnabled: &disabled,
	})

	r := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/events", toJSON(t, model.IngestEventRequest{
		Type:   "video_start",
		Source: "player",
	}))
	rr := httptest.NewRecorder()

	h.Ingest(rr, r)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted, got %d", rr.Code)
	}

	var resp struct {
		Data struct {
			Accepted int    `json:"accepted"`
			Status   string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Data.Accepted != 0 || resp.Data.Status != "disabled" {
		t.Errorf("expected accepted=0, status=disabled, got %+v", resp.Data)
	}

	// Verify nothing was recorded in store
	if len(ts.List("", "", 10)) != 0 {
		t.Error("expected 0 events in store when telemetry is disabled")
	}
}

func TestTelemetryHandler_Ingest_InvalidJSON(t *testing.T) {
	h, _, _ := newTelemetryHandler()

	r := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/events", bytes.NewBufferString("not valid json"))
	rr := httptest.NewRecorder()

	h.Ingest(rr, r)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	assertJSONErrorCode(t, rr, "BAD_REQUEST")
}

// ── GET /api/v1/telemetry/events ─────────────────────────────────────────────

func TestTelemetryHandler_List_Filtered(t *testing.T) {
	h, ts, _ := newTelemetryHandler()

	ts.Record(&model.IngestEventRequest{Type: "click", Source: "launcher"})
	ts.Record(&model.IngestEventRequest{Type: "video_start", Source: "player"})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/events?type=click", nil)
	rr := httptest.NewRecorder()

	h.List(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Data []model.TelemetryEvent `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Meta.Total != 1 || len(resp.Data) != 1 || resp.Data[0].Type != "click" {
		t.Errorf("unexpected list response: %+v", resp)
	}
}

// ── GET /api/v1/telemetry/summary ────────────────────────────────────────────

func TestTelemetryHandler_Summary(t *testing.T) {
	h, ts, _ := newTelemetryHandler()

	ts.Record(&model.IngestEventRequest{Type: "click", Source: "launcher"})
	ts.Record(&model.IngestEventRequest{Type: "click", Source: "launcher"})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/summary", nil)
	rr := httptest.NewRecorder()

	h.Summary(rr, r)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Data model.TelemetrySummary `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Data.TotalRecorded != 2 || resp.Data.ByType["click"] != 2 {
		t.Errorf("unexpected summary response: %+v", resp.Data)
	}
}

// ── WebSocket Broadcasts ─────────────────────────────────────────────────────

func TestTelemetryHandler_BroadcastsEvents(t *testing.T) {
	ts := store.NewTelemetryStore()
	ss := store.NewSettingsStore()
	spy := &spyBroadcaster{}
	h := handler.NewTelemetryHandler(ts, ss, spy)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/events", toJSON(t, model.IngestEventRequest{
		Type:   "video_start",
		Source: "player",
	}))
	rr := httptest.NewRecorder()

	h.Ingest(rr, r)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rr.Code)
	}

	if len(spy.events) != 1 {
		t.Fatalf("expected 1 broadcast event, got %d", len(spy.events))
	}
	if spy.events[0].Type != ws.EventTelemetryRecorded || spy.events[0].Resource != "telemetry" {
		t.Errorf("unexpected broadcast event: %+v", spy.events[0])
	}
}
