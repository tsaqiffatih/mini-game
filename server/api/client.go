package api

import (
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
}

type ClientRegistry struct {
	clients     map[string]*Client
	generations map[string]uint64
	mu          sync.RWMutex
}

func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{
		clients:     make(map[string]*Client),
		generations: make(map[string]uint64),
	}
}

func (r *ClientRegistry) Attach(playerID string, conn *websocket.Conn, pongWait time.Duration) *Client {
	return r.AttachWithGeneration(playerID, 0, conn, pongWait)
}

func (r *ClientRegistry) AttachWithGeneration(playerID string, generation uint64, conn *websocket.Conn, pongWait time.Duration) *Client {
	return r.AttachWithGenerationInRoom("", playerID, generation, conn, pongWait)
}

func (r *ClientRegistry) AttachWithGenerationInRoom(roomID string, playerID string, generation uint64, conn *websocket.Conn, pongWait time.Duration) *Client {
	if generation == 0 {
		generation = r.NextGeneration(playerID)
	}

	client := &Client{
		PlayerID:     playerID,
		RoomID:       roomID,
		Generation:   generation,
		ConnectionID: playerID + "#" + strconv.FormatUint(generation, 10),
		Conn:         conn,
		Send:         make(chan []byte, 256),
		done:         make(chan struct{}),
	}

	conn.SetReadLimit(maxMessageBytes)
	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		observability.Logger().Warn("websocket read deadline setup failed",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "websocket_read_deadline_error",
			"generation", generation,
			"connection_id", client.ConnectionID,
			"error", err,
		)
	}
	conn.SetPongHandler(func(string) error {
		if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			observability.Logger().Warn("websocket pong deadline refresh failed",
				"room_id", roomID,
				"player_id", playerID,
				"event_type", "websocket_pong_deadline_error",
				"generation", generation,
				"connection_id", client.ConnectionID,
				"error", err,
			)
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
		deadline := time.Now().Add(writeWait)
		return conn.WriteControl(websocket.CloseMessage, message, deadline)
	})

	var existing *Client
	r.mu.Lock()
	if generation > r.generations[playerID] {
		r.generations[playerID] = generation
	}
	existing = r.clients[playerID]
	r.clients[playerID] = client
	r.mu.Unlock()

	if existing != nil {
		observability.Logger().Info("websocket duplicate connection replaced",
			"room_id", existing.RoomID,
			"player_id", playerID,
			"event_type", "websocket_duplicate_replaced",
			"close_code", CloseCodeDuplicateConnection,
			"generation", generation,
			"connection_id", client.ConnectionID,
			"previous_generation", existing.Generation,
			"previous_connection_id", existing.ConnectionID,
			"previous_room_id", existing.RoomID,
			"new_room_id", roomID,
		)
		existing.CloseWithCode(CloseCodeDuplicateConnection, "duplicate connection")
	}

	return client
}

func (r *ClientRegistry) NextGeneration(playerID string) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.generations[playerID]++
	return r.generations[playerID]
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

func (r *ClientRegistry) HasClientInRoom(playerID string, roomID string) bool {
	client, exists := r.Get(playerID)
	return exists && client.RoomID == roomID
}

func (r *ClientRegistry) IsCurrentDisconnectedGeneration(playerID string, generation uint64) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.clients[playerID]; exists {
		return false
	}
	return r.generations[playerID] == generation
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

func (c *Client) CloseWithCode(code int, reason string) {
	c.close.Do(func() {
		close(c.done)
		closeWebsocketWithCode(c.Conn, code, reason)
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
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				observability.Logger().Warn("websocket write deadline failed",
					"room_id", c.RoomID,
					"player_id", c.PlayerID,
					"event_type", "websocket_write_error",
					"generation", c.Generation,
					"connection_id", c.ConnectionID,
					"error", err,
				)
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
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
			if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				observability.Logger().Warn("websocket ping deadline failed",
					"room_id", c.RoomID,
					"player_id", c.PlayerID,
					"event_type", "websocket_ping_error",
					"generation", c.Generation,
					"connection_id", c.ConnectionID,
					"error", err,
				)
				return
			}
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
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
