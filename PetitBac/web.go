package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Joueur struct {
	ID         string            `json:"id"`
	Nom        string            `json:"name"`
	Score      int               `json:"score"`
	ScoreTotal int               `json:"totalScore"`
	Reponses   map[string]string `json:"-"`
	Pret       bool              `json:"ready"`
	Actif      bool              `json:"active"`
}

type MessageRecu struct {
	Type     string            `json:"type"`    // "join", "answers", "ready"
	Nom      string            `json:"name"`    // pour "join"
	Reponses map[string]string `json:"answers"` // pour "answers"
}

type EtatPartie struct {
	Type                   string   `json:"type"`
	Lettre                 string   `json:"letter"`
	Categories             []string `json:"categories"`
	Joueurs                []Joueur `json:"players"`
	RemainingSecond        int      `json:"remainingSeconds"`
	RoundActive            bool     `json:"roundActive"`
	WaitingRestart         bool     `json:"waitingRestart"`
	ReadyCount             int      `json:"readyCount"`
	ReadyTotal             int      `json:"readyTotal"`
	ActivePlayersThisRound int      `json:"activePlayers"`
	ManchesJouees          int      `json:"roundNumber"`
	ManchesMaximum         int      `json:"roundLimit"`
	PartieTerminee         bool     `json:"gameOver"`
	TempsParManche         int      `json:"roundDuration"`
}

type DonneesPage struct {
	Lettre          string
	Categories      []string
	TempsParManche  int
	NombreDeManches int
}

type ParametresPartie struct {
	Categories   []string
	TempsManche  int
	NombreManche int
}

var (
	modeleHTML *template.Template

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	mu               sync.Mutex
	configuration    ParametresPartie
	lettreCourante   rune
	clients          = make(map[*websocket.Conn]*Joueur)
	remainingSeconds int
	roundActive      bool
	attenteRejouer   bool
	partieTerminee   bool
	manchesJouees    int
	compteurJoueur   int
)

func main() {
	var err error

	configuration = ParametresPartie{
		Categories:   ObtenirCategories(),
		TempsManche:  90,
		NombreManche: nbrs_manche,
	}
	lettreCourante = ObtenirLettreAleatoire()

	modeleHTML, err = template.ParseFiles("templates/ptitbac.html")
	if err != nil {
		log.Fatalf("Erreur de chargement du template : %v", err)
	}

	// ressources statiques
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", handlerPage)
	http.HandleFunc("/ws", handlerWebSocket)
	http.HandleFunc("/config", handlerConfiguration)

	// première manche automatique
	demarrerNouvelleMancheAvecSelection(false)

	log.Println("Serveur lancé sur http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Erreur serveur : %v", err)
	}
}

func handlerPage(w http.ResponseWriter, r *http.Request) {
	d := DonneesPage{
		Lettre:          string(lettreCourante),
		Categories:      append([]string(nil), configuration.Categories...),
		TempsParManche:  configuration.TempsManche,
		NombreDeManches: configuration.NombreManche,
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
	compteurJoueur++
	joueur := &Joueur{
		ID:         "joueur-" + strconv.Itoa(compteurJoueur),
		Nom:        "Anonyme",
		Score:      0,
		ScoreTotal: 0,
		Reponses:   make(map[string]string),
		Pret:       false,
		Actif:      roundActive && !partieTerminee,
	}
	clients[conn] = joueur
	mu.Unlock()

	// envoie l'identifiant au client
	if err := conn.WriteJSON(map[string]string{
		"type": "identity",
		"id":   joueur.ID,
	}); err != nil {
		log.Printf("Erreur envoi identité : %v", err)
	}

	diffuserEtat()

	go boucleLecture(conn)
}

type requeteConfiguration struct {
	Categories []string `json:"categories"`
	Temps      int      `json:"temps"`
	Manches    int      `json:"manches"`
}

func handlerConfiguration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Methode non supportee", http.StatusMethodNotAllowed)
		return
	}

	var payload requeteConfiguration
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "JSON invalide", http.StatusBadRequest)
		return
	}

	nouvellesCategories := make([]string, 0, len(payload.Categories))
	for _, cat := range payload.Categories {
		cat = strings.TrimSpace(cat)
		if cat != "" {
			nouvellesCategories = append(nouvellesCategories, cat)
		}
	}
	if len(nouvellesCategories) == 0 {
		nouvellesCategories = ObtenirCategories()
	}

	if payload.Temps < 15 {
		payload.Temps = 15
	}
	if payload.Manches <= 0 {
		payload.Manches = nbrs_manche
	}

	mu.Lock()
	configuration.Categories = nouvellesCategories
	configuration.TempsManche = payload.Temps
	configuration.NombreManche = payload.Manches
	roundActive = false
	attenteRejouer = false
	partieTerminee = false
	remainingSeconds = 0
	manchesJouees = 0
	lettreCourante = ObtenirLettreAleatoire()
	for _, joueur := range clients {
		joueur.Score = 0
		joueur.ScoreTotal = 0
		joueur.Reponses = make(map[string]string)
		joueur.Pret = false
		joueur.Actif = false
	}
	mu.Unlock()

	go demarrerNouvelleMancheAvecSelection(false)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func boucleLecture(conn *websocket.Conn) {
	defer func() {
		mu.Lock()
		delete(clients, conn)
		// si on était en attente de relance, on peut démarrer automatiquement
		tenterDemarrageApresVotesVerrouille()
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
			if strings.TrimSpace(msg.Nom) != "" {
				joueur.Nom = msg.Nom
			}
		case "answers":
			if !roundActive || !joueur.Actif {
				break
			}

			toutesRemplies := true
			if joueur.Reponses == nil {
				joueur.Reponses = make(map[string]string)
			}

			for _, cat := range configuration.Categories {
				rep := ""
				if msg.Reponses != nil {
					rep = msg.Reponses[cat]
				}
				joueur.Reponses[cat] = rep
				if strings.TrimSpace(rep) == "" {
					toutesRemplies = false
				}
			}

			if toutesRemplies {
				mu.Unlock()
				arreterMancheParCompletion()
				continue
			}
		case "ready":
			if !attenteRejouer || joueur.Pret {
				break
			}
			joueur.Pret = true
			if tenterDemarrageApresVotesVerrouille() {
				mu.Unlock()
				continue
			}
		}

		mu.Unlock()
		diffuserEtat()
	}
}

