package petitbac

import (
	"fmt"
	"strings"

	"github.com/gorilla/websocket"
)

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
