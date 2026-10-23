package middleware

import (
	"crypto/subtle"
	"net/http"

	"tv/internal/config"
	"tv/internal/handler"
)

// APIKey returns a middleware that enforces the presence and correctness of the
// X-API-Key header using a constant-time comparison to prevent timing attacks.
//
// Apply this middleware only to routes that require authentication.
// The /health route and WebSocket upgrade endpoints may bypass it where appropriate.
func APIKey(cfg *config.Config) func(http.Handler) http.Handler {
	expected := []byte(cfg.APIKey)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				handler.WriteError(w, http.StatusUnauthorized, "MISSING_API_KEY", "X-API-Key header is required")
				return
			}

			// ConstantTimeCompare guards against timing side-channel attacks.
			// It also returns 0 when the lengths differ, so length mismatches are safe.
			if subtle.ConstantTimeCompare([]byte(key), expected) != 1 {
				handler.WriteError(w, http.StatusUnauthorized, "INVALID_API_KEY", "invalid API key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
