package main

import (
	"context"
	"database/sql"
	_ "embed"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	_ "modernc.org/sqlite"

	"blindTest/internal/game"
	"blindTest/internal/rooms"
	"blindTest/internal/ws"
)

var tplLayout *template.Template
var tplIndex *template.Template
var tplRoomBlindTest *template.Template
var db *sql.DB
var roomHubs = map[string]*ws.Hub{}
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var scoreboardActualPointInGame = map[int]int{} // userID -> points in current game

func main() {

	var err error
	db, err = sql.Open("sqlite", "./groupie.db")
	if err != nil {
		log.Fatal(err)
	}

	db.SetMaxOpenConns(1)
	db.Exec("PRAGMA foreign_keys = ON;")

	if err := migrate(db); err != nil {
		log.Fatal(err)
	}

	mustTpl := func(name, content string) *template.Template {
		tpl, err := template.New(name).Funcs(template.FuncMap{
			"eq": func(a, b any) bool { return a == b },
		}).Parse(content)
		if err != nil {
			log.Fatal(err)
		}
		return tpl
	}

	tplLayout = mustTpl("layout", layoutHTML)
	tplIndex = mustTpl("index", indexHTML)
	tplRoomBlindTest = mustTpl("room_blindtest", roomBlindTestHTML)

	http.HandleFunc("/", handleIndex)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))
	http.HandleFunc("/rooms/create", handleCreateRoomBlindTest)
	http.HandleFunc("/rooms/join", handleJoinRoom)
	http.HandleFunc("/rooms/", handleRoomPage)
	http.HandleFunc("/rooms/start", handleStartBlindTest)
	http.HandleFunc("/ws/room/", handleRoomWS)

	addr := getEnv("ADDR", ":8080")

	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(string(migrateSQL))
	return err
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"Title": "Groupie Tracker",
	}

	if err := tplIndex.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleCreateRoomBlindTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	playlist := r.FormValue("playlist")
	respTimeStr := r.FormValue("time")
	roundsStr := r.FormValue("rounds")

	if !rooms.ValidPlaylist(playlist) {
		http.Error(w, "invalid playlist", http.StatusBadRequest)
		return
	}

	respTime := 37
	if v, err := strconv.Atoi(respTimeStr); err == nil && v > 5 && v <= 120 {
		respTime = v
	}

	maxRounds := 5
	if v, err := strconv.Atoi(roundsStr); err == nil && v >= 1 && v <= 20 {
		maxRounds = v
	}

	roomID := randomCode(12)
	roomCode := strings.ToUpper(randomCode(6))

	_, err := db.Exec(`INSERT INTO rooms(id, code, host_user_id, game_type, status, created_at)
    VALUES (?, ?, ?, ?, ?, ?)`,
		roomID, roomCode, 1, "blindtest", "waiting", time.Now())
	if err != nil {
		http.Error(w, "cannot create room", http.StatusInternalServerError)
		return
	}

	cfg := game.BlindTestConfig{
		Playlist:        playlist,
		ResponseTimeSec: respTime,
		MaxRounds:       maxRounds,
	}

	if err := game.CreateGame(db, roomID, cfg); err != nil {
		http.Error(w, "cannot create game", http.StatusInternalServerError)
		return
	}

	hub := ws.NewHub()
	roomHubs[roomID] = hub
	go hub.Run()

	http.Redirect(w, r, "/rooms/"+roomID, http.StatusFound)
}

func handleJoinRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	code := strings.ToUpper(strings.TrimSpace(r.FormValue("code")))

	var roomID string
	var status string

	err := db.QueryRow(`SELECT id, status FROM rooms WHERE code = ? AND game_type = 'blindtest'`, code).Scan(&roomID, &status)
	if err != nil {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	if status == "finished" {
		http.Error(w, "room finished", http.StatusBadRequest)
		return
	}

	_, _ = db.Exec(`INSERT OR IGNORE INTO room_players(room_id, user_id, joined_at) VALUES (?, ?, ?)`, roomID, 1, time.Now())
	http.Redirect(w, r, "/rooms/"+roomID, http.StatusFound)
}

func handleRoomPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/rooms/"), "/")
	roomID := parts[0]

	var code, status string
	var gameID int

	err := db.QueryRow(`SELECT code, status FROM rooms WHERE id = ?`, roomID).Scan(&code, &status)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	err = db.QueryRow(`SELECT id FROM games WHERE room_id = ?`, roomID).Scan(&gameID)
	if err != nil {
		http.Error(w, "game missing", http.StatusInternalServerError)
		return
	}

	cfg, err := game.GetConfig(db, gameID)
	if err != nil {
		http.Error(w, "config missing", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":       "Blind Test",
		"RoomID":      roomID,
		"RoomCode":    code,
		"GameID":      gameID,
		"Status":      status,
		"Config":      cfg,
		"WSURL":       "/ws/room/" + roomID,
		"Description": "Choisis une des playlist !",
	}

	if err := tplRoomBlindTest.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleStartBlindTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	roomID := r.FormValue("room_id")

	var status string

	err := db.QueryRow(`SELECT status FROM rooms WHERE id=?`, roomID).Scan(&status)
	if err != nil {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	if status != "waiting" {
		http.Error(w, "already started", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`SELECT user_id FROM room_players WHERE room_id=?`, roomID)
	if err != nil {
		http.Error(w, "players error", http.StatusInternalServerError)
		return
	}

	defer rows.Close()
	var playerIDs []int
	for rows.Next() {
		var uid int
		if err := rows.Scan(&uid); err == nil {
			playerIDs = append(playerIDs, uid)
		}
	}

	rObj := rooms.Room{
		ID:        roomID,
		Code:      "",
		GameType:  "blindtest",
		Status:    status,
		PlayerIDs: playerIDs,
		MaxRounds: 1,
		Config: map[string]any{
			"playlist": "Rock",
		},
	}

	if !rooms.IsRoomReady(rObj) {
		http.Error(w, "room not ready", http.StatusBadRequest)
		return
	}

	_, _ = db.Exec(`UPDATE rooms SET status='in_progress' WHERE id=?`, roomID)

	var gameID int

	_ = db.QueryRow(`SELECT id FROM games WHERE room_id = ?`, roomID).Scan(&gameID)
	cfg, err := game.GetConfig(db, gameID)
	if err != nil {
		http.Error(w, "config error", http.StatusInternalServerError)
		return
	}

	hub := roomHubs[roomID]
	if hub == nil {
		http.Error(w, "ws hub missing", http.StatusInternalServerError)
		return
	}

	engine := game.NewBlindTestEngine(db, hub, gameID, cfg, scoreboardActualPointInGame)
	go engine.Run(context.Background())

	http.Redirect(w, r, "/rooms/"+roomID, http.StatusFound)
}

func handleRoomWS(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimPrefix(r.URL.Path, "/ws/room/")
	if roomID == "" {
		http.Error(w, "missing room", http.StatusBadRequest)
		return
	}
	hub := roomHubs[roomID]
	if hub == nil {
		hub = ws.NewHub()
		roomHubs[roomID] = hub
		go hub.Run()
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "upgrade failed", http.StatusInternalServerError)
		return
	}
	client := ws.NewClient(hub, conn)
	hub.Register <- client
	go client.ReadPump()
	go client.WritePump()
}

func randomCode(n int) string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

var layoutHTML string
var indexHTML string
var roomBlindTestHTML string
var migrateSQL []byte
