package blindtest

import (
	"log"
	"time"
)

func startGame(room *Room) {
	room.mu.Lock()
	room.GameStarted = true
	room.RoundNumber = 0
	room.mu.Unlock()

	tracks, err := fetchTracksFromDeezer(room.Playlist, room.MaxRounds)
	if err != nil {
		log.Println("Error fetching tracks:", err)
		return
	}

	room.Tracks = tracks

	room.mu.RLock()
	for _, player := range room.Players {
		player.Conn.WriteJSON(Message{
			Type: "game_start",
			Data: map[string]interface{}{
				"maxRounds": room.MaxRounds,
			},
		})
	}
	room.mu.RUnlock()

	time.Sleep(2 * time.Second)

	for i := 0; i < room.MaxRounds && i < len(room.Tracks); i++ {
		room.mu.Lock()
		room.CurrentTrackIdx = i
		room.CurrentTrack = &room.Tracks[i]
		room.RoundNumber = i + 1
		room.RoundStartTime = time.Now()
		room.CorrectAnswers = make(map[string]bool)
		room.PlayerAnswers = make(map[string]*PlayerAnswer)
		room.mu.Unlock()

		startRound(room)
		time.Sleep(time.Duration(room.RoundTime) * time.Second)
		endRound(room)
		time.Sleep(5 * time.Second)
	}

	endGame(room)
}

func startRound(room *Room) {
	room.mu.RLock()
	defer room.mu.RUnlock()

	msg := Message{
		Type: "round_start",
		Data: map[string]interface{}{
			"round":   room.RoundNumber,
			"preview": room.CurrentTrack.Preview,
		},
	}

	for _, player := range room.Players {
		player.Conn.WriteJSON(msg)
	}
}

func endRound(room *Room) {
	room.mu.RLock()
	defer room.mu.RUnlock()

	msg := Message{
		Type: "round_end",
		Data: map[string]interface{}{
			"title":  room.CurrentTrack.Title,
			"artist": room.CurrentTrack.Artist,
			"album":  room.CurrentTrack.Album,
		},
	}

	for _, player := range room.Players {
		player.Conn.WriteJSON(msg)
	}

	broadcastPlayerList(room)
}

func endGame(room *Room) {
	room.mu.RLock()
	defer room.mu.RUnlock()

	players := make([]map[string]interface{}, 0)
	for _, p := range room.Players {
		players = append(players, map[string]interface{}{
			"username": p.Username,
			"score":    p.Score,
		})
	}

	msg := Message{
		Type: "game_end",
		Data: map[string]interface{}{
			"players": players,
		},
	}

	for _, player := range room.Players {
		player.Conn.WriteJSON(msg)
	}
}
