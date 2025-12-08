package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Player struct {
	ID       string
	Username string
	Conn     *websocket.Conn
	Score    int
	Ready    bool
}

type Track struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Preview  string `json:"preview"`
	Album    string `json:"album"`
	Duration int    `json:"duration"`
}

type PlayerAnswer struct {
	FoundTitle  bool
	FoundArtist bool
	TimeTitle   time.Time
	TimeArtist  time.Time
}

type Room struct {
	ID              string
	Players         map[string]*Player
	CurrentTrack    *Track
	RoundNumber     int
	GameStarted     bool
	RoundStartTime  time.Time
	CorrectAnswers  map[string]bool
	PlayerAnswers   map[string]*PlayerAnswer
	Tracks          []Track
	CurrentTrackIdx int
	MaxRounds       int
	RoundTime       int
	Playlist        string
	mu              sync.RWMutex
}

type Message struct {
	Type      string                 `json:"type"`
	Username  string                 `json:"username,omitempty"`
	RoomID    string                 `json:"roomId,omitempty"`
	Answer    string                 `json:"answer,omitempty"`
	Playlist  string                 `json:"playlist,omitempty"`
	MaxRounds int                    `json:"maxRounds,omitempty"`
	RoundTime int                    `json:"roundTime,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

var (
	rooms   = make(map[string]*Room)
	roomsMu sync.RWMutex
)

func main() {
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", handleWebSocket)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
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

func handleAnswer(room *Room, player *Player, answer string) {
	room.mu.Lock()
	defer room.mu.Unlock()

	if room.CorrectAnswers[player.ID] {
		return
	}

	if !room.GameStarted || room.CurrentTrack == nil {
		return
	}

	normalizedAnswer := normalizeString(answer)
	normalizedTitle := normalizeString(room.CurrentTrack.Title)
	normalizedArtist := normalizeString(room.CurrentTrack.Artist)

	foundTitle := contains(normalizedAnswer, normalizedTitle) || contains(normalizedTitle, normalizedAnswer)
	foundArtist := contains(normalizedAnswer, normalizedArtist) || contains(normalizedArtist, normalizedAnswer)

	if !foundTitle && !foundArtist {
		return
	}

	if room.PlayerAnswers[player.ID] == nil {
		room.PlayerAnswers[player.ID] = &PlayerAnswer{}
	}

	playerAnswer := room.PlayerAnswers[player.ID]
	hadBothBefore := playerAnswer.FoundTitle && playerAnswer.FoundArtist

	if foundTitle && !playerAnswer.FoundTitle {
		playerAnswer.FoundTitle = true
		playerAnswer.TimeTitle = time.Now()
	}

	if foundArtist && !playerAnswer.FoundArtist {
		playerAnswer.FoundArtist = true
		playerAnswer.TimeArtist = time.Now()
	}

	hasBothNow := playerAnswer.FoundTitle && playerAnswer.FoundArtist

	var points int
	var color string
	var answerType string

	if hasBothNow && !hadBothBefore {
		room.CorrectAnswers[player.ID] = true

		var elapsed float64
		if foundTitle && foundArtist {
			elapsed = time.Since(room.RoundStartTime).Seconds()
			answerType = "both"
		} else if foundTitle {
			elapsed = time.Since(room.RoundStartTime).Seconds()
			answerType = "title_completing"
		} else {
			elapsed = time.Since(room.RoundStartTime).Seconds()
			answerType = "artist_completing"
		}

		points = calculatePoints(elapsed)
		player.Score += points
		color = "green"

		for _, p := range room.Players {
			p.Conn.WriteJSON(Message{
				Type: "correct_answer",
				Data: map[string]interface{}{
					"username":    player.Username,
					"points":      points,
					"answerType":  answerType,
					"color":       color,
					"foundTitle":  playerAnswer.FoundTitle,
					"foundArtist": playerAnswer.FoundArtist,
				},
			})
		}
	} else if !hadBothBefore {
		elapsed := time.Since(room.RoundStartTime).Seconds()
		points = calculatePoints(elapsed) / 2

		player.Score += points

		if foundTitle {
			answerType = "title_partial"
			color = "orange"
		} else {
			answerType = "artist_partial"
			color = "orange"
		}

		for _, p := range room.Players {
			p.Conn.WriteJSON(Message{
				Type: "correct_answer",
				Data: map[string]interface{}{
					"username":    player.Username,
					"points":      points,
					"answerType":  answerType,
					"color":       color,
					"foundTitle":  playerAnswer.FoundTitle,
					"foundArtist": playerAnswer.FoundArtist,
				},
			})
		}
	}
}

func normalizeString(s string) string {
	result := ""
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if r >= 'A' && r <= 'Z' {
				result += string(r + 32)
			} else {
				result += string(r)
			}
		}
	}
	return result
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func calculatePoints(elapsed float64) int {
	if elapsed < 5 {
		return 1000
	} else if elapsed < 10 {
		return 800
	} else if elapsed < 15 {
		return 600
	} else if elapsed < 20 {
		return 400
	} else if elapsed < 25 {
		return 200
	}
	return 100
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

func fetchTracksFromDeezer(playlist string, limit int) ([]Track, error) {
	if playlist == "generale" {
		return fetchMixedGenreTracks(limit)
	}

	if playlist == "francaise" {
		return fetchFrenchTracks(limit)
	}

	genreID := getGenreID(playlist)

	url := fmt.Sprintf("https://api.deezer.com/chart/%d/tracks?limit=100", genreID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			Title   string `json:"title"`
			Preview string `json:"preview"`
			Artist  struct {
				Name string `json:"name"`
			} `json:"artist"`
			Album struct {
				Title string `json:"title"`
			} `json:"album"`
			Duration int `json:"duration"`
		} `json:"data"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	tracks := make([]Track, 0)
	for _, item := range result.Data {
		if item.Preview != "" {
			tracks = append(tracks, Track{
				Title:    item.Title,
				Artist:   item.Artist.Name,
				Preview:  item.Preview,
				Album:    item.Album.Title,
				Duration: item.Duration,
			})
		}
	}

	rand.Shuffle(len(tracks), func(i, j int) {
		tracks[i], tracks[j] = tracks[j], tracks[i]
	})

	if len(tracks) > limit {
		tracks = tracks[:limit]
	}

	return tracks, nil
}

