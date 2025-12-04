// main.go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

type Song struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	FilePath string `json:"file_path"`
	Genre    string `json:"genre"`
}

type Message struct {
	Type   string `json:"type"`
	Player string `json:"player"`
	Answer string `json:"answer"`
}

var (
	db        *sql.DB
	upgrader  = websocket.Upgrader{}
	clients   = make(map[*websocket.Conn]string) // Conn -> PlayerName
	scores    = make(map[string]int)
	current   Song
	startTime time.Time
)

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Création table si inexistante
	createTables()

	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/start", startRound)

	fmt.Println("Serveur démarré sur http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func createTables() {
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS songs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		file_path TEXT,
		genre TEXT
	);
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}

	// Exemple d'insertion si vide
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM songs").Scan(&count)
	if count == 0 {
		db.Exec("INSERT INTO songs (title, file_path, genre) VALUES (?, ?, ?)", "Rock Song 1", "/music/Cendrillon.mp3", "Rock")
		db.Exec("INSERT INTO songs (title, file_path, genre) VALUES (?, ?, ?)", "Rap Song 1", "/music/Un_Autre_Monde.mp3", "Rock")
		db.Exec("INSERT INTO songs (title, file_path, genre) VALUES (?, ?, ?)", "Pop Song 1", "/music/pop1.mp3", "Pop")
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()

	// Lecture des messages
	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Println("Client déconnecté")
			delete(clients, ws)
			break
		}

		if msg.Type == "join" {
			clients[ws] = msg.Player
			scores[msg.Player] = 0
		} else if msg.Type == "answer" {
			checkAnswer(msg.Player, msg.Answer)
		}
	}
}

func startRound(w http.ResponseWriter, r *http.Request) {
	genre := r.URL.Query().Get("genre")
	if genre == "" {
		genre = "Rock"
	}

	rows, err := db.Query("SELECT id, title, file_path, genre FROM songs WHERE genre = ?", genre)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var songs []Song
	for rows.Next() {
		var s Song
		rows.Scan(&s.ID, &s.Title, &s.FilePath, &s.Genre)
		songs = append(songs, s)
	}

	if len(songs) == 0 {
		http.Error(w, "Pas de chansons", 404)
		return
	}

	current = songs[rand.Intn(len(songs))]
	startTime = time.Now()

	// Diffusion aux joueurs
	for c := range clients {
		c.WriteJSON(map[string]interface{}{
			"type": "song",
			"file": current.FilePath,
		})
	}
}

func checkAnswer(player, answer string) {
	if answer == current.Title {
		elapsed := time.Since(startTime).Seconds()
		points := int(100 - elapsed*2)
		if points < 10 {
			points = 10
		}
		scores[player] += points

		// Broadcast scoreboard
		for c := range clients {
			c.WriteJSON(map[string]interface{}{
				"type":   "scoreboard",
				"scores": scores,
			})
		}
	}
}
