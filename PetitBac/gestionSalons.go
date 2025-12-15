package petitbac

import (
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
)

func newRoom(code string) *Room {
	return &Room{
		code:        normalizeRoomCode(code),
		reglages:    GameConfig{Categories: listeCategories(), Temps: 90, Manches: 5},
		lettreActu:  lettreAleatoire(),
		players:     make(map[string]*Player),
		connections: make(map[*websocket.Conn]string),
	}
}

func normalizeRoomCode(code string) string {
	return strings.TrimSpace(strings.ToUpper(code))
}

func (r *Room) hasRoom() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.players) < maxSalonPlayers
}

func (r *Room) addPlayer(conn *websocket.Conn) (*Player, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.players) >= maxSalonPlayers {
		return nil, fmt.Errorf("salon plein (max %d joueurs)", maxSalonPlayers)
	}
	r.compteurJoueurs++
	playerID := fmt.Sprintf("j-%s-%d", strings.ToLower(r.code), r.compteurJoueurs)
	player := &Player{
		ID:       playerID,
		Nom:      "Anonyme",
		Reponses: make(map[string]string),
		Actif:    r.mancheEnCours && !r.termine,
		Conn:     conn,
	}
	r.players[playerID] = player
	r.connections[conn] = playerID
	return player, nil
}

func (r *Room) removePlayer(conn *websocket.Conn) {
	r.mu.Lock()
	finalize := false
	if id, ok := r.connections[conn]; ok {
		player := r.players[id]
		wasActive := false
		if player != nil {
			wasActive = player.Actif
		}
		delete(r.players, id)
		delete(r.connections, conn)
		finalize = r.adjustValidationOnLeaveLocked(id, wasActive)
	}
	r.mu.Unlock()

	if finalize {
		r.modeAttente()
	}
}
