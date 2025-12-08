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
}

type reglageJeu struct {
	Categories []string `json:"categories"`
	Temps      int      `json:"temps"`
	Manches    int      `json:"manches"`
}

var (
	tplJeu          *template.Template
	monterWS        = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	mu              sync.Mutex
	reglages        reglageJeu
	lettreActu      rune
	joueurs         = make(map[*websocket.Conn]*joueurDonnees)
	tempsRest       int
	mancheEnCours   bool
	attenteVotes    bool
	termine         bool
	nbManches       int
	compteurJoueurs int
)

func main() {
	reglages = reglageJeu{
		Categories: listeCategories(),
		Temps:      90,
		Manches:    5,
	}
	lettreActu = lettreAleatoire()

	var err error
	tplJeu, err = template.ParseFiles("templates/ptitbac.html")
	if err != nil {
		log.Fatalf("Erreur template %v", err)
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", pageJeu)
	http.HandleFunc("/ws", socketJeu)
	http.HandleFunc("/config", configJeu)

	demarrerManche(false)

	log.Println("Serveur lance sur http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Erreur serveur %v", err)
	}
}

func pageJeu(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := donneesPage{
		Lettre:          string(lettreActu),
		Categories:      append([]string(nil), reglages.Categories...),
		TempsParManche:  reglages.Temps,
		NombreDeManches: reglages.Manches,
	}
	if err := tplJeu.Execute(w, data); err != nil {
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
	}
}

func socketJeu(w http.ResponseWriter, r *http.Request) {
	conn, err := monterWS.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Erreur WS %v", err)
		return
	}

	mu.Lock()
	compteurJoueurs++
	j := &joueurDonnees{
		ID:       "joueur-" + strconv.Itoa(compteurJoueurs),
		Nom:      "Anonyme",
		Reponses: make(map[string]string),
		Actif:    mancheEnCours && !termine,
	}
	joueurs[conn] = j
	mu.Unlock()

	_ = conn.WriteJSON(map[string]string{"type": "identity", "id": j.ID})
	envoyerEtat()
	go boucleWS(conn)
}

func configJeu(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Methode refusee", http.StatusMethodNotAllowed)
		return
	}
	var reg reglageJeu
	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		http.Error(w, "JSON invalide", http.StatusBadRequest)
		return
	}

	nouvelles := []string{}
	for _, cat := range reg.Categories {
		cat = strings.TrimSpace(cat)
		if cat != "" {
			nouvelles = append(nouvelles, cat)
		}
	}
	if len(nouvelles) == 0 {
		nouvelles = listeCategories()
	}
	if reg.Temps < 15 {
		reg.Temps = 15
	}
	if reg.Manches <= 0 {
		if reglages.Manches > 0 {
			reg.Manches = reglages.Manches
		} else {
			reg.Manches = 5
		}
	}

	mu.Lock()
	reglages = reg
	mancheEnCours = false
	attenteVotes = false
	termine = false
	tempsRest = 0
	nbManches = 0
	lettreActu = lettreAleatoire()
	for _, joueur := range joueurs {
		joueur.Score = 0
		joueur.Total = 0
		joueur.Reponses = make(map[string]string)
		joueur.Pret = false
		joueur.Actif = false
	}
	mu.Unlock()

	demarrerManche(false)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func boucleWS(conn *websocket.Conn) {
	defer func() {
		mu.Lock()
		delete(joueurs, conn)
		mu.Unlock()
		conn.Close()
		envoyerEtat()
	}()

	for {
		var msg messageJeu
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}

		mu.Lock()
		j, ok := joueurs[conn]
		if !ok {
			mu.Unlock()
			return
		}

		if msg.Type == "join" {
			if nom := strings.TrimSpace(msg.Nom); nom != "" {
				j.Nom = nom
			}
		}

		if msg.Type == "answers" && mancheEnCours && j.Actif {
			complet := true
			for _, cat := range reglages.Categories {
				valeur := ""
				if msg.Reponses != nil {
					valeur = msg.Reponses[cat]
				}
				j.Reponses[cat] = valeur
				if strings.TrimSpace(valeur) == "" {
					complet = false
				}
			}
			if complet {
				mu.Unlock()
				finMancheRemplie()
				continue
			}
		}

		if msg.Type == "ready" && attenteVotes && !j.Pret {
			j.Pret = true
			if verifieVotes() {
				mu.Unlock()
				continue
			}
		}

		mu.Unlock()
		envoyerEtat()
	}
}

