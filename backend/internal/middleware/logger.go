package middleware

import (
	"log"
	"net/http"
	"time"
)

// statusWriter wraps http.ResponseWriter to capture the HTTP status code written
// by the downstream handler so it can be included in the request log line.
type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (sw *statusWriter) WriteHeader(code int) {
	if sw.wrote {
		return // prevent duplicate WriteHeader calls
	}
	sw.status = code
	sw.wrote = true
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Write(b []byte) (int, error) {
	if !sw.wrote {
		// net/http defaults to 200 when Write is called without WriteHeader.
		sw.WriteHeader(http.StatusOK)
	}
	return sw.ResponseWriter.Write(b)
}

// Logger logs each request with its method, path, response status, and elapsed time.
// The log line is emitted after the handler returns so the final status is known.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Microsecond))
	})
}
