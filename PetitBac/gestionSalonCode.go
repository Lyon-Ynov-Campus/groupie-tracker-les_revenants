package petitbac

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const (
	defaultRoomCode = "CLASSIC"
	roomCodeLength  = 5
	maxSalonPlayers = 5
)

func init() {
	rand.Seed(time.Now().UnixNano())
	ensureDefaultRoom()
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
	reg := GameConfig{
		Categories: sanitizeCategories(payload.Categories),
		Temps:      clampTemps(payload.Temps),
		Manches:    clampRounds(payload.Manches),
	}

	room := createConfiguredRoom(reg, payload.Host)
	respondJSON(w, map[string]string{"code": room.code})
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

	room, err := getRoomForJoin(payload.Code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if !room.hasRoom() {
		http.Error(w, "salon plein (5 joueurs maximum)", http.StatusConflict)
		return
	}
	respondJSON(w, map[string]string{"status": "ok", "code": room.code})
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
	code := normalizeRoomCode(r.URL.Query().Get("room"))
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
	room, err := getRoomForJoin(payload.Code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if payload.Host != "" && !isRoomHost(room.code, payload.Host) {
		http.Error(w, "action reservee a l'hote", http.StatusForbidden)
		return
	}
	room.demarrerManche(false)
	respondJSON(w, map[string]string{"status": "started"})
}

func ensureDefaultRoom() *Room {
	roomsMu.Lock()
	defer roomsMu.Unlock()
	if existing, ok := rooms[defaultRoomCode]; ok {
		return existing
	}
	room := newRoom(defaultRoomCode)
	rooms[defaultRoomCode] = room
	return room
}

func defaultRoom() *Room {
	roomsMu.RLock()
	room, ok := rooms[defaultRoomCode]
	roomsMu.RUnlock()
	if ok {
		return room
	}
	return ensureDefaultRoom()
}

func getRoom(code string) (*Room, bool) {
	code = normalizeRoomCode(code)
	if code == "" {
		return defaultRoom(), true
	}
	roomsMu.RLock()
	room, ok := rooms[code]
	roomsMu.RUnlock()
	return room, ok
}

func getRoomForJoin(code string) (*Room, error) {
	code = normalizeRoomCode(code)
	if code == "" {
		return nil, fmt.Errorf("code requis")
	}
	roomsMu.RLock()
	room, ok := rooms[code]
	roomsMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("salon %s introuvable", code)
	}
	return room, nil
}

func createConfiguredRoom(reg GameConfig, host string) *Room {
	room := newRoom(generateRoomCode())
	room.applyConfig(reg)
	roomsMu.Lock()
	rooms[room.code] = room
	roomsMu.Unlock()
	persistRoomConfiguration(room.code, reg, host)
	return room
}

func generateRoomCode() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for {
		builder := strings.Builder{}
		for i := 0; i < roomCodeLength; i++ {
			builder.WriteByte(letters[rand.Intn(len(letters))])
		}
		code := builder.String()
		roomsMu.RLock()
		_, exists := rooms[code]
		roomsMu.RUnlock()
		if !exists {
			return code
		}
	}
}

func sanitizeCategories(cats []string) []string {
	seen := make(map[string]struct{})
	res := []string{}
	for _, c := range cats {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
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