func fetchMixedGenreTracks(limit int) ([]Track, error) {
	allGenres := []string{"pop", "rock", "rap", "electronic", "indie", "classic", "country", "jazz", "blues", "reggae", "rnb", "soul", "metal", "alternative", "techno"}

	tracksPerGenre := 10
	allTracks := make([]Track, 0)

	for _, genre := range allGenres {
		tracks, err := fetchTracksFromGenre(genre, tracksPerGenre)
		if err == nil {
			allTracks = append(allTracks, tracks...)
		}
	}

	rand.Shuffle(len(allTracks), func(i, j int) {
		allTracks[i], allTracks[j] = allTracks[j], allTracks[i]
	})

	if len(allTracks) > limit {
		allTracks = allTracks[:limit]
	}

	return allTracks, nil
}

func fetchFrenchTracks(limit int) ([]Track, error) {
	url := "https://api.deezer.com/playlist/1313621735/tracks?limit=100"

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			Title   string `json:"title"`
			Preview string `json:"preview"`
			Artist  struct {
				Name string `json:"name"`
			} `json:"artist"`
			Album struct {
				Title string `json:"title"`
			} `json:"album"`
			Duration int `json:"duration"`
		} `json:"data"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	tracks := make([]Track, 0)
	for _, item := range result.Data {
		if item.Preview != "" {
			tracks = append(tracks, Track{
				Title:    item.Title,
				Artist:   item.Artist.Name,
				Preview:  item.Preview,
				Album:    item.Album.Title,
				Duration: item.Duration,
			})
		}
	}

	rand.Shuffle(len(tracks), func(i, j int) {
		tracks[i], tracks[j] = tracks[j], tracks[i]
	})

	if len(tracks) > limit {
		tracks = tracks[:limit]
	}

	return tracks, nil
}

func fetchTracksFromGenre(genre string, limit int) ([]Track, error) {
	genreID := getGenreID(genre)
	if genreID == 0 {
		return []Track{}, nil
	}

	url := fmt.Sprintf("https://api.deezer.com/chart/%d/tracks?limit=%d", genreID, limit)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			Title   string `json:"title"`
			Preview string `json:"preview"`
			Artist  struct {
				Name string `json:"name"`
			} `json:"artist"`
			Album struct {
				Title string `json:"title"`
			} `json:"album"`
			Duration int `json:"duration"`
		} `json:"data"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	tracks := make([]Track, 0)
	for _, item := range result.Data {
		if item.Preview != "" {
			tracks = append(tracks, Track{
				Title:    item.Title,
				Artist:   item.Artist.Name,
				Preview:  item.Preview,
				Album:    item.Album.Title,
				Duration: item.Duration,
			})
		}
	}

	return tracks, nil
}

func getGenreID(playlist string) int {
	genreMap := map[string]int{
		"pop":         132,
		"rock":        152,
		"rap":         116,
		"electronic":  106,
		"indie":       85,
		"classic":     98,
		"country":     2,
		"jazz":        129,
		"blues":       153,
		"reggae":      144,
		"rnb":         165,
		"soul":        169,
		"metal":       464,
		"alternative": 85,
		"latin":       197,
	}

	if id, exists := genreMap[playlist]; exists {
		return id
	}

	return 0
}
