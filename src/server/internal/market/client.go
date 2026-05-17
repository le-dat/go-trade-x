package market

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait   = 10 * time.Second
	pongWait    = 30 * time.Second
	maxMsgSize  = 512
	sendBufSize = 256
)

// activeConnections decrementer - set by main to allow clients to report disconnections
var decrementConnections func()

// RegisterConnectionDecrementer sets the function to call when a client disconnects
func RegisterConnectionDecrementer(fn func()) {
	decrementConnections = fn
}

// Client represents a WebSocket client connected to the hub.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	symbol string
	done   chan struct{}
}

// NewClient creates a new client.
func NewClient(hub *Hub, conn *websocket.Conn, symbol string) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, sendBufSize),
		symbol: symbol,
		done:   make(chan struct{}),
	}
}

// Read pumps messages from the WebSocket to the hub.
func (c *Client) Read() {
	defer func() {
		if decrementConnections != nil {
			decrementConnections()
		}
		c.hub.Unsubscribe(c)
		c.conn.Close()
		close(c.done)
	}()

	c.conn.SetReadLimit(maxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// Write pumps messages from the hub to the WebSocket.
func (c *Client) Write() {
	ticker := time.NewTicker(pongWait)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Symbol returns the symbol this client is subscribed to.
func (c *Client) Symbol() string {
	return c.symbol
}