func demarrerNouvelleMancheAvecSelection(selectionParPrets bool) {
	mu.Lock()
	defer mu.Unlock()
	lancerNouvelleMancheVerrouille(selectionParPrets)
}

func lancerNouvelleMancheVerrouille(selectionParPrets bool) {
	if partieTerminee || (configuration.NombreManche > 0 && manchesJouees >= configuration.NombreManche) {
		terminerPartieVerrouille()
		return
	}

	actifs := 0
	for _, j := range clients {
		j.Score = 0
		j.Reponses = make(map[string]string)
		if selectionParPrets {
			j.Actif = j.Pret
		} else {
			j.Actif = true
		}
		if j.Actif {
			actifs++
		}
		j.Pret = false
	}

	if selectionParPrets && actifs == 0 {
		attenteRejouer = true
		remainingSeconds = 0
		return
	}

	manchesJouees++
	lettreCourante = ObtenirLettreAleatoire()
	remainingSeconds = 90
	roundActive = true
	attenteRejouer = false
	partieTerminee = false

	go diffuserEtat()
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

		if remainingSeconds == 0 {
			log.Println("Fin de manche par timer (0s)")
			roundActive = false
			attribuerScoresFinMancheVerrouille()
			passerEnModeAttenteVerrouille()
			mu.Unlock()
			diffuserEtat()
			return
		}
		mu.Unlock()

		diffuserEtat()
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
	remainingSeconds = 0
	attribuerScoresFinMancheVerrouille()
	passerEnModeAttenteVerrouille()
	go diffuserEtat()
}

func attribuerScoresFinMancheVerrouille() {
	if len(clients) == 0 {
		return
	}

	reponsesJoueurs := make([]map[string]string, 0, len(clients))
	ordre := make([]*Joueur, 0, len(clients))
	for _, joueur := range clients {
		if joueur.Reponses == nil {
			joueur.Reponses = make(map[string]string)
		}
		if joueur.Actif {
			reponsesJoueurs = append(reponsesJoueurs, joueur.Reponses)
			ordre = append(ordre, joueur)
		} else {
			joueur.Score = 0
		}
	}

	scores := CalculerScoresCollectifs(reponsesJoueurs, configuration.Categories, lettreCourante)
	for i, joueur := range ordre {
		joueur.Score = scores[i]
		joueur.ScoreTotal += scores[i]
	}
}

func passerEnModeAttenteVerrouille() {
	if configuration.NombreManche > 0 && manchesJouees >= configuration.NombreManche {
		terminerPartieVerrouille()
		return
	}
	attenteRejouer = true
	remainingSeconds = 0
	for _, joueur := range clients {
		joueur.Actif = false
		joueur.Pret = false
	}
}

func tenterDemarrageApresVotesVerrouille() bool {
	if !attenteRejouer || len(clients) == 0 || partieTerminee {
		return false
	}
	prets, total := compterJoueursPretsVerrouille()
	if prets == 0 {
		return false
	}
	if prets*3 > total {
		lancerNouvelleMancheVerrouille(true)
		return true
	}
	return false
}

func terminerPartieVerrouille() {
	roundActive = false
	attenteRejouer = false
	partieTerminee = true
	remainingSeconds = 0
	for _, joueur := range clients {
		joueur.Actif = false
		joueur.Pret = false
	}
}

func compterJoueursPretsVerrouille() (int, int) {
	total := len(clients)
	prets := 0
	for _, joueur := range clients {
		if joueur.Pret {
			prets++
		}
	}
	return prets, total
}

func diffuserEtat() {
	mu.Lock()
	etat := EtatPartie{
		Type:            "state",
		Lettre:          string(lettreCourante),
		Categories:      append([]string(nil), configuration.Categories...),
		RemainingSecond: remainingSeconds,
		RoundActive:     roundActive,
		WaitingRestart:  attenteRejouer,
		ManchesJouees:   manchesJouees,
		ManchesMaximum:  configuration.NombreManche,
		PartieTerminee:  partieTerminee,
	}
	prets, total := compterJoueursPretsVerrouille()
	etat.ReadyCount = prets
	etat.ReadyTotal = total

	joueurs := make([]Joueur, 0, len(clients))
	actifs := 0
	for _, j := range clients {
		if j.Actif {
			actifs++
		}
		joueurs = append(joueurs, *j)
	}
	etat.Joueurs = joueurs
	etat.ActivePlayersThisRound = actifs

	conns := make([]*websocket.Conn, 0, len(clients))
	for c := range clients {
		conns = append(conns, c)
	}
	mu.Unlock()

	for _, c := range conns {
		if err := c.WriteJSON(etat); err != nil {
			log.Printf("Erreur envoi WebSocket : %v", err)
		}
	}
}
