package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"tv/internal/middleware"
)

func TestLogger_PropagatesStatus(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	rr := httptest.NewRecorder()
	middleware.Logger(h).ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/test", nil))

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201 from downstream, got %d", rr.Code)
	}
}

func TestLogger_DefaultsTo200(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	rr := httptest.NewRecorder()
	middleware.Logger(h).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/test", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for implicit WriteHeader, got %d", rr.Code)
	}
}

func TestLogger_DoesNotAlterBody(t *testing.T) {
	const want = "hello"
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(want))
	})

	rr := httptest.NewRecorder()
	middleware.Logger(h).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rr.Body.String(); got != want {
		t.Errorf("expected body %q, got %q", want, got)
	}
}
