package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tv/internal/middleware"
)

func TestRecover_PanicReturns500(t *testing.T) {
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("deliberate test panic")
	})

	rr := httptest.NewRecorder()
	// The test itself must not panic — Recover must absorb it.
	middleware.Recover(panicking).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}

	body := rr.Body.Bytes()

	// Response must be valid JSON.
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("panic response is not valid JSON: %v — body: %s", err, body)
	}

	// Must contain the error envelope.
	if _, ok := resp["error"]; !ok {
		t.Error("response must have an 'error' key")
	}

	bodyStr := string(body)

	// Must NOT expose the panic value.
	if strings.Contains(bodyStr, "deliberate test panic") {
		t.Error("panic value must not appear in the HTTP response body")
	}

	// Must NOT expose a stack trace.
	if strings.Contains(bodyStr, "goroutine") {
		t.Error("stack trace must not appear in the HTTP response body")
	}
}

func TestRecover_NoPanic_PassesThrough(t *testing.T) {
	normal := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	rr := httptest.NewRecorder()
	middleware.Recover(normal).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rr.Code)
	}
}
