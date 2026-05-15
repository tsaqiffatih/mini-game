package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsaqiffatih/mini-game/actions"
	"github.com/tsaqiffatih/mini-game/game"
)

type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Message struct {
	Action    string       `json:"action"`
	Message   interface{}  `json:"message"`
	Sender    *game.Player `json:"sender"`
	TimeStamp time.Time    `json:"timestamp"`
}

type ErrorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const (
	pingPeriod = 30 * time.Second
	pongWait   = 60 * time.Second
)

func HandleWebSocket(w http.ResponseWriter, r *http.Request, roomManager *game.RoomManager, playerManager *game.PlayerManager) {
	roomID := r.URL.Query().Get("room_id")
	playerID := r.URL.Query().Get("player_id")

	if !validateRoomAndPlayerIDs(w, roomID, playerID) {
		return
	}

	room, player, err := getRoomAndPlayer(w, roomManager, roomID, playerID)
	if err != nil {
		return
	}

	// for upgrade http connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		sendErrorResponse(w, "could not upgrade connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	setupPlayerConnection(player, conn)

	go player.WritePump()

	log.Printf("Player %s connected to room %s", playerID, roomID)

	notifyRoomOnConnection(roomManager, room, player)

	done := make(chan struct{})

	// goroutine to read messages
	go readMessages(conn, done, roomManager, room, player, playerManager)

	// goroutine to handle ping pong
	go handlePingPong(conn, done)

	<-done
	fmt.Println("Socket connection closed")

	// goroutine for remove player after a delay
	handlePlayerDisconnection(roomManager, room, playerID, player)
}

func setupPlayerConnection(player *game.Player, conn *websocket.Conn) {
	player.Conn = conn
	player.Send = make(chan []byte, 256)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
}

func handlePingPong(conn *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("Error sending ping:", err)
				return
			}
		case <-done:
			return
		}
	}
}

func handlePlayerDisconnection(roomManager *game.RoomManager, room *game.Room, playerID string, player *game.Player) {
	player.Conn = nil
	message := Message{
		Action:  actions.USER_LEFT_ROOM,
		Message: fmt.Sprintf("Player %s left the room", playerID),
		Sender:  player,
	}
	NotifyToClientsInRoom(roomManager, room.RoomID, &message)

	go removePlayerAfterDelay(roomManager, room, playerID)
}

func removePlayerAfterDelay(roomManager *game.RoomManager, room *game.Room, playerID string) {
	time.Sleep(30 * time.Second)
	room.Mu.Lock()
	defer room.Mu.Unlock()

	if player, exists := room.Players[playerID]; exists && player.Conn == nil {
		delete(room.Players, playerID)
		handleRoomAfterPlayerLeft(roomManager, room)
		log.Println("removedplayer", playerID)
	}
}
