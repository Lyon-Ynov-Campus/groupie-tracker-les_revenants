package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var (
	clientID     string
	clientSecret string
	accessToken  string
	tokenExpiry  time.Time
)

type Game struct {
	RoomCode       string
	Players        map[string]*Player
	CurrentTrack   *Track
	RoundNumber    int
	MaxRounds      int
	Playlist       string
	RoundDuration  int
	Started        bool
	Finished       bool
	RoundStartTime time.Time
	mu             sync.RWMutex
	Clients        map[*Client]bool
	Broadcast      chan interface{}
}

type Client struct {
	Conn      *websocket.Conn
	Pseudonym string
	RoomCode  string
}

type Player struct {
	Pseudonym  string
	Score      int
	Answered   bool
	AnswerTime int
}

type Track struct {
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	URI        string `json:"uri"`
	PreviewURL string `json:"previewUrl"`
}

type GameUpdate struct {
	Type         string             `json:"type"`
	RoundNumber  int                `json:"roundNumber"`
	MaxRounds    int                `json:"maxRounds"`
	CurrentTrack *Track             `json:"currentTrack"`
	Players      map[string]*Player `json:"players"`
	Started      bool               `json:"started"`
	Finished     bool               `json:"finished"`
}

var games = make(map[string]*Game)
var gamesMu sync.RWMutex

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func init() {
	godotenv.Load()
	clientID = os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret = os.Getenv("SPOTIFY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatal("Missing SPOTIFY_CLIENT_ID or SPOTIFY_CLIENT_SECRET in .env")
	}
}

func getSpotifyToken() error {
	if time.Now().Before(tokenExpiry) && accessToken != "" {
		return nil
	}

	url := "https://accounts.spotify.com/api/token"
	data := fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", clientID, clientSecret)

	resp, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Printf("Failed to parse Spotify token response: %v\n", err)
		return err
	}

	token, ok := result["access_token"].(string)
	if !ok {
		log.Printf("Access token not found or wrong type in response: %v\n", result)
		return fmt.Errorf("invalid access token response")
	}

	expiresInInterface, ok := result["expires_in"]
	if !ok {
		return fmt.Errorf("expires_in not found in response")
	}

	expiresIn, ok := expiresInInterface.(float64)
	if !ok {
		return fmt.Errorf("expires_in is not a number")
	}

	accessToken = token
	tokenExpiry = time.Now().Add(time.Duration(expiresIn-60) * time.Second)
	log.Printf("Spotify token obtained successfully, expires in: %.0f seconds\n", expiresIn)

	return nil
}

func getPlaylistTracks(playlistName string) ([]Track, error) {
	if err := getSpotifyToken(); err != nil {
		return nil, err
	}

	seed := "rock"
	switch playlistName {
	case "Rock":
		seed = "rock"
	case "Rap":
		seed = "hip-hop"
	case "Pop":
		seed = "pop"
	default:
		seed = "rock"
	}

	url := fmt.Sprintf("https://api.spotify.com/v1/recommendations?seed_genres=%s&limit=50&min_popularity=50", seed)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Printf("Failed to parse Spotify response: %v\n", err)
		return nil, err
	}

	var tracks []Track

	tracksInterface, ok := result["tracks"]
	if !ok {
		log.Println("No tracks in response")
		return tracks, nil
	}

	tracksList, ok := tracksInterface.([]interface{})
	if !ok {
		log.Println("Tracks is not a slice")
		return tracks, nil
	}

	for _, trackItem := range tracksList {
		track, ok := trackItem.(map[string]interface{})
		if !ok {
			continue
		}

		title, _ := track["name"].(string)
		uri, _ := track["uri"].(string)

		if title == "" || uri == "" {
			continue
		}

		artist := ""
		if artists, ok := track["artists"].([]interface{}); ok && len(artists) > 0 {
			if artistMap, ok := artists[0].(map[string]interface{}); ok {
				artist, _ = artistMap["name"].(string)
			}
		}

		previewURL := ""
		if preview, ok := track["preview_url"].(string); ok {
			previewURL = preview
		}

		if previewURL != "" {
			tracks = append(tracks, Track{Title: title, Artist: artist, URI: uri, PreviewURL: previewURL})
		}
	}

	log.Printf("Fetched %d tracks from Spotify\n", len(tracks))
	return tracks, nil
}

