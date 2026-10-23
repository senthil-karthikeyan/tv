package ws_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"tv/internal/ws"
)

// waitFor waits until predicate returns true or timeout expires.
func waitFor(t *testing.T, timeout time.Duration, predicate func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return predicate()
}

func TestHub_RegistrationAndBroadcast(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()
	defer hub.Stop()

	// Connect client via httptest server
	handler := ws.NewHandler(hub, "*")
	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial failed: %v (response status: %v)", err, resp)
	}
	defer conn.Close()

	// Wait for client to be registered in the hub
	if !waitFor(t, 500*time.Millisecond, func() bool { return hub.ClientCount() == 1 }) {
		t.Fatalf("expected 1 client registered, got %d", hub.ClientCount())
	}

	// Broadcast an event
	evt := ws.Event{
		Type:       ws.EventSceneCreated,
		Resource:   "scene",
		ResourceID: "sc_test_1",
		Operation:  ws.OpCreate,
		Data: map[string]string{
			"id":   "sc_test_1",
			"name": "Test Scene",
		},
	}
	hub.Broadcast(evt)

	// Read message from websocket connection
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	messageType, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read broadcast message: %v", err)
	}
	if messageType != websocket.TextMessage {
		t.Fatalf("expected TextMessage (%d), got %d", websocket.TextMessage, messageType)
	}

	var received ws.Event
	if err := json.Unmarshal(data, &received); err != nil {
		t.Fatalf("failed to unmarshal broadcast JSON: %v", err)
	}

	if received.Type != ws.EventSceneCreated {
		t.Errorf("expected Type %q, got %q", ws.EventSceneCreated, received.Type)
	}
	if received.Resource != "scene" {
		t.Errorf("expected Resource 'scene', got %q", received.Resource)
	}
	if received.ResourceID != "sc_test_1" {
		t.Errorf("expected ResourceID 'sc_test_1', got %q", received.ResourceID)
	}
	if received.Operation != ws.OpCreate {
		t.Errorf("expected Operation %q, got %q", ws.OpCreate, received.Operation)
	}
	if received.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	// Close client connection and verify unregistration
	_ = conn.Close()
	if !waitFor(t, 500*time.Millisecond, func() bool { return hub.ClientCount() == 0 }) {
		t.Errorf("expected 0 clients after disconnect, got %d", hub.ClientCount())
	}
}

func TestHub_OriginValidation(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()
	defer hub.Stop()

	allowedOrigin := "http://localhost:4200"
	handler := ws.NewHandler(hub, allowedOrigin)
	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	t.Run("allowed origin succeeds", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Origin", allowedOrigin)
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, headers)
		if err != nil {
			t.Fatalf("expected connection to succeed with allowed origin: %v", err)
		}
		defer conn.Close()
		if resp.StatusCode != http.StatusSwitchingProtocols {
			t.Errorf("expected status 101, got %d", resp.StatusCode)
		}
	})

	t.Run("disallowed origin rejected with 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Origin", "http://evil-site.com")
		_, resp, err := websocket.DefaultDialer.Dial(wsURL, headers)
		if err == nil {
			t.Fatal("expected connection to be rejected for disallowed origin")
		}
		if resp == nil || resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected status 403 Forbidden, got %v", resp)
		}
	})

	t.Run("empty origin succeeds (non-browser clients)", func(t *testing.T) {
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("expected non-browser client without Origin header to succeed: %v", err)
		}
		defer conn.Close()
		if resp.StatusCode != http.StatusSwitchingProtocols {
			t.Errorf("expected status 101, got %d", resp.StatusCode)
		}
	})
}

func TestHub_StopShutsDownCleanly(t *testing.T) {
	hub := ws.NewHub()
	go hub.Run()

	handler := ws.NewHandler(hub, "*")
	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	if !waitFor(t, 500*time.Millisecond, func() bool { return hub.ClientCount() == 1 }) {
		t.Fatalf("expected 1 client, got %d", hub.ClientCount())
	}

	// Stop the hub
	hub.Stop()

	// Read from client should see EOF or close message
	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Error("expected read error after hub shutdown, got nil")
	}
}
