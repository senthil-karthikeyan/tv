package handler

import (
	"net/http"
)

// healthPayload is the response body of GET /health.
type healthPayload struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Health handles GET /health.
// It requires no authentication and is intended for load-balancer and readiness probes.
// Non-GET methods receive a 405 response for defence-in-depth even when the mux
// restricts routing to GET.
func Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteMethodNotAllowed(w)
		return
	}
	WriteData(w, http.StatusOK, healthPayload{
		Status:  "ok",
		Version: "1",
	})
}
