package petitbac

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
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
	s := salons.createSalon()
	s.demarrerManche(false)
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

func newSalon(code string) *salon {
	return &salon{
		code:       normalizeSalonCode(code),
		reglages:   reglageJeu{Categories: listeCategories(), Temps: 90, Manches: 5},
		lettreActu: lettreAleatoire(),
		joueurs:    make(map[*websocket.Conn]*joueurDonnees),
	}
}

func normalizeSalonCode(code string) string {
	code = strings.TrimSpace(strings.ToUpper(code))
	return code
}

func (s *salon) hasRoom() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.joueurs) < maxSalonPlayers
}

func (s *salon) playerCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.joueurs)
}

func (s *salon) addPlayer(conn *websocket.Conn) (*joueurDonnees, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.joueurs) >= maxSalonPlayers {
		return nil, fmt.Errorf("salon plein (max %d joueurs)", maxSalonPlayers)
	}
	s.compteurJoueurs++
	j := &joueurDonnees{
		ID:       fmt.Sprintf("j-%s-%d", strings.ToLower(s.code), s.compteurJoueurs),
		Nom:      "Anonyme",
		Reponses: make(map[string]string),
		Actif:    s.mancheEnCours && !s.termine,
	}
	s.joueurs[conn] = j
	return j, nil
}

func (s *salon) removePlayer(conn *websocket.Conn) {
	s.mu.Lock()
	delete(s.joueurs, conn)
	s.mu.Unlock()
}
