package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tv/internal/handler"
)

func TestWriteData_StatusAndEnvelope(t *testing.T) {
	rr := httptest.NewRecorder()
	handler.WriteData(rr, http.StatusOK, map[string]string{"key": "value"})

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json Content-Type, got %q", ct)
	}

	var resp struct {
		Data map[string]string `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Data["key"] != "value" {
		t.Errorf("unexpected data: %v", resp.Data)
	}
}

func TestWriteData_CustomStatus(t *testing.T) {
	rr := httptest.NewRecorder()
	handler.WriteData(rr, http.StatusCreated, struct{ ID string }{ID: "abc"})

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rr.Code)
	}
}

func TestWriteCollection_EnvelopeAndMeta(t *testing.T) {
	rr := httptest.NewRecorder()
	handler.WriteCollection(rr, []string{"a", "b"}, 2)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Data []string `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Data))
	}
	if resp.Meta.Total != 2 {
		t.Errorf("expected meta.total=2, got %d", resp.Meta.Total)
	}
}

func TestWriteError_CodeAndMessage(t *testing.T) {
	rr := httptest.NewRecorder()
	handler.WriteError(rr, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}

	var resp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %q", resp.Error.Code)
	}
	if resp.Error.Message != "name is required" {
		t.Errorf("unexpected message: %q", resp.Error.Message)
	}
}

func TestWriteNotFound(t *testing.T) {
	rr := httptest.NewRecorder()
	handler.WriteNotFound(rr, "scene")

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestWriteMethodNotAllowed(t *testing.T) {
	rr := httptest.NewRecorder()
	handler.WriteMethodNotAllowed(rr)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestWriteInternalError_HidesDetails(t *testing.T) {
	rr := httptest.NewRecorder()
	handler.WriteInternalError(rr)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}

	body := rr.Body.Bytes()
	var resp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("expected INTERNAL_ERROR code, got %q", resp.Error.Code)
	}
	// Ensure no stack-trace or implementation detail keys are present.
	var raw map[string]any
	_ = json.Unmarshal(body, &raw)
	if _, has := raw["stack"]; has {
		t.Error("stack trace must not be present in the response")
	}
}

func TestWriteValidationError(t *testing.T) {
	rr := httptest.NewRecorder()
	handler.WriteValidationError(rr, "field X is required")

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
}
