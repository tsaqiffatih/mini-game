package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsaqiffatih/mini-game/actions"
	"github.com/tsaqiffatih/mini-game/api/dto"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/internal/observability"
	"github.com/tsaqiffatih/mini-game/service"
)

type websocketReadResult struct {
	CloseCode   int
	CloseReason string
}

// message.go
func readMessages(
	ctx context.Context,
	conn *websocket.Conn,
	done chan<- websocketReadResult,
	clients *ClientRegistry,
	gameService *service.GameService,
	roomID string,
	player game.PlayerSnapshot,
	client *Client,
) {
	result := websocketReadResult{CloseCode: websocket.CloseNormalClosure}
	defer func() {
		done <- result
	}()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			result = websocketReadCloseResult(err)
			observability.Logger().InfoContext(ctx, "websocket read closed",
				"room_id", roomID,
				"player_id", player.ID,
				"event_type", "websocket_read_closed",
				"close_code", result.CloseCode,
				"close_reason", result.CloseReason,
				"error", err,
			)
			break
		}

		var message WebSocketMessage
		if err := json.Unmarshal(msg, &message); err != nil {
			sendErrorMessage(client, "Invalid message format")
			observability.Logger().WarnContext(ctx, "websocket message unmarshal failed",
				"room_id", roomID,
				"player_id", player.ID,
				"event_type", "websocket_message_invalid",
				"error", err,
			)
			continue
		}

		observability.Logger().InfoContext(ctx, "websocket event received",
			"room_id", roomID,
			"player_id", player.ID,
			"event_type", message.Type,
		)

		gameService.UpdatePlayerActivityWithContext(ctx, roomID, player.ID)
		handleMessageAction(ctx, clients, gameService, roomID, player, client, message)
	}
}

func websocketReadCloseResult(err error) websocketReadResult {
	result := websocketReadResult{
		CloseCode: websocket.CloseAbnormalClosure,
	}

	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		result.CloseCode = closeErr.Code
		result.CloseReason = closeErr.Text
		return result
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		result.CloseReason = "timeout"
		return result
	}

	return result
}

func handleMessageAction(
	ctx context.Context,
	clients *ClientRegistry,
	gameService *service.GameService,
	roomID string,
	player game.PlayerSnapshot,
	client *Client,
	message WebSocketMessage,
) {
	switch message.Type {
	case actions.TICTACTOE_MOVE:
		processTicTacToeMove(ctx, player, client, clients, gameService, roomID, message)
	case actions.CHESS_MOVE:
		processChessMove(ctx, player, client, clients, gameService, roomID, player.ID, message)
	case actions.CHESS_UNDO_REQUEST:
		processChessUndo(ctx, player, client, clients, gameService, roomID)
	case actions.CHAT_SEND:
		processChatSend(ctx, player, client, clients, gameService, roomID, message)
	case actions.CREATE_ROOM_WITH_AI:
		var requestedRoomID string
		if err := json.Unmarshal(message.Payload, &requestedRoomID); err != nil {
			sendErrorMessage(client, "Invalid create room payload")
			return
		}
		event, err := gameService.CreateRoomWithAIByIDWithContext(ctx, requestedRoomID, "tictactoe")
		if err != nil {
			sendErrorMessage(client, err.Error())
			return
		}
		NotifyToClientsInRoom(clients, gameService, event.RoomID, EventRoomUpdate, nil)
	default:
		sendErrorMessage(client, "Unsupported message type")
		log.Println(message.Type, "<<<<<<<<<<<")
	}
}

func notifyRoomOnConnection(ctx context.Context, clients *ClientRegistry, gameService *service.GameService, roomID string, player game.PlayerSnapshot, eventType string) {
	if eventType != "" {
		NotifyToClientsInRoom(clients, gameService, roomID, eventType, EventPayload{
			Message:   connectionEventMessage(eventType, player.ID, roomID),
			Player:    playerEventDTO(player),
			Timestamp: time.Now(),
		})
	}

	snapshot, err := gameService.RoomSnapshotWithContext(ctx, roomID)
	if err != nil {
		observability.Logger().WarnContext(ctx, "room snapshot failed",
			"room_id", roomID,
			"player_id", player.ID,
			"event_type", "room_snapshot_error",
			"error", err,
		)
		return
	}

	sendRoomSnapshotToClient(clientForPlayer(clients, player.ID), snapshot)
	sendChatHistoryToClient(ctx, clients, gameService, roomID, player.ID)
}

func connectionEventMessage(eventType string, playerID string, roomID string) string {
	switch eventType {
	case EventPlayerReconnected:
		return fmt.Sprintf("Player %s reconnected to room %s", playerID, roomID)
	default:
		return fmt.Sprintf("Player %s connected to room %s", playerID, roomID)
	}
}

func notifyChessClients(clients *ClientRegistry, gameService *service.GameService, roomID string, player game.PlayerSnapshot) {
	snapshot, err := gameService.RoomSnapshot(roomID)
	if err != nil {
		observability.Logger().Warn("room snapshot failed",
			"room_id", roomID,
			"player_id", player.ID,
			"event_type", "room_snapshot_error",
			"error", err,
		)
		return
	}

	NotifySnapshotToClients(clients, snapshot, Event{
		Type:    EventGameUpdate,
		Payload: marshalPayload(dto.FromRoomSnapshot(snapshot)),
	})
}

