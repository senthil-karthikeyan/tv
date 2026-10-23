package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tv/internal/config"
	"tv/internal/middleware"
)

func apikeyTestCfg() *config.Config {
	return &config.Config{APIKey: "correct-secret-key"}
}

func TestAPIKey_MissingHeader_Returns401(t *testing.T) {
	h := middleware.APIKey(apikeyTestCfg())(okHandler())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No X-API-Key header set.
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	assertJSONError(t, rr, "MISSING_API_KEY")
}

func TestAPIKey_WrongKey_Returns401(t *testing.T) {
	h := middleware.APIKey(apikeyTestCfg())(okHandler())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	assertJSONError(t, rr, "INVALID_API_KEY")
}

func TestAPIKey_CorrectKey_PassesThrough(t *testing.T) {
	h := middleware.APIKey(apikeyTestCfg())(okHandler())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "correct-secret-key")
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAPIKey_EmptyKey_Returns401(t *testing.T) {
	h := middleware.APIKey(apikeyTestCfg())(okHandler())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "")
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for empty header value, got %d", rr.Code)
	}
}

// okHandler returns a handler that always writes 200.
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// assertJSONError checks that the response body contains the expected error code.
func assertJSONError(t *testing.T, rr *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if resp.Error.Code != wantCode {
		t.Errorf("expected error code %q, got %q", wantCode, resp.Error.Code)
	}
}
