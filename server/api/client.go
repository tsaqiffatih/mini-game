package api

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsaqiffatih/mini-game/internal/observability"
)

type Client struct {
	PlayerID string
	Conn     *websocket.Conn
	Send     chan []byte
	done     chan struct{}
	close    sync.Once
}

type ClientRegistry struct {
	clients map[string]*Client
	mu      sync.RWMutex
}

func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{
		clients: make(map[string]*Client),
	}
}

func (r *ClientRegistry) Attach(playerID string, conn *websocket.Conn, pongWait time.Duration) *Client {
	client := &Client{
		PlayerID: playerID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		done:     make(chan struct{}),
	}

	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	r.mu.Lock()
	if existing := r.clients[playerID]; existing != nil {
		existing.Close()
	}
	r.clients[playerID] = client
	r.mu.Unlock()

	return client
}

func (r *ClientRegistry) Get(playerID string) (*Client, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	client, exists := r.clients[playerID]
	return client, exists
}

func (r *ClientRegistry) IsConnected(playerID string) bool {
	_, exists := r.Get(playerID)
	return exists
}

func (r *ClientRegistry) Remove(playerID string) {
	r.mu.Lock()
	client, exists := r.clients[playerID]
	if exists {
		delete(r.clients, playerID)
	}
	r.mu.Unlock()

	if exists {
		client.Close()
	}
}

func (r *ClientRegistry) RemoveClient(client *Client) bool {
	r.mu.Lock()
	current, exists := r.clients[client.PlayerID]
	if exists && current == client {
		delete(r.clients, client.PlayerID)
	}
	r.mu.Unlock()

	if exists && current == client {
		client.Close()
		return true
	}

	return false
}

func (r *ClientRegistry) CloseAll() {
	r.mu.Lock()
	clients := make([]*Client, 0, len(r.clients))
	for playerID, client := range r.clients {
		delete(r.clients, playerID)
		clients = append(clients, client)
	}
	r.mu.Unlock()

	for _, client := range clients {
		client.Close()
	}
}

func (c *Client) Enqueue(message []byte) bool {
	select {
	case <-c.done:
		return false
	default:
	}

	select {
	case c.Send <- message:
		return true
	case <-c.done:
		return false
	default:
		c.Close()
		return false
	}
}

func (c *Client) Close() {
	c.close.Do(func() {
		close(c.done)
		if c.Conn != nil {
			_ = c.Conn.Close()
		}
	})
}

func (c *Client) WritePump(writeWait time.Duration, pingPeriod time.Duration) {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.Close()

		if r := recover(); r != nil {
			observability.Logger().Error("websocket write pump recovered",
				"room_id", "",
				"player_id", c.PlayerID,
				"event_type", "websocket_write_recovered",
				"panic", r,
			)
		}
	}()

	for {
		select {
		case <-c.done:
			return
		case message := <-c.Send:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				observability.Logger().Warn("websocket write deadline failed",
					"room_id", "",
					"player_id", c.PlayerID,
					"event_type", "websocket_write_error",
					"error", err,
				)
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				observability.Logger().Warn("websocket write failed",
					"room_id", "",
					"player_id", c.PlayerID,
					"event_type", "websocket_write_error",
					"error", err,
				)
				return
			}

		case <-ticker.C:
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				observability.Logger().Warn("websocket ping deadline failed",
					"room_id", "",
					"player_id", c.PlayerID,
					"event_type", "websocket_ping_error",
					"error", err,
				)
				return
			}
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				observability.Logger().Warn("websocket ping failed",
					"room_id", "",
					"player_id", c.PlayerID,
					"event_type", "websocket_ping_error",
					"error", err,
				)
				return
			}
		}
	}
}
