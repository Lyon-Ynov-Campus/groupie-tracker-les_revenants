package petitbac

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var (
	tplJeu   *template.Template
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func RegisterRoutes(authMiddleware func(http.HandlerFunc) http.HandlerFunc) error {
	var err error
	tplJeu, err = template.ParseFiles("PetitBac/templates/ptitbac.html")
	if err != nil {
		return fmt.Errorf("impossible de charger PetitBac/templates/ptitbac.html: %w", err)
	}

	http.HandleFunc("/PetitBac", authMiddleware(pageJeu))
	http.HandleFunc("/ws", socketJeu)
	http.HandleFunc("/config", configJeu)
	registerSalonHandlers(authMiddleware)

	fsJeu := http.FileServer(http.Dir("PetitBac/Pstatic"))
	http.Handle("/Pstatic/", http.StripPrefix("/Pstatic/", fsJeu))

	salons.defaultSalon().demarrerManche(false)
	return nil
}

func pageJeu(w http.ResponseWriter, r *http.Request) {
	s, err := salonFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tplJeu.Execute(w, s.templateData()); err != nil {
		log.Println("Erreur affichage jeu:", err)
	}
}

func configJeu(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "NO", http.StatusMethodNotAllowed)
		return
	}

	s, err := salonFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var reg reglageJeu
	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		http.Error(w, "invalid config", http.StatusBadRequest)
		return
	}

	s.applyConfig(reg)
	s.demarrerManche(false)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func socketJeu(w http.ResponseWriter, r *http.Request) {
	s, err := salonFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	joueur, joinErr := s.addPlayer(conn)
	if joinErr != nil {
		conn.WriteJSON(map[string]string{"type": "error", "message": joinErr.Error()})
		conn.Close()
		return
	}

	conn.WriteJSON(map[string]string{"type": "identity", "id": joueur.ID, "room": s.code})
	s.envoyerEtat()
	go s.boucleWS(conn)
}

func salonFromRequest(r *http.Request) (*salon, error) {
	code := normalizeSalonCode(r.URL.Query().Get("room"))
	if code == "" {
		return salons.defaultSalon(), nil
	}
	if s, ok := salons.getSalon(code); ok {
		return s, nil
	}
	return nil, fmt.Errorf("salon %s introuvable", code)
}

func (s *salon) templateData() donneesPage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return donneesPage{
		Lettre:          string(s.lettreActu),
		Categories:      append([]string(nil), s.reglages.Categories...),
		TempsParManche:  s.reglages.Temps,
		NombreDeManches: s.reglages.Manches,
		SalonCode:       s.code,
	}
}

func (s *salon) applyConfig(reg reglageJeu) {
	s.mu.Lock()
	if len(reg.Categories) > 0 {
		s.reglages.Categories = reg.Categories
	}
	if reg.Temps >= 15 {
		s.reglages.Temps = reg.Temps
	}
	if reg.Manches > 0 {
		s.reglages.Manches = reg.Manches
	}
	s.mancheEnCours, s.attenteVotes, s.termine = false, false, false
	s.nbManches, s.tempsRest = 0, 0
	s.lettreActu = lettreAleatoire()
	for _, j := range s.joueurs {
		j.Score, j.Total, j.Actif, j.Pret = 0, 0, false, false
		j.Reponses = make(map[string]string)
	}
	s.mu.Unlock()
}

func (s *salon) boucleWS(conn *websocket.Conn) {
	defer func() {
		s.removePlayer(conn)
		conn.Close()
		s.envoyerEtat()
	}()

	for {
		var msg messageJeu
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}

		s.mu.Lock()
		j, ok := s.joueurs[conn]
		if !ok {
			s.mu.Unlock()
			return
		}
		if msg.Type == "join" {
			if n := strings.TrimSpace(msg.Nom); n != "" {
				j.Nom = n
			}
		} else if msg.Type == "answers" && s.mancheEnCours && j.Actif {
			complet := true
			for _, cat := range s.reglages.Categories {
				val := msg.Reponses[cat]
				j.Reponses[cat] = val
				if strings.TrimSpace(val) == "" {
					complet = false
				}
			}
			if complet {
				s.mu.Unlock()
				s.finMancheRemplie()
				continue
			}
		} else if msg.Type == "ready" && s.attenteVotes && !j.Pret {
			j.Pret = true
			s.mu.Unlock()
			if s.verifieVotes() {
				continue
			}
			s.envoyerEtat()
			continue
		}
		s.mu.Unlock()
		s.envoyerEtat()
	}
}

func (s *salon) demarrerManche(selection bool) {
	s.mu.Lock()
	if s.termine || (s.reglages.Manches > 0 && s.nbManches >= s.reglages.Manches) {
		s.finPartieLocked()
		s.mu.Unlock()
		s.envoyerEtat()
		return
	}

	actifs := 0
	for _, j := range s.joueurs {
		j.Score = 0
		j.Reponses = make(map[string]string)
		j.Actif = !selection || j.Pret
		if j.Actif {
			actifs++
		}
		j.Pret = false
	}

	if selection && actifs == 0 {
		s.attenteVotes = true
		s.tempsRest = 0
		s.mu.Unlock()
		s.envoyerEtat()
		return
	}

	s.nbManches++
	if s.reglages.Temps <= 0 {
		s.reglages.Temps = 90
	}
	s.lettreActu = lettreAleatoire()
	s.tempsRest = s.reglages.Temps
	s.mancheEnCours, s.attenteVotes, s.termine = true, false, false
	s.mu.Unlock()

	go s.compteRebours()
	s.envoyerEtat()
}

