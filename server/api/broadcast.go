package api

import "github.com/tsaqiffatih/mini-game/internal/observability"

// BroadcastToRoom sends the given message to all clients connected to the specified room.
// It iterates over the internal client map and enqueues the message for each matching client.
func (r *ClientRegistry) BroadcastToRoom(roomID string, message []byte) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    for _, client := range r.clients {
        if client.RoomID == roomID {
            // Use non‑blocking enqueue; if the client is closed the send will be ignored.
            client.Enqueue(message)
        }
    }
    observability.Logger().Debug("broadcast to room", "room_id", roomID, "message_len", len(message))
}
