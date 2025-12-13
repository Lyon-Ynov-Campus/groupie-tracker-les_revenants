package petitbac

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

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
