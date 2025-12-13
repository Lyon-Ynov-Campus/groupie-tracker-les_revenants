package blindtest

import "math/rand"

func createRoom(maxRounds, roundTime int, playlist string) *Room {
	roomID := generateRoomCode()
	room := &Room{
		ID:              roomID,
		Players:         make(map[string]*Player),
		GameStarted:     false,
		CorrectAnswers:  make(map[string]bool),
		PlayerAnswers:   make(map[string]*PlayerAnswer),
		CurrentTrackIdx: 0,
		MaxRounds:       maxRounds,
		RoundTime:       roundTime,
		Playlist:        playlist,
	}

	roomsMu.Lock()
	rooms[roomID] = room
	roomsMu.Unlock()

	return room
}

func generateRoomCode() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func broadcastPlayerList(room *Room) {
	room.mu.RLock()
	defer room.mu.RUnlock()

	players := make([]map[string]interface{}, 0)
	for _, p := range room.Players {
		players = append(players, map[string]interface{}{
			"id":       p.ID,
			"username": p.Username,
			"score":    p.Score,
			"ready":    p.Ready,
		})
	}

	msg := Message{
		Type: "player_list",
		Data: map[string]interface{}{
			"players": players,
		},
	}

	for _, player := range room.Players {
		player.Conn.WriteJSON(msg)
	}
}

func handlePlayerDisconnect(room *Room, player *Player) {
	room.mu.Lock()
	delete(room.Players, player.ID)
	room.mu.Unlock()

	if !room.GameStarted {
		broadcastPlayerList(room)
	}

	room.mu.RLock()
	playerCount := len(room.Players)
	room.mu.RUnlock()

	if playerCount == 0 {
		roomsMu.Lock()
		delete(rooms, room.ID)
		roomsMu.Unlock()
	}
}
