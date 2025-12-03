package main

import (
	"html/template"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// --------- Types utilisés pour le WebSocket ---------

type Joueur struct {
	Nom   string `json:"name"`
	Score int    `json:"score"`
}

type MessageRecu struct {
	Type     string            `json:"type"`     // "join" ou "answers"
	Nom      string            `json:"name"`     // pour "join"
	Reponses map[string]string `json:"answers"`  // pour "answers"
}

type EtatPartie struct {
	Type       string   `json:"type"`       // "state"
	Lettre     string   `json:"letter"`
	Categories []string `json:"categories"`
	Joueurs    []Joueur `json:"players"`
}

// --------- Variables globales ---------

var (
	modeleHTML *template.Template

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Pour un petit projet local, on autorise tout.
			return true
		},
	}

	mu         sync.Mutex
	categories []string
	lettre     rune

	// Chaque connexion WebSocket est associée à un Joueur
	clients = make(map[*websocket.Conn]*Joueur)
)

// Données pour le rendu de la page HTML (template)
type DonneesPage struct {
	Lettre     string
	Categories []string
}

func main() {
	var err error

	// Logique de base : catégories + lettre
	categories = ObtenirCategories()
	lettre = ObtenirLettreAleatoire()

	// Chargement du template HTML
	modeleHTML, err = template.ParseFiles("templates/ptitbac.html")
	if err != nil {
		log.Fatalf("Erreur de chargement du template : %v", err)
	}

	// Fichiers statiques (CSS)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Page principale
	http.HandleFunc("/", handlerPage)

	// Endpoint WebSocket
	http.HandleFunc("/ws", handlerWebSocket)

	log.Println("Serveur lancé sur http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Erreur serveur : %v", err)
	}
}

// --------- HTTP : page HTML ---------

func handlerPage(w http.ResponseWriter, r *http.Request) {
	donnees := DonneesPage{
		Lettre:     string(lettre),
		Categories: categories,
	}

	if err := modeleHTML.Execute(w, donnees); err != nil {
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
		log.Printf("Erreur template : %v", err)
	}
}

// --------- WebSocket : multi-joueurs temps réel ---------

func handlerWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP -> WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Erreur upgrade WebSocket : %v", err)
		return
	}

	// À la connexion, on crée un joueur "Anonyme"
	mu.Lock()
	clients[conn] = &Joueur{
		Nom:   "Anonyme",
		Score: 0,
	}
	mu.Unlock()

	// On envoie l'état initial à tout le monde
	diffuserEtat()

	// Boucle de lecture : on attend les messages du client
	go boucleLecture(conn)
}

func boucleLecture(conn *websocket.Conn) {
	defer func() {
		// À la déconnexion, on enlève le joueur et on diffuse l'état
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

		switch msg.Type {
		case "join":
			// Mise à jour du pseudo
			if msg.Nom != "" {
				joueur.Nom = msg.Nom
			}

		case "answers":
			// Calcul du score en fonction des réponses envoyées
			score := 0
			for cat, rep := range msg.Reponses {
				// On utilise EstValidePourLettre de logic.go
				if EstValidePourLettre(rep, lettre) {
					score++
				} else {
					_ = cat // cat est dispo si tu veux logger
				}
			}
			joueur.Score = score
		}

		mu.Unlock()

		// Après chaque message, on renvoie l'état global à tout le monde
		diffuserEtat()
	}
}

func diffuserEtat() {
	mu.Lock()
	defer mu.Unlock()

	etat := EtatPartie{
		Type:       "state",
		Lettre:     string(lettre),
		Categories: categories,
	}

	for _, j := range clients {
		etat.Joueurs = append(etat.Joueurs, *j)
	}

	for conn := range clients {
		if err := conn.WriteJSON(etat); err != nil {
			log.Printf("Erreur envoi WebSocket : %v", err)
		}
	}
}
