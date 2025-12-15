package petitbac

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Player struct {
	ID       string            `json:"id"`
	Nom      string            `json:"name"`
	Score    int               `json:"score"`
	Total    int               `json:"totalScore"`
	Reponses map[string]string `json:"-"`
	Pret     bool              `json:"ready"`
	Actif    bool              `json:"active"`
	Conn     *websocket.Conn   `json:"-"`
}

type Message struct {
	Type     string            `json:"type"`
	Nom      string            `json:"name"`
	Reponses map[string]string `json:"answers"`
}

type GameState struct {
	Type           string    `json:"type"`
	Lettre         string    `json:"letter"`
	Categories     []string  `json:"categories"`
	Joueurs        []Player  `json:"players"`
	Secondes       int       `json:"remainingSeconds"`
	MancheActive   bool      `json:"roundActive"`
	Attente        bool      `json:"waitingRestart"`
	CompteurPrets  int       `json:"readyCount"`
	CompteurTotal  int       `json:"readyTotal"`
	Actifs         int       `json:"activePlayers"`
	NumeroManche   int       `json:"roundNumber"`
	LimiteManches  int       `json:"roundLimit"`
	JeuTermine     bool      `json:"gameOver"`
	TempsParManche int       `json:"roundDuration"`
}

type PageData struct {
	Lettre          string
	Categories      []string
	TempsParManche  int
	NombreDeManches int
	SalonCode       string
}

type GameConfig struct {
	Categories []string `json:"categories"`
	Temps      int      `json:"temps"`
	Manches    int      `json:"manches"`
}

type Room struct {
	code            string
	mu              sync.RWMutex
	reglages        GameConfig
	lettreActu      rune
	players         map[string]*Player
	connections     map[*websocket.Conn]string
	tempsRest       int
	mancheEnCours   bool
	attenteVotes    bool
	termine         bool
	nbManches       int
	compteurJoueurs int
}

var (
	rooms   = make(map[string]*Room)
	roomsMu sync.RWMutex
)