func demarrerManche(selection bool) {
	mu.Lock()
	if termine || (reglages.Manches > 0 && nbManches >= reglages.Manches) {
		finPartie()
		mu.Unlock()
		return
	}

	actifs := 0
	for _, j := range joueurs {
		j.Score = 0
		j.Reponses = make(map[string]string)
		if selection {
			j.Actif = j.Pret
		} else {
			j.Actif = true
		}
		if j.Actif {
			actifs++
		}
		j.Pret = false
	}
	if selection && actifs == 0 {
		attenteVotes = true
		tempsRest = 0
		mu.Unlock()
		return
	}

	nbManches++
	if reglages.Temps <= 0 {
		reglages.Temps = 90
	}
	lettreActu = lettreAleatoire()
	tempsRest = reglages.Temps
	mancheEnCours = true
	attenteVotes = false
	termine = false
	mu.Unlock()

	go compteRebours()
	envoyerEtat()
}

func compteRebours() {
	t := time.NewTicker(time.Second)
	defer t.Stop()

	for range t.C {
		mu.Lock()
		if !mancheEnCours {
			mu.Unlock()
			return
		}
		if tempsRest > 0 {
			tempsRest--
		}
		if tempsRest == 0 {
			mancheEnCours = false
			scoresFin()
			modeAttente()
			mu.Unlock()
			envoyerEtat()
			return
		}
		mu.Unlock()
		envoyerEtat()
	}
}

func finMancheRemplie() {
	mu.Lock()
	if !mancheEnCours {
		mu.Unlock()
		return
	}
	mancheEnCours = false
	tempsRest = 0
	scoresFin()
	modeAttente()
	mu.Unlock()
	envoyerEtat()
}

func scoresFin() {
	if len(joueurs) == 0 {
		return
	}
	tous := []map[string]string{}
	ordre := []*joueurDonnees{}
	for _, j := range joueurs {
		if j.Reponses == nil {
			j.Reponses = make(map[string]string)
		}
		if j.Actif {
			tous = append(tous, j.Reponses)
			ordre = append(ordre, j)
		} else {
			j.Score = 0
		}
	}
	points := scoresCollectifs(tous, reglages.Categories, lettreActu)
	for i, j := range ordre {
		j.Score = points[i]
		j.Total += points[i]
	}
}

func modeAttente() {
	if reglages.Manches > 0 && nbManches >= reglages.Manches {
		finPartie()
		return
	}
	attenteVotes = true
	tempsRest = 0
	for _, j := range joueurs {
		j.Actif = false
		j.Pret = false
	}
}

func verifieVotes() bool {
	if !attenteVotes || len(joueurs) == 0 || termine {
		return false
	}
	prets, total := compterPrets()
	if prets == 0 {
		return false
	}
	if prets*3 > total {
		demarrerManche(true)
		return true
	}
	return false
}

func finPartie() {
	mancheEnCours = false
	attenteVotes = false
	termine = true
	tempsRest = 0
	for _, j := range joueurs {
		j.Actif = false
		j.Pret = false
	}
}

func compterPrets() (int, int) {
	prets := 0
	total := len(joueurs)
	for _, j := range joueurs {
		if j.Pret {
			prets++
		}
	}
	return prets, total
}

func envoyerEtat() {
	mu.Lock()
	etat := paquetEtat{
		Type:           "state",
		Lettre:         string(lettreActu),
		Categories:     append([]string(nil), reglages.Categories...),
		Secondes:       tempsRest,
		MancheActive:   mancheEnCours,
		Attente:        attenteVotes,
		NumeroManche:   nbManches,
		LimiteManches:  reglages.Manches,
		JeuTermine:     termine,
		TempsParManche: reglages.Temps,
	}
	prets, total := compterPrets()
	etat.CompteurPrets = prets
	etat.CompteurTotal = total

	if mancheEnCours && total == 0 {
		etat.MancheActive = false
	}

	jListe := make([]joueurDonnees, 0, len(joueurs))
	actifs := 0
	for _, j := range joueurs {
		if j.Actif {
			actifs++
		}
		jListe = append(jListe, *j)
	}
	etat.Joueurs = jListe
	etat.Actifs = actifs

	dest := make([]*websocket.Conn, 0, len(joueurs))
	for conn := range joueurs {
		dest = append(dest, conn)
	}
	mu.Unlock()

	for _, conn := range dest {
		_ = conn.WriteJSON(etat)
	}
}
