package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ====== Types pour le jeu ======

type Joueur struct {
	Nom      string            `json:"name"`
	Score    int               `json:"score"`
	Reponses map[string]string `json:"-"`
}

type MessageRecu struct {
	Type     string            `json:"type"`    // "join" ou "answers"
	Nom      string            `json:"name"`    // pour "join"
	Reponses map[string]string `json:"answers"` // pour "answers"
}

type EtatPartie struct {
	Type            string   `json:"type"` // "state"
	Lettre          string   `json:"letter"`
	Categories      []string `json:"categories"`
	Joueurs         []Joueur `json:"players"`
	RemainingSecond int      `json:"remainingSeconds"` // temps restant en secondes
	RoundActive     bool     `json:"roundActive"`      // true = manche en cours
}

// ====== Variables globales ======

var (
	modeleHTML *template.Template

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	mu               sync.Mutex
	categories       []string
	lettreCourante   rune
	clients          = make(map[*websocket.Conn]*Joueur)
	remainingSeconds int
	roundActive      bool
)

// Données envoyées au template HTML
type DonneesPage struct {
	Lettre     string
	Categories []string
}

// ====== main ======

func main() {
	var err error

	// Logique de base (logic.go)
	categories = ObtenirCategories()
	lettreCourante = ObtenirLettreAleatoire()

	// Charge template
	modeleHTML, err = template.ParseFiles("templates/ptitbac.html")
	if err != nil {
		log.Fatalf("Erreur de chargement du template : %v", err)
	}

	// Fichiers statiques
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Routes
	http.HandleFunc("/", handlerPage)
	http.HandleFunc("/ws", handlerWebSocket)

	// Démarre la première manche
	demarrerNouvelleManche()

	log.Println("Serveur lancé sur http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Erreur serveur : %v", err)
	}
}

// ====== Gestion des manches / timer ======

func demarrerNouvelleManche() {
	mu.Lock()
	defer mu.Unlock()

	lettreCourante = ObtenirLettreAleatoire()
	remainingSeconds = 90       // 1 min 30
	roundActive = true          // manche active
	for _, j := range clients { // reset des scores
		j.Score = 0
		j.Reponses = make(map[string]string)
	}
	log.Printf("Nouvelle manche, lettre = %c", lettreCourante)

	// On envoie l'état initial
	go diffuserEtat()

	// On (re)lance le timer
	go lancerTimer()
}

func lancerTimer() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mu.Lock()
		if !roundActive {
			mu.Unlock()
			return
		}

		if remainingSeconds > 0 {
			remainingSeconds--
		}

		// Timer arrivé à 0 ⇒ fin de manche
		if remainingSeconds == 0 {
			log.Println("Fin de manche par timer (0s)")
			roundActive = false
			attribuerScoresFinMancheVerrouille()
			mu.Unlock()
			diffuserEtat()
			return
		}
		mu.Unlock()

		diffuserEtat()
	}
}

func attribuerScoresFinMancheVerrouille() {
	if len(clients) == 0 {
		return
	}

	reponsesJoueurs := make([]map[string]string, 0, len(clients))
	ordreJoueurs := make([]*Joueur, 0, len(clients))

	for _, joueur := range clients {
		if joueur.Reponses == nil {
			joueur.Reponses = make(map[string]string)
		}
		reponsesJoueurs = append(reponsesJoueurs, joueur.Reponses)
		ordreJoueurs = append(ordreJoueurs, joueur)
	}

	scores := CalculerScoresCollectifs(reponsesJoueurs, categories, lettreCourante)
	for i, joueur := range ordreJoueurs {
		joueur.Score = scores[i]
	}
}

func arreterMancheParCompletion() {
	mu.Lock()
	defer mu.Unlock()

	if !roundActive {
		return
	}
	log.Println("Fin de manche : un joueur a rempli toutes les catégories")
	roundActive = false
	attribuerScoresFinMancheVerrouille()
	go diffuserEtat()
}

// ====== HTTP & WebSocket ======

func handlerPage(w http.ResponseWriter, r *http.Request) {
	d := DonneesPage{
		Lettre:     string(lettreCourante),
		Categories: categories,
	}
	if err := modeleHTML.Execute(w, d); err != nil {
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
		log.Printf("Erreur template : %v", err)
	}
}

func handlerWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Erreur upgrade WebSocket : %v", err)
		return
	}

	mu.Lock()
	clients[conn] = &Joueur{
		Nom:      "Anonyme",
		Score:    0,
		Reponses: make(map[string]string),
	}
	mu.Unlock()

	// On envoie l'état actuel à ce nouveau joueur et aux autres
	diffuserEtat()

	go boucleLecture(conn)
}

func boucleLecture(conn *websocket.Conn) {
	defer func() {
		mu.Lock()
		delete(clients, conn)
		mu.Unlock()
		conn.Close()
		diffuserEtat()
	}()

	for {
		var msg MessageRecu
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("Erreur lecture WebSocket : %v", err)
			return
		}

		mu.Lock()
		joueur, ok := clients[conn]
		if !ok {
			mu.Unlock()
			return
		}

		// Si la manche est finie, on ignore les réponses
		if !roundActive {
			mu.Unlock()
			continue
		}

		switch msg.Type {
		case "join":
			if strings.TrimSpace(msg.Nom) != "" {
				joueur.Nom = msg.Nom
			}

		case "answers":
			toutesRemplies := true
			if joueur.Reponses == nil {
				joueur.Reponses = make(map[string]string)
			}

			for _, cat := range categories {
				rep := ""
				if msg.Reponses != nil {
					rep = msg.Reponses[cat]
				}
				joueur.Reponses[cat] = rep
				if strings.TrimSpace(rep) == "" {
					toutesRemplies = false
				}
			}

			// Si ce joueur a rempli toutes les catégories (même sans valider la lettre),
			// on stoppe la manche.
			if toutesRemplies {
				mu.Unlock()
				arreterMancheParCompletion()
				continue
			}
		}

		mu.Unlock()
		diffuserEtat()
	}
}

// ====== Diffusion de l'état ======

func diffuserEtat() {
	// snapshot de l'état sous lock
	mu.Lock()
	etat := EtatPartie{
		Type:            "state",
		Lettre:          string(lettreCourante),
		Categories:      append([]string(nil), categories...),
		RemainingSecond: remainingSeconds,
		RoundActive:     roundActive,
	}
	joueurs := make([]Joueur, 0, len(clients))
	for _, j := range clients {
		joueurs = append(joueurs, *j)
	}
	etat.Joueurs = joueurs

	conns := make([]*websocket.Conn, 0, len(clients))
	for c := range clients {
		conns = append(conns, c)
	}
	mu.Unlock()

	// envoi sans tenir le lock (pour éviter les blocages)
	for _, c := range conns {
		if err := c.WriteJSON(etat); err != nil {
			log.Printf("Erreur envoi WebSocket : %v", err)
		}
	}
}