func createGame(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Playlist      string `json:"playlist"`
		RoundDuration int    `json:"roundDuration"`
		MaxRounds     int    `json:"maxRounds"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	roomCode := generateRoomCode()
	game := &Game{
		RoomCode:      roomCode,
		Players:       make(map[string]*Player),
		MaxRounds:     req.MaxRounds,
		Playlist:      req.Playlist,
		RoundDuration: req.RoundDuration,
		Clients:       make(map[*Client]bool),
		Broadcast:     make(chan interface{}, 10),
	}

	gamesMu.Lock()
	games[roomCode] = game
	gamesMu.Unlock()

	go game.broadcastLoop()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"roomCode": roomCode})
}

func joinGame(w http.ResponseWriter, r *http.Request) {
	roomCode := r.URL.Query().Get("code")
	pseudonym := r.URL.Query().Get("pseudonym")

	gamesMu.RLock()
	game, exists := games[roomCode]
	gamesMu.RUnlock()

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	game.mu.Lock()
	game.Players[pseudonym] = &Player{Pseudonym: pseudonym, Score: 0}
	log.Printf("Player joined: %s, Total players: %d\n", pseudonym, len(game.Players))
	game.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "joined"})
}

func gameWebSocket(w http.ResponseWriter, r *http.Request) {
	roomCode := r.URL.Query().Get("code")
	pseudonym := r.URL.Query().Get("pseudonym")

	gamesMu.RLock()
	game, exists := games[roomCode]
	gamesMu.RUnlock()

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		Conn:      conn,
		Pseudonym: pseudonym,
		RoomCode:  roomCode,
	}

	game.mu.Lock()
	game.Clients[client] = true
	game.mu.Unlock()

	defer func() {
		game.mu.Lock()
		delete(game.Clients, client)
		game.mu.Unlock()
		conn.Close()
	}()

	for {
		var msg struct {
			Type   string `json:"type"`
			Answer string `json:"answer"`
		}

		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		game.mu.Lock()
		if msg.Type == "start" && !game.Started {
			game.Started = true
			go game.playRound()
		} else if msg.Type == "answer" {
			player := game.Players[pseudonym]
			if !player.Answered {
				player.Answered = true
				player.AnswerTime = int(time.Since(game.RoundStartTime).Seconds())
				if msg.Answer == game.CurrentTrack.Title {
					points := game.RoundDuration - player.AnswerTime
					if points < 0 {
						points = 0
					}
					player.Score += points
				}
			}
		}
		game.mu.Unlock()
	}
}

func (g *Game) broadcastLoop() {
	for msg := range g.Broadcast {
		g.mu.RLock()
		clients := make([]*Client, 0, len(g.Clients))
		for client := range g.Clients {
			clients = append(clients, client)
		}
		g.mu.RUnlock()

		for _, client := range clients {
			err := client.Conn.WriteJSON(msg)
			if err != nil {
				g.mu.Lock()
				delete(g.Clients, client)
				g.mu.Unlock()
			}
		}
	}
}

func (g *Game) playRound() {
	for round := 1; round <= g.MaxRounds; round++ {
		tracks, err := getPlaylistTracks(g.Playlist)
		if err != nil || len(tracks) == 0 {
			log.Println("Error fetching tracks:", err)
			return
		}

		g.mu.Lock()
		g.RoundNumber = round
		g.CurrentTrack = &tracks[round%len(tracks)]
		g.RoundStartTime = time.Now()

		for _, player := range g.Players {
			player.Answered = false
		}
		g.mu.Unlock()

		update := GameUpdate{
			Type:         "round_start",
			RoundNumber:  round,
			MaxRounds:    g.MaxRounds,
			CurrentTrack: g.CurrentTrack,
			Players:      g.Players,
			Started:      true,
			Finished:     false,
		}
		g.Broadcast <- update

		time.Sleep(time.Duration(g.RoundDuration) * time.Second)

		g.mu.Lock()
		roundCopy := make(map[string]*Player)
		for k, v := range g.Players {
			p := *v
			roundCopy[k] = &p
		}
		g.mu.Unlock()

		update = GameUpdate{
			Type:        "round_end",
			RoundNumber: round,
			MaxRounds:   g.MaxRounds,
			Players:     roundCopy,
			Started:     true,
			Finished:    false,
		}
		g.Broadcast <- update

		time.Sleep(2 * time.Second)
	}

	g.mu.Lock()
	g.Finished = true
	roundCopy := make(map[string]*Player)
	for k, v := range g.Players {
		p := *v
		roundCopy[k] = &p
	}
	g.mu.Unlock()

	update := GameUpdate{
		Type:     "game_end",
		Players:  roundCopy,
		Finished: true,
		Started:  true,
	}
	g.Broadcast <- update
}

func getGameStatus(w http.ResponseWriter, r *http.Request) {
	roomCode := r.URL.Query().Get("code")

	gamesMu.RLock()
	game, exists := games[roomCode]
	gamesMu.RUnlock()

	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	game.mu.RLock()
	defer game.mu.RUnlock()

	if game.Players == nil {
		game.Players = make(map[string]*Player)
	}

	response := map[string]interface{}{
		"RoomCode":      game.RoomCode,
		"Players":       game.Players,
		"CurrentTrack":  game.CurrentTrack,
		"RoundNumber":   game.RoundNumber,
		"MaxRounds":     game.MaxRounds,
		"Playlist":      game.Playlist,
		"RoundDuration": game.RoundDuration,
		"Started":       game.Started,
		"Finished":      game.Finished,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateRoomCode() string {
	return fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
}

func main() {
	http.HandleFunc("/api/create-game", createGame)
	http.HandleFunc("/api/join-game", joinGame)
	http.HandleFunc("/api/game-status", getGameStatus)
	http.HandleFunc("/ws/game", gameWebSocket)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
