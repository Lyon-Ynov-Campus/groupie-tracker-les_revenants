package petitbac

import (
	"strings"

	"github.com/gorilla/websocket"
)

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
		switch msg.Type {
		case "join":
			if n := strings.TrimSpace(msg.Nom); n != "" {
				j.Nom = n
			}
		case "answers":
			if s.mancheEnCours && j.Actif {
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
			}
		case "ready":
			if s.attenteVotes && !j.Pret {
				j.Pret = true
				s.mu.Unlock()
				if s.verifieVotes() {
					continue
				}
				s.envoyerEtat()
				continue
			}
		}
		s.mu.Unlock()
		s.envoyerEtat()
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