func sendErrorMessage(client *Client, message string) {
	sendEvent(client, "error", ErrorMessage{Message: message})
}

func processChatSend(
	ctx context.Context,
	player game.PlayerSnapshot,
	client *Client,
	clients *ClientRegistry,
	gameService *service.GameService,
	roomID string,
	message WebSocketMessage,
) {
	var payload dto.ChatSendPayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		sendErrorMessage(client, "Invalid chat payload")
		return
	}

	chatMessage, err := gameService.HandleChatMessageWithContext(ctx, roomID, player.ID, payload.Message)
	if err != nil {
		sendErrorMessage(client, err.Error())
		return
	}

	snapshot, err := gameService.RoomSnapshotWithContext(ctx, roomID)
	if err != nil {
		observability.Logger().WarnContext(ctx, "room snapshot failed after chat message",
			"room_id", roomID,
			"player_id", player.ID,
			"event_type", "room_snapshot_error",
			"error", err,
		)
		return
	}

	NotifySnapshotToClients(clients, snapshot, Event{
		Type:    EventChatMessage,
		Payload: marshalPayload(dto.FromChatMessageEvent(chatMessage)),
	})
}

func sendChatHistoryToClient(ctx context.Context, clients *ClientRegistry, gameService *service.GameService, roomID string, playerID string) {
	client := clientForPlayer(clients, playerID)
	if client == nil {
		return
	}

	history, err := gameService.ChatHistoryWithContext(ctx, roomID)
	if err != nil {
		observability.Logger().WarnContext(ctx, "chat history failed",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "chat_history_error",
			"error", err,
		)
		return
	}

	sendEvent(client, EventChatHistory, dto.FromChatHistory(history))
}

func NotifyToClientsInRoom(
	clients *ClientRegistry,
	gameService *service.GameService,
	roomID string,
	eventType string,
	payload interface{},
) {
	snapshot, err := gameService.RoomSnapshot(roomID)
	if err != nil {
		observability.Logger().Warn("room snapshot failed",
			"room_id", roomID,
			"player_id", "",
			"event_type", "room_snapshot_error",
			"error", err,
		)
		return
	}

	if eventType == EventRoomUpdate {
		NotifyRoomUpdateToClients(clients, snapshot)
		return
	}

	NotifySnapshotToClients(clients, snapshot, Event{
		Type: eventType,
		Payload: marshalPayload(RoomEventPayload{
			Room: dto.FromRoomSnapshot(snapshot),
			Data: marshalPayload(payload),
		}),
	})
}

func NotifyRoomUpdateToClients(clients *ClientRegistry, snapshot game.RoomSnapshot) {
	NotifySnapshotToClients(clients, snapshot, Event{
		Type:    EventRoomUpdate,
		Payload: marshalPayload(dto.FromRoomSnapshot(snapshot)),
	})
}

func NotifySnapshotToClients(clients *ClientRegistry, snapshot game.RoomSnapshot, events ...Event) {
	if len(events) == 0 {
		events = []Event{{Type: EventRoomUpdate, Payload: marshalPayload(dto.FromRoomSnapshot(snapshot))}}
	}

	for _, event := range events {
		messageBytes, err := json.Marshal(event)
		if err != nil {
			observability.Logger().Warn("websocket event marshal failed",
				"room_id", snapshot.RoomID,
				"player_id", "",
				"event_type", event.Type,
				"error", err,
			)
			return
		}

		for _, player := range snapshot.Players {
			client, connected := clients.Get(player.ID)
			if !connected {
				continue
			}
			if !client.Enqueue(messageBytes) {
				clients.RemoveClient(client)
			}
		}
	}
}

func NotifyGameUpdateToClients(clients *ClientRegistry, snapshot game.RoomSnapshot) {
	NotifySnapshotToClients(clients, snapshot, Event{
		Type:    EventGameUpdate,
		Payload: marshalPayload(dto.FromRoomSnapshot(snapshot)),
	})
}

func sendRoomSnapshotToClient(client *Client, snapshot game.RoomSnapshot) {
	if client == nil {
		return
	}

	sendEvent(client, EventRoomUpdate, dto.FromRoomSnapshot(snapshot))
}

func sendEvent(client *Client, eventType string, payload interface{}) {
	if client == nil {
		return
	}

	messageBytes, err := json.Marshal(Event{
		Type:    eventType,
		Payload: marshalPayload(payload),
	})
	if err != nil {
		observability.Logger().Warn("websocket event marshal failed",
			"room_id", "",
			"player_id", client.PlayerID,
			"event_type", eventType,
			"error", err,
		)
		return
	}

	if !client.Enqueue(messageBytes) {
		observability.Logger().Warn("websocket client send queue full",
			"room_id", "",
			"player_id", client.PlayerID,
			"event_type", "websocket_slow_client",
		)
	}
}

func marshalPayload(payload interface{}) json.RawMessage {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		observability.Logger().Warn("websocket payload marshal failed",
			"room_id", "",
			"player_id", "",
			"event_type", "websocket_payload_error",
			"error", err,
		)
		return json.RawMessage(`{}`)
	}
	return payloadBytes
}

func clientForPlayer(clients *ClientRegistry, playerID string) *Client {
	client, connected := clients.Get(playerID)
	if !connected {
		return nil
	}
	return client
}
