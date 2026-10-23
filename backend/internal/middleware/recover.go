package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"tv/internal/handler"
)

// Recover catches panics from downstream handlers, logs the full stack trace internally,
// and returns a safe 500 JSON response without exposing any internal details to the client.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				// Log the full stack trace for internal diagnostics only.
				log.Printf("PANIC recovered [%s %s]: %v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
				handler.WriteInternalError(w)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
