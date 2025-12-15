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
	Type         string            `json:"type"`
	Nom          string            `json:"name"`
	Reponses     map[string]string `json:"answers"`
	Approve      bool              `json:"approve,omitempty"`
	TargetID     string            `json:"targetId,omitempty"`
	Category     string            `json:"category,omitempty"`
	Answer       string            `json:"answer,omitempty"`
	ValidationID int               `json:"validationId,omitempty"`
}

type GameState struct {
	Type              string             `json:"type"`
	Lettre            string             `json:"letter"`
	Categories        []string           `json:"categories"`
	Joueurs           []Player           `json:"players"`
	Secondes          int                `json:"remainingSeconds"`
	MancheActive      bool               `json:"roundActive"`
	Attente           bool               `json:"waitingRestart"`
	CompteurPrets     int                `json:"readyCount"`
	CompteurTotal     int                `json:"readyTotal"`
	Actifs            int                `json:"activePlayers"`
	NumeroManche      int                `json:"roundNumber"`
	LimiteManches     int                `json:"roundLimit"`
	JeuTermine        bool               `json:"gameOver"`
	TempsParManche    int                `json:"roundDuration"`
	ValidationActive  bool               `json:"validationActive"`
	ValidationEntry   *ValidationDisplay `json:"validationEntry,omitempty"`
	ValidationPending int                `json:"validationPending"`
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
	code              string
	mu                sync.RWMutex
	reglages          GameConfig
	lettreActu        rune
	players           map[string]*Player
	connections       map[*websocket.Conn]string
	tempsRest         int
	mancheEnCours     bool
	attenteVotes      bool
	termine           bool
	nbManches         int
	compteurJoueurs   int
	validationActive  bool
	validationEntries []*validationEntry
	validationIndex   int
}

var (
	rooms   = make(map[string]*Room)
	roomsMu sync.RWMutex
)

type validationEntry struct {
	ID        int
	PlayerID  string
	PlayerNom string
	Category  string
	Answer    string
	Approvals map[string]bool
	Required  int
	Completed bool
	Accepted  bool
}

type ValidationDisplay struct {
	ID        int             `json:"id"`
	PlayerID  string          `json:"playerId"`
	PlayerNom string          `json:"playerName"`
	Category  string          `json:"category"`
	Answer    string          `json:"answer"`
	Required  int             `json:"required"`
	Votes     int             `json:"votes"`
	Approvals map[string]bool `json:"approvals"`
	Accepted  bool            `json:"accepted"`
	Completed bool            `json:"completed"`
}
