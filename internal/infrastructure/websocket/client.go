package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/omanjaya/patra/pkg/logger"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 8192
)

// Client wraps a gorilla WebSocket connection.
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	closeOnce sync.Once
}

// closeSend safely closes the send channel exactly once, preventing double-close panics.
func (c *Client) closeSend() {
	c.closeOnce.Do(func() {
		close(c.send)
	})
}

func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 64),
	}
}

// ReadPump pumps messages from the WebSocket to the hub.
func (c *Client) ReadPump(rc *RoomClient) {
	defer func() {
		c.hub.Unregister <- c
		c.closeSend()
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	const maxConsecutiveErrors = 10
	consecutiveErrors := 0

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Log.Debugf("ws read error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutiveErrors {
				logger.Log.Warnf("ws: closing connection after %d consecutive unmarshal errors", consecutiveErrors)
				break
			}
			continue
		}
		consecutiveErrors = 0

		if msg.Event == EventHeartbeat {
			resp, _ := json.Marshal(Message{
				Event: EventTimeSync,
				Data:  TimeSyncPayload{ServerTime: time.Now().UTC().Format(time.RFC3339)},
			})
			select {
			case c.send <- resp:
			default:
			}
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		if r := recover(); r != nil {
			logger.Log.Debugf("ws writePump recovered from panic: %v", r)
		}
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				logger.Log.Debugf("ws writePump: write error: %v", err)
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Log.Debugf("ws writePump: ping error: %v", err)
				return
			}
		}
	}
}
