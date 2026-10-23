// Package ws implements the WebSocket hub and connection handler.
package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Handler upgrades HTTP requests to WebSocket connections and registers them with the Hub.
type Handler struct {
	hub      *Hub
	upgrader websocket.Upgrader
}

// NewHandler creates a WebSocket HTTP handler backed by the given Hub.
// allowedOrigin specifies the expected Origin header value (e.g. from config.FrontendURL).
// If allowedOrigin is "*" or empty, origin checking is disabled.
func NewHandler(hub *Hub, allowedOrigin string) *Handler {
	return &Handler{
		hub: hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				if allowedOrigin == "" || allowedOrigin == "*" {
					return true
				}
				origin := r.Header.Get("Origin")
				if origin == "" {
					// Allow non-browser clients (tests, cURL, native apps) with no Origin header.
					return true
				}
				return origin == allowedOrigin
			},
		},
	}
}

// ServeHTTP handles the WebSocket handshake (GET /ws) and spawns client pumps on success.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: upgrade error from %s: %v", r.RemoteAddr, err)
		return
	}

	client := newClient(h.hub, conn)

	select {
	case h.hub.register <- client:
	case <-h.hub.quit:
		_ = conn.Close()
		return
	}

	go client.writePump()
	go client.readPump()
}
