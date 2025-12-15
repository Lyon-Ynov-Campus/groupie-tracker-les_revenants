package petitbac

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type categoriesPageData struct {
	Selected map[string]bool
	Custom   string
	Error    string
}

type timePageData struct {
	Categories []string
	Duration   int
	Rounds     int
	Error      string
}

type joinPageData struct {
	Code  string
	Error string
}

type waitingPageData struct {
	PageData
	JoueursAttente []dbPlayer
}

func pagePetitBacHome(w http.ResponseWriter, _ *http.Request) {
	renderStaticPage(w, tplHome, nil)
}

func pageCreateCategories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		selected := sanitizeCategories(r.URL.Query()["cats"])
		data := buildCategoriesPageData(selected, r.URL.Query().Get("custom"), "")
		renderStaticPage(w, tplCreateCategories, data)
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "formulaire invalide", http.StatusBadRequest)
			return
		}
		selected := sanitizeCategories(r.PostForm["categories"])
		custom := strings.TrimSpace(r.FormValue("custom"))
		if custom != "" {
			selected = append(selected, custom)
		}
		selected = sanitizeCategories(selected)
		if len(selected) == 0 {
			data := buildCategoriesPageData(selected, custom, "Merci de choisir au moins une categorie.")
			renderStaticPage(w, tplCreateCategories, data)
			return
		}
		vals := url.Values{}
		for _, c := range selected {
			vals.Add("cats", c)
		}
		http.Redirect(w, r, "/PetitBac/create/time?"+vals.Encode(), http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func pageCreateTime(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cats := sanitizeCategories(r.URL.Query()["cats"])
		if len(cats) == 0 {
			http.Redirect(w, r, "/PetitBac/create/categories", http.StatusSeeOther)
			return
		}
		data := timePageData{
			Categories: cats,
			Duration:   60,
			Rounds:     5,
		}
		renderStaticPage(w, tplCreateTime, data)
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "formulaire invalide", http.StatusBadRequest)
			return
		}
		cats := sanitizeCategories(r.PostForm["categories"])
		if len(cats) == 0 {
			data := timePageData{
				Categories: cats,
				Duration:   60,
				Rounds:     5,
				Error:      "Selectionne au moins une categorie.",
			}
			renderStaticPage(w, tplCreateTime, data)
			return
		}
		duration := clampTemps(parseIntOrDefault(r.FormValue("duration"), 60))
		rounds := clampRounds(parseIntOrDefault(r.FormValue("rounds"), 5))
		reg := GameConfig{
			Categories: cats,
			Temps:      duration,
			Manches:    rounds,
		}
		room := createConfiguredRoom(reg, currentUserPseudo(r))
		http.Redirect(w, r, "/PetitBac/wait?room="+url.QueryEscape(room.code), http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func pageJoinSalon(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := joinPageData{Code: normalizeRoomCode(r.URL.Query().Get("code"))}
		renderStaticPage(w, tplJoinRoom, data)
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "formulaire invalide", http.StatusBadRequest)
			return
		}
		code := normalizeRoomCode(r.FormValue("room"))
		if code == "" {
			renderStaticPage(w, tplJoinRoom, joinPageData{Error: "Merci de saisir un code valide."})
			return
		}
		s, err := getRoomForJoin(code)
		if err != nil {
			renderStaticPage(w, tplJoinRoom, joinPageData{Error: err.Error(), Code: code})
			return
		}
		if !s.hasRoom() {
			renderStaticPage(w, tplJoinRoom, joinPageData{Error: "Salon complet pour le moment.", Code: code})
			return
		}
		http.Redirect(w, r, "/PetitBac/wait?room="+url.QueryEscape(code), http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func pageWaitingRoom(w http.ResponseWriter, r *http.Request) {
	s, err := roomFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	players, err := fetchRoomPlayers(s.code)
	if err != nil {
		log.Println("PetitBac: impossible de charger les joueurs:", err)
	}
	data := waitingPageData{
		PageData:       s.templateData(),
		JoueursAttente: players,
	}
	renderStaticPage(w, tplWaiting, data)
}

func renderStaticPage(w http.ResponseWriter, tpl *template.Template, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.Execute(w, data); err != nil {
		log.Println("Erreur affichage template Petit Bac:", err)
	}
}

func pageJeu(w http.ResponseWriter, r *http.Request) {
	room, err := roomFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	renderStaticPage(w, tplJeu, room.templateData())
}

func configJeu(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "NO", http.StatusMethodNotAllowed)
		return
	}

	room, err := roomFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var reg GameConfig
	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		http.Error(w, "invalid config", http.StatusBadRequest)
		return
	}

	room.applyConfig(reg)
	room.demarrerManche(false)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func socketJeu(w http.ResponseWriter, r *http.Request) {
	room, err := roomFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	joueur, joinErr := room.addPlayer(conn)
	if joinErr != nil {
		conn.WriteJSON(map[string]string{"type": "error", "message": joinErr.Error()})
		conn.Close()
		return
	}

	conn.WriteJSON(map[string]string{"type": "identity", "id": joueur.ID, "room": room.code})
	room.envoyerEtat()
	go room.boucleWS(conn)
}

func roomFromRequest(r *http.Request) (*Room, error) {
	code := normalizeRoomCode(r.URL.Query().Get("room"))
	if code == "" {
		return defaultRoom(), nil
	}
	if room, ok := getRoom(code); ok {
		return room, nil
	}
	return nil, fmt.Errorf("salon %s introuvable", code)
}

func buildCategoriesPageData(selected []string, custom, errMsg string) categoriesPageData {
	data := categoriesPageData{
		Selected: make(map[string]bool),
		Custom:   custom,
		Error:    errMsg,
	}
	if len(selected) == 0 {
		selected = listeCategories()
	}
	for _, c := range selected {
		data.Selected[c] = true
	}
	return data
}

func parseIntOrDefault(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return v
}

func currentUserPseudo(r *http.Request) string {
	if userResolver == nil {
		return ""
	}
	user, err := userResolver(r)
	if err != nil || user == nil {
		return ""
	}
	return strings.TrimSpace(user.Pseudo)
}
