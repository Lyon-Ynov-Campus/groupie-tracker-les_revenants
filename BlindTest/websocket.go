package blindtest

import (
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	playerID := uuid.New().String()
	var currentRoom *Room
	var currentPlayer *Player

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			if currentRoom != nil && currentPlayer != nil {
				handlePlayerDisconnect(currentRoom, currentPlayer)
			}
			break
		}

		switch msg.Type {
		case "create_room":
			maxRounds := msg.MaxRounds
			if maxRounds <= 0 || maxRounds > 20 {
				maxRounds = 10
			}
			roundTime := msg.RoundTime
			if roundTime < 10 || roundTime > 60 {
				roundTime = 30
			}
			playlist := msg.Playlist
			if playlist == "" {
				playlist = "pop"
			}

			room := createRoom(maxRounds, roundTime, playlist)
			player := &Player{
				ID:       playerID,
				Username: msg.Username,
				Conn:     conn,
				Score:    0,
				Ready:    false,
			}
			room.Players[playerID] = player
			currentRoom = room
			currentPlayer = player

			conn.WriteJSON(Message{
				Type:   "room_created",
				RoomID: room.ID,
				Data: map[string]interface{}{
					"playerId": playerID,
				},
			})

		case "join_room":
			roomsMu.RLock()
			room, exists := rooms[msg.RoomID]
			roomsMu.RUnlock()

			if !exists {
				conn.WriteJSON(Message{
					Type: "error",
					Data: map[string]interface{}{
						"message": "Room not found",
					},
				})
				continue
			}

			if room.GameStarted {
				conn.WriteJSON(Message{
					Type: "error",
					Data: map[string]interface{}{
						"message": "Game already started",
					},
				})
				continue
			}

			player := &Player{
				ID:       playerID,
				Username: msg.Username,
				Conn:     conn,
				Score:    0,
				Ready:    false,
			}

			room.mu.Lock()
			room.Players[playerID] = player
			room.mu.Unlock()

			currentRoom = room
			currentPlayer = player

			conn.WriteJSON(Message{
				Type:   "room_joined",
				RoomID: room.ID,
				Data: map[string]interface{}{
					"playerId": playerID,
				},
			})

			broadcastPlayerList(room)

		case "ready":
			if currentRoom != nil && currentPlayer != nil {
				currentPlayer.Ready = true
				broadcastPlayerList(currentRoom)

				allReady := true
				currentRoom.mu.RLock()
				playerCount := len(currentRoom.Players)
				for _, p := range currentRoom.Players {
					if !p.Ready {
						allReady = false
						break
					}
				}
				currentRoom.mu.RUnlock()

				if allReady && playerCount >= 1 {
					go startGame(currentRoom)
				}
			}

		case "answer":
			if currentRoom != nil && currentPlayer != nil && currentRoom.GameStarted {
				handleAnswer(currentRoom, currentPlayer, msg.Answer)
			}
		}
	}
}
