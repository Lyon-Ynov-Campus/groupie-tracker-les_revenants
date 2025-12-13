package petitbac

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const (
	defaultSalonCode = "CLASSIC"
	salonCodeLength  = 5
	maxSalonPlayers  = 5
)

var salons = newSalonManager()

func init() {
	rand.Seed(time.Now().UnixNano())
}

func newSalonManager() *salonManager {
	m := &salonManager{
		salons: make(map[string]*salon),
	}
	m.salons[defaultSalonCode] = newSalon(defaultSalonCode)
	return m
}

func registerSalonHandlers(authMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	http.HandleFunc("/PetitBac/salons", authMiddleware(handleSalonCreate))
	http.HandleFunc("/PetitBac/salons/join", authMiddleware(handleSalonJoin))
}

func handleSalonCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Categories []string `json:"categories"`
		Temps      int      `json:"temps"`
		Manches    int      `json:"manches"`
		Host       string   `json:"host"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	reg := reglageJeu{
		Categories: sanitizeCategories(payload.Categories),
		Temps:      clampTemps(payload.Temps),
		Manches:    clampRounds(payload.Manches),
	}
<<<<<<< HEAD
	s := salons.createSalon()
	s.applyConfig(reg)
	persistRoomConfiguration(s.code, reg, payload.Host)
=======
	s := createConfiguredSalon(reg, payload.Host)
>>>>>>> v1seb
	respondJSON(w, map[string]string{"code": s.code})
}

func handleSalonJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	s, err := salons.getSalonForJoin(payload.Code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if !s.hasRoom() {
		http.Error(w, "salon plein (5 joueurs maximum)", http.StatusConflict)
		return
	}
	respondJSON(w, map[string]string{"status": "ok", "code": s.code})
}

func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func handleRoomPlayers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	code := normalizeSalonCode(r.URL.Query().Get("room"))
	if code == "" {
		http.Error(w, "code manquant", http.StatusBadRequest)
		return
	}
	players, err := fetchRoomPlayers(code)
	if err != nil {
		http.Error(w, "erreur base", http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]interface{}{"players": players})
}

func handleStartGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Code string `json:"code"`
		Host string `json:"host"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	s, err := salons.getSalonForJoin(payload.Code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if payload.Host != "" && !isRoomHost(s.code, payload.Host) {
<<<<<<< HEAD
		http.Error(w, "action reservee a l'hote", http.StatusForbidden)
=======
		http.Error(w, "action réservée à l'hôte", http.StatusForbidden)
>>>>>>> v1seb
		return
	}
	s.demarrerManche(false)
	respondJSON(w, map[string]string{"status": "started"})
}

func (m *salonManager) createSalon() *salon {
	m.mu.Lock()
	defer m.mu.Unlock()
	code := m.generateCodeLocked()
	s := newSalon(code)
	m.salons[code] = s
	return s
}

func (m *salonManager) defaultSalon() *salon {
	m.mu.RLock()
	s, ok := m.salons[defaultSalonCode]
	m.mu.RUnlock()
	if ok {
		return s
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.salons[defaultSalonCode]; ok {
		return existing
	}
	s = newSalon(defaultSalonCode)
	m.salons[defaultSalonCode] = s
	return s
}

func (m *salonManager) getSalon(code string) (*salon, bool) {
	code = normalizeSalonCode(code)
	if code == "" {
		return m.defaultSalon(), true
	}
	m.mu.RLock()
	s, ok := m.salons[code]
	m.mu.RUnlock()
	return s, ok
}

func (m *salonManager) getSalonForJoin(code string) (*salon, error) {
	code = normalizeSalonCode(code)
	if code == "" {
		return nil, errors.New("code requis")
	}
	m.mu.RLock()
	s, ok := m.salons[code]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("salon %s introuvable", code)
	}
	return s, nil
}

func (m *salonManager) generateCodeLocked() string {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for {
		builder := strings.Builder{}
		for i := 0; i < salonCodeLength; i++ {
			builder.WriteByte(letters[rand.Intn(len(letters))])
		}
		code := builder.String()
		if _, exists := m.salons[code]; !exists {
			return code
		}
	}
}

<<<<<<< HEAD
func sanitizeCategories(cats []string) []string {
	seen := make(map[string]struct{})
	var cleaned []string
=======
func createConfiguredSalon(reg reglageJeu, host string) *salon {
	s := salons.createSalon()
	s.applyConfig(reg)
	persistRoomConfiguration(s.code, reg, host)
	return s
}

func sanitizeCategories(cats []string) []string {
	seen := make(map[string]struct{})
	res := []string{}
>>>>>>> v1seb
	for _, c := range cats {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
<<<<<<< HEAD
		upper := strings.ToUpper(c)
		if _, ok := seen[upper]; ok {
			continue
		}
		seen[upper] = struct{}{}
		cleaned = append(cleaned, c)
	}
	if len(cleaned) == 0 {
		return listeCategories()
	}
	return cleaned
=======
		key := strings.ToUpper(c)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		res = append(res, c)
	}
	if len(res) == 0 {
		return listeCategories()
	}
	return res
>>>>>>> v1seb
}

func clampTemps(v int) int {
	if v < 30 {
		return 30
	}
	if v > 180 {
		return 180
	}
	return v
}

func clampRounds(v int) int {
	if v < 3 {
		return 3
	}
	if v > 10 {
		return 10
	}
	return v
}
