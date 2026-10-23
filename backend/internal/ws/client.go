package ws

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// writeWait is the maximum duration allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// pongWait is the maximum duration allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// pingPeriod is the interval between sending ping frames to peer. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize is the maximum allowed size in bytes for inbound messages.
	maxMessageSize = 512

	// sendBufferSize is the capacity of the outbound message channel.
	sendBufferSize = 256
)

// Client is a middleman between the WebSocket connection and the Hub.
type Client struct {
	hub *Hub

	// The WebSocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// closeOnce ensures connection close and hub unregistration happen exactly once.
	closeOnce sync.Once
}

func newClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, sendBufferSize),
	}
}

func (c *Client) close() {
	c.closeOnce.Do(func() {
		select {
		case c.hub.unregister <- c:
		case <-c.hub.quit:
		}
		_ = c.conn.Close()
	})
}

// readPump pumps messages from the WebSocket connection to the hub.
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			// Client disconnected, network error, or unexpected close
			break
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
// A goroutine running writePump is started for each connection. The application
// ensures that there is at most one writer on a connection by executing all
// writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