func (s *salon) compteRebours() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		s.mu.Lock()
		if !s.mancheEnCours {
			s.mu.Unlock()
			return
		}
		if s.tempsRest > 0 {
			s.tempsRest--
		}
		if s.tempsRest == 0 {
			s.mancheEnCours = false
			s.mu.Unlock()
			s.scoresFin()
			s.modeAttente()
			s.envoyerEtat()
			return
		}
		s.mu.Unlock()
		s.envoyerEtat()
	}
}

func (s *salon) finMancheRemplie() {
	s.mu.Lock()
	if !s.mancheEnCours {
		s.mu.Unlock()
		return
	}
	s.mancheEnCours = false
	s.tempsRest = 0
	s.mu.Unlock()
	s.scoresFin()
	s.modeAttente()
	s.envoyerEtat()
}

func (s *salon) scoresFin() {
	s.mu.Lock()
	if len(s.joueurs) == 0 {
		s.mu.Unlock()
		return
	}
	tous := []map[string]string{}
	ordre := []*joueurDonnees{}
	for _, j := range s.joueurs {
		if j.Actif {
			tous = append(tous, j.Reponses)
			ordre = append(ordre, j)
		} else {
			j.Score = 0
		}
	}
	cats := append([]string(nil), s.reglages.Categories...)
	lettre := s.lettreActu
	s.mu.Unlock()

	points := scoresCollectifs(tous, cats, lettre)

	s.mu.Lock()
	for i, j := range ordre {
		if i < len(points) {
			j.Score = points[i]
			j.Total += points[i]
		}
	}
	s.mu.Unlock()
}

func (s *salon) modeAttente() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reglages.Manches > 0 && s.nbManches >= s.reglages.Manches {
		s.finPartieLocked()
		return
	}
	s.attenteVotes = true
	s.tempsRest = 0
	for _, j := range s.joueurs {
		j.Actif = false
		j.Pret = false
	}
}

func (s *salon) verifieVotes() bool {
	s.mu.Lock()
	if !s.attenteVotes || len(s.joueurs) == 0 || s.termine {
		s.mu.Unlock()
		return false
	}
	prets := 0
	for _, j := range s.joueurs {
		if j.Pret {
			prets++
		}
	}
	total := len(s.joueurs)
	s.mu.Unlock()

	if prets > 0 && float64(prets) >= float64(total)*0.66 {
		s.demarrerManche(true)
		return true
	}
	return false
}

func (s *salon) finPartieLocked() {
	s.mancheEnCours, s.attenteVotes, s.termine = false, false, true
	s.tempsRest = 0
	for _, j := range s.joueurs {
		j.Actif, j.Pret = false, false
	}
}

func (s *salon) envoyerEtat() {
	s.mu.Lock()
	etat := paquetEtat{
		Type:           "state",
		Lettre:         string(s.lettreActu),
		Categories:     append([]string(nil), s.reglages.Categories...),
		Secondes:       s.tempsRest,
		MancheActive:   s.mancheEnCours,
		Attente:        s.attenteVotes,
		NumeroManche:   s.nbManches,
		LimiteManches:  s.reglages.Manches,
		JeuTermine:     s.termine,
		TempsParManche: s.reglages.Temps,
	}
	jListe := make([]joueurDonnees, 0, len(s.joueurs))
	dest := make([]*websocket.Conn, 0, len(s.joueurs))
	for c, j := range s.joueurs {
		dest = append(dest, c)
		jListe = append(jListe, *j)
		if j.Actif {
			etat.Actifs++
		}
		if j.Pret {
			etat.CompteurPrets++
		}
	}
	etat.Joueurs = jListe
	etat.CompteurTotal = len(s.joueurs)
	s.mu.Unlock()

	for _, c := range dest {
		c.WriteJSON(etat)
	}
}

func listeCategories() []string {
	res := []string{}
	res = append(res, "Artiste")
	res = append(res, "Album")
	res = append(res, "Groupe de musique")
	res = append(res, "Instrument de musique")
	res = append(res, "Featuring")
	return res
}

func lettreAleatoire() rune {
	n := rand.Intn(26)
	return rune('A' + n)
}

func compteReponses(reponses map[string]string, categories []string) int {
	score := 0
	for i := 0; i < len(categories); i++ {
		cat := categories[i]
		texte := strings.TrimSpace(reponses[cat])
		if texte != "" {
			score++
		}
	}
	return score
}

func reponseValide(texte string, lettre rune) bool {
	texte = strings.TrimSpace(texte)
	if texte == "" {
		return false
	}
	texte = strings.ToUpper(texte)
	valeurs := []rune(texte)
	if len(valeurs) == 0 {
		return false
	}
	return valeurs[0] == lettre
}

func scoresCollectifs(reponsesJoueurs []map[string]string, categories []string, lettre rune) []int {
	resultats := make([]int, len(reponsesJoueurs))
	if len(reponsesJoueurs) == 0 {
		return resultats
	}

	seuil := (2*len(reponsesJoueurs) + 2) / 3

	for _, cat := range categories {
		occur := make(map[string]int)
		ok := make([]bool, len(reponsesJoueurs))
		objets := make([]string, len(reponsesJoueurs))

		for i := 0; i < len(reponsesJoueurs); i++ {
			reponses := reponsesJoueurs[i]
			texte := reponses[cat]
			if reponseValide(texte, lettre) {
				forme := strings.ToUpper(strings.TrimSpace(texte))
				occur[forme]++
				ok[i] = true
				objets[i] = forme
			}
		}

		for i := 0; i < len(reponsesJoueurs); i++ {
			if !ok[i] {
				continue
			}
			if occur[objets[i]] < seuil {
				continue
			}
			if occur[objets[i]] == 1 {
				resultats[i] += 2
			} else {
				resultats[i]++
			}
		}
	}

	return resultats
}
