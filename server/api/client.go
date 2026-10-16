package api

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsaqiffatih/mini-game/internal/observability"
)

type Client struct {
	PlayerID     string
	RoomID       string
	Generation   uint64
	ConnectionID string
	Conn         *websocket.Conn
	Send         chan []byte
	done         chan struct{}
	close        sync.Once
	writeMu      sync.Mutex
}

type clientKey struct {
	roomID   string
	playerID string
}

func newClientKey(roomID string, playerID string) clientKey {
	return clientKey{roomID: roomID, playerID: playerID}
}

type ClientRegistry struct {
	clients     map[clientKey]*Client
	generations map[clientKey]uint64
	mu          sync.RWMutex
}

func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{
		clients:     make(map[clientKey]*Client),
		generations: make(map[clientKey]uint64),
	}
}

func (r *ClientRegistry) Attach(playerID string, conn *websocket.Conn, pongWait time.Duration) *Client {
	return r.AttachWithGenerationInRoom("", playerID, 0, conn, pongWait)
}

func (r *ClientRegistry) AttachWithGeneration(playerID string, generation uint64, conn *websocket.Conn, pongWait time.Duration) *Client {
	return r.AttachWithGenerationInRoom("", playerID, generation, conn, pongWait)
}

func (r *ClientRegistry) AttachWithGenerationInRoom(roomID string, playerID string, generation uint64, conn *websocket.Conn, pongWait time.Duration) *Client {
	key := newClientKey(roomID, playerID)
	if generation == 0 {
		generation = r.NextGeneration(roomID, playerID)
	}

	client := &Client{
		PlayerID:     playerID,
		RoomID:       roomID,
		Generation:   generation,
		ConnectionID: roomID + "#" + playerID + "#" + strconv.FormatUint(generation, 10),
		Conn:         conn,
		Send:         make(chan []byte, 256),
		done:         make(chan struct{}),
	}

	conn.SetReadLimit(maxMessageBytes)
	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		observability.Logger().Warn("websocket read deadline setup failed",
			"room_id", roomID, "player_id", playerID, "event_type", "websocket_read_deadline_error",
			"generation", generation, "connection_id", client.ConnectionID, "error", err)
	}
	conn.SetPongHandler(func(string) error {
		if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			observability.Logger().Warn("websocket pong deadline refresh failed",
				"room_id", roomID, "player_id", playerID, "event_type", "websocket_pong_deadline_error",
				"generation", generation, "connection_id", client.ConnectionID, "error", err)
			return err
		}
		observability.Logger().Debug("websocket pong received",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "websocket_pong",
			"generation", generation,
			"connection_id", client.ConnectionID,
		)
		return nil
	})
	conn.SetCloseHandler(func(code int, text string) error {
		observability.Logger().Info("websocket close frame received",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "websocket_close_frame_received",
			"generation", generation,
			"connection_id", client.ConnectionID,
			"close_code", code,
			"close_reason", text,
		)
		message := websocket.FormatCloseMessage(code, "")
		// Use synchronized WriteMessage to prevent race conditions with WritePump or CloseWithCode.
		_ = client.WriteMessage(websocket.CloseMessage, message, writeWait)
		return nil
	})

	r.mu.Lock()
	if generation > r.generations[key] {
		r.generations[key] = generation
	}
	existing := r.clients[key]
	r.clients[key] = client
	r.mu.Unlock()

	if existing != nil {
		// Replace the existing client with the new connection.
		observability.Logger().Info(
			"websocket duplicate connection replaced",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "websocket_duplicate_replaced",
			"close_code", CloseCodeDuplicateConnection,
			"generation", generation,
			"connection_id", client.ConnectionID,
			"previous_generation", existing.Generation,
			"previous_connection_id", existing.ConnectionID,
		)
		// Close the old client.
		existing.CloseWithCode(CloseCodeDuplicateConnection, "duplicate connection")
		// Keep the new client.
		return client
	}
	// No existing client, continue with new client.

	return client
}

func (r *ClientRegistry) NextGeneration(roomID string, playerID string) uint64 {
	key := newClientKey(roomID, playerID)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.generations[key]++
	return r.generations[key]
}

func (r *ClientRegistry) Get(roomID string, playerID string) (*Client, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	client, exists := r.clients[newClientKey(roomID, playerID)]
	return client, exists
}

func (r *ClientRegistry) IsConnected(roomID string, playerID string) bool {
	_, exists := r.Get(roomID, playerID)
	return exists
}

func (r *ClientRegistry) HasClientInRoom(playerID string, roomID string) bool {
	return r.IsConnected(roomID, playerID)
}

func (r *ClientRegistry) IsCurrentDisconnectedGenerationInRoom(roomID string) func(string, uint64) bool {
	return func(playerID string, generation uint64) bool {
		key := newClientKey(roomID, playerID)
		r.mu.RLock()
		defer r.mu.RUnlock()
		if _, exists := r.clients[key]; exists {
			return false
		}
		return r.generations[key] == generation
	}
}

func (r *ClientRegistry) Remove(roomID string, playerID string) {
	key := newClientKey(roomID, playerID)
	r.mu.Lock()
	client, exists := r.clients[key]
	if exists {
		delete(r.clients, key)
	}
	r.mu.Unlock()
	if exists {
		client.Close()
	}
}

func (r *ClientRegistry) RemoveClient(client *Client) bool {
	key := newClientKey(client.RoomID, client.PlayerID)
	r.mu.Lock()
	current, exists := r.clients[key]
	if exists && current == client {
		delete(r.clients, key)
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
	for key, client := range r.clients {
		delete(r.clients, key)
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

func (c *Client) WriteMessage(messageType int, data []byte, timeout time.Duration) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.Conn == nil {
		return errors.New("connection is nil")
	}
	if err := c.Conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}
	return c.Conn.WriteMessage(messageType, data)
}

func (c *Client) CloseWithCode(code int, reason string) {
	c.close.Do(func() {
		close(c.done)
		message := websocket.FormatCloseMessage(code, reason)
		_ = c.WriteMessage(websocket.CloseMessage, message, closeFrameWait)
	})
}

func (c *Client) WritePump(writeWait time.Duration, pingPeriod time.Duration) {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.Close()

		if r := recover(); r != nil {
			observability.Logger().Error("websocket write pump recovered",
				"room_id", c.RoomID,
				"player_id", c.PlayerID,
				"event_type", "websocket_write_recovered",
				"generation", c.Generation,
				"connection_id", c.ConnectionID,
				"panic", r,
			)
		}
	}()

	for {
		select {
		case <-c.done:
			return
		case message := <-c.Send:
			if err := c.WriteMessage(websocket.TextMessage, message, writeWait); err != nil {
				observability.Logger().Warn("websocket write failed",
					"room_id", c.RoomID,
					"player_id", c.PlayerID,
					"event_type", "websocket_write_error",
					"generation", c.Generation,
					"connection_id", c.ConnectionID,
					"error", err,
				)
				return
			}

		case <-ticker.C:
			if err := c.WriteMessage(websocket.PingMessage, nil, writeWait); err != nil {
				observability.Logger().Warn("websocket ping failed",
					"room_id", c.RoomID,
					"player_id", c.PlayerID,
					"event_type", "websocket_ping_error",
					"generation", c.Generation,
					"connection_id", c.ConnectionID,
					"error", err,
				)
				return
			}
		}
	}
}
