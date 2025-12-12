package petitbac

import (
	"sync"

	"github.com/gorilla/websocket"
)

type joueurDonnees struct {
	ID       string            `json:"id"`
	Nom      string            `json:"name"`
	Score    int               `json:"score"`
	Total    int               `json:"totalScore"`
	Reponses map[string]string `json:"-"`
	Pret     bool              `json:"ready"`
	Actif    bool              `json:"active"`
}

type messageJeu struct {
	Type     string            `json:"type"`
	Nom      string            `json:"name"`
	Reponses map[string]string `json:"answers"`
}

type paquetEtat struct {
	Type           string          `json:"type"`
	Lettre         string          `json:"letter"`
	Categories     []string        `json:"categories"`
	Joueurs        []joueurDonnees `json:"players"`
	Secondes       int             `json:"remainingSeconds"`
	MancheActive   bool            `json:"roundActive"`
	Attente        bool            `json:"waitingRestart"`
	CompteurPrets  int             `json:"readyCount"`
	CompteurTotal  int             `json:"readyTotal"`
	Actifs         int             `json:"activePlayers"`
	NumeroManche   int             `json:"roundNumber"`
	LimiteManches  int             `json:"roundLimit"`
	JeuTermine     bool            `json:"gameOver"`
	TempsParManche int             `json:"roundDuration"`
}

type donneesPage struct {
	Lettre          string
	Categories      []string
	TempsParManche  int
	NombreDeManches int
	SalonCode       string
}

type reglageJeu struct {
	Categories []string `json:"categories"`
	Temps      int      `json:"temps"`
	Manches    int      `json:"manches"`
}

type salon struct {
	code            string
	mu              sync.Mutex
	reglages        reglageJeu
	lettreActu      rune
	joueurs         map[*websocket.Conn]*joueurDonnees
	tempsRest       int
	mancheEnCours   bool
	attenteVotes    bool
	termine         bool
	nbManches       int
	compteurJoueurs int
}

type salonManager struct {
	mu     sync.RWMutex
	salons map[string]*salon
}
