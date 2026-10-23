// Package ws implements the WebSocket hub and connection handler for the OTT Dashboard backend.
package ws

import (
	"encoding/json"
	"sync"
	"time"
)

// Broadcaster is the interface that handlers use to publish real-time events.
// Decoupling via this interface allows handlers to be tested without running a real WebSocket hub.
type Broadcaster interface {
	Broadcast(event Event)
}

// Hub manages active WebSocket clients and broadcasts events to all connected clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound events to broadcast to all clients.
	broadcast chan Event

	// Register requests from clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Channel for graceful shutdown.
	quit chan struct{}

	// Protects clients map for concurrent reads like ClientCount().
	mu sync.RWMutex
}

// NewHub creates a new Hub instance ready to be started with Run.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		quit:       make(chan struct{}),
	}
}

// Run executes the hub's main event loop. It must be called in its own goroutine.
// It terminates when Stop is called.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			h.removeClientLocked(client)
			h.mu.Unlock()

		case event := <-h.broadcast:
			if event.Timestamp.IsZero() {
				event.Timestamp = time.Now().UTC()
			}
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}

			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- data:
				default:
					// Client buffer is full (slow consumer) — drop the client
					// to prevent it from blocking other clients.
					h.removeClientLocked(client)
				}
			}
			h.mu.Unlock()

		case <-h.quit:
			h.mu.Lock()
			for client := range h.clients {
				h.removeClientLocked(client)
			}
			h.mu.Unlock()
			return
		}
	}
}

// removeClientLocked removes a client from the registry and closes its send channel.
// Must be called while holding h.mu.
// It ensures close(client.send) is executed at most once across all code paths.
func (h *Hub) removeClientLocked(client *Client) {
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
}

// Broadcast sends an event to all connected clients asynchronously.
// If the broadcast channel is full, the event is dropped to prevent blocking callers.
func (h *Hub) Broadcast(event Event) {
	select {
	case h.broadcast <- event:
	default:
		// Broadcast channel full; drop event to avoid blocking HTTP handler
	}
}

// ClientCount returns the number of currently connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Stop shuts down the hub and disconnects all clients.
func (h *Hub) Stop() {
	select {
	case <-h.quit:
		// Already stopped
	default:
		close(h.quit)
	}
}
