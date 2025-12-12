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

	blindtest "groupie-tracker/BlindTest"
)

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

func main() {

	initDatabase()

	reglages = reglageJeu{
		Categories: listeCategories(),
		Temps:      90,
		Manches:    5,
	}
	lettreActu = lettreAleatoire()

	var err error
	tplJeu, err = template.ParseFiles("PetitBac/templates/ptitbac.html")
	if err != nil {
		log.Fatal("ERREUR : Impossible de trouver PetitBac/templates/ptitbac.html.", err)
	}

	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	fsJeu := http.FileServer(http.Dir("PetitBac/Pstatic"))
	http.Handle("/Pstatic/", http.StripPrefix("/Pstatic/", fsJeu))

	http.HandleFunc("/", pageAccueil)
	http.HandleFunc("/login", pageLogin)
	http.HandleFunc("/register", pageRegister)
	http.HandleFunc("/logout", pageLogout)
	http.HandleFunc("/api/user", apiUserInfo)
	http.HandleFunc("/PetitBac", requireAuth(pageJeu))
	http.HandleFunc("/ws", socketJeu)
	http.HandleFunc("/config", configJeu)

	blindtest.RegisterRoutes(requireAuth)

	demarrerManche(false)

	log.Println("SERVEUR PRET")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func pageAccueil(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	tmpl, err := template.ParseFiles("web/index.html")
	if err != nil {
		log.Println(err)
		return
	}
	tmpl.Execute(w, nil)
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
		log.Println("Erreur affichage jeu:", err)
	}
}

func configJeu(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "NO", 405)
		return
	}
	var reg reglageJeu
	json.NewDecoder(r.Body).Decode(&reg)

	mu.Lock()
	if len(reg.Categories) > 0 {
		reglages.Categories = reg.Categories
	}
	if reg.Temps >= 15 {
		reglages.Temps = reg.Temps
	}
	if reg.Manches > 0 {
		reglages.Manches = reg.Manches
	}

	mancheEnCours, attenteVotes, termine = false, false, false
	nbManches, tempsRest = 0, 0
	lettreActu = lettreAleatoire()
	for _, j := range joueurs {
		j.Score, j.Total, j.Actif, j.Pret = 0, 0, false, false
		j.Reponses = make(map[string]string)
	}
	mu.Unlock()
	demarrerManche(false)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func socketJeu(w http.ResponseWriter, r *http.Request) {
	conn, err := monterWS.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	mu.Lock()
	compteurJoueurs++
	j := &joueurDonnees{
		ID: "j-" + strconv.Itoa(compteurJoueurs), Nom: "Anonyme",
		Reponses: make(map[string]string), Actif: mancheEnCours && !termine,
	}
	joueurs[conn] = j
	mu.Unlock()
	conn.WriteJSON(map[string]string{"type": "identity", "id": j.ID})
	envoyerEtat()
	go boucleWS(conn)
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
			if n := strings.TrimSpace(msg.Nom); n != "" {
				j.Nom = n
			}
		} else if msg.Type == "answers" && mancheEnCours && j.Actif {
			complet := true
			for _, cat := range reglages.Categories {
				val := msg.Reponses[cat]
				j.Reponses[cat] = val
				if strings.TrimSpace(val) == "" {
					complet = false
				}
			}
			if complet {
				mu.Unlock()
				finMancheRemplie()
				continue
			}
		} else if msg.Type == "ready" && attenteVotes && !j.Pret {
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
		j.Actif = !selection || j.Pret
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
	mancheEnCours, attenteVotes, termine = true, false, false
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
	if prets > 0 && float64(prets) >= float64(total)*0.66 {
		demarrerManche(true)
		return true
	}
	return false
}

func finPartie() {
	mancheEnCours, attenteVotes, termine = false, false, true
	tempsRest = 0
	for _, j := range joueurs {
		j.Actif, j.Pret = false, false
	}
}

func compterPrets() (int, int) {
	prets := 0
	for _, j := range joueurs {
		if j.Pret {
			prets++
		}
	}
	return prets, len(joueurs)
}

func envoyerEtat() {
	mu.Lock()
	etat := paquetEtat{
		Type: "state", Lettre: string(lettreActu), Categories: reglages.Categories,
		Secondes: tempsRest, MancheActive: mancheEnCours, Attente: attenteVotes,
		NumeroManche: nbManches, LimiteManches: reglages.Manches, JeuTermine: termine,
		TempsParManche: reglages.Temps,
	}
	etat.CompteurPrets, etat.CompteurTotal = compterPrets()
	etat.Actifs = 0
	jListe := []joueurDonnees{}
	dest := []*websocket.Conn{}
	for c, j := range joueurs {
		dest = append(dest, c)
		jListe = append(jListe, *j)
		if j.Actif {
			etat.Actifs++
		}
	}
	etat.Joueurs = jListe
	mu.Unlock()
	for _, c := range dest {
		c.WriteJSON(etat)
	}
}
