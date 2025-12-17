package petitbac

import (
	"strings"

	"github.com/gorilla/websocket"
)

func (r *Room) boucleWS(conn *websocket.Conn) {
	defer func() {
		r.removePlayer(conn)
		conn.Close()
		r.envoyerEtat()
	}()

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}

		r.mu.Lock()
		playerID, ok := r.connections[conn]
		if !ok {
			r.mu.Unlock()
			return
		}
		player := r.players[playerID]
		switch msg.Type {
		case "join":
			if n := strings.TrimSpace(msg.Nom); n != "" {
				player.Nom = n
				recordPlayerEntry(r.code, player.Nom)
			}
		case "answers":
			if r.mancheEnCours && player.Actif {
				complet := true
				for _, cat := range r.reglages.Categories {
					val := msg.Reponses[cat]
					player.Reponses[cat] = val
					if strings.TrimSpace(val) == "" {
						complet = false
					}
				}
				if complet {
					r.mu.Unlock()
					r.finMancheRemplie()
					continue
				}
			}
		case "ready":
			if r.attenteVotes && !player.Pret {
				player.Pret = true
				r.mu.Unlock()
				if r.verifieVotes() {
					continue
				}
				r.envoyerEtat()
				continue
			}
		case "validate":
			validationID := msg.ValidationID
			voterID := player.ID
			r.mu.Unlock()
			r.handleValidationVote(voterID, msg.Approve, validationID)
			continue
		}
		r.mu.Unlock()
		r.envoyerEtat()
	}
}

func (r *Room) envoyerEtat() {
	r.mu.RLock()
	etat := GameState{
		Type:           "state",
		Lettre:         string(r.lettreActu),
		Categories:     append([]string(nil), r.reglages.Categories...),
		Secondes:       r.tempsRest,
		MancheActive:   r.mancheEnCours,
		Attente:        r.attenteVotes,
		NumeroManche:   r.nbManches,
		LimiteManches:  r.reglages.Manches,
		JeuTermine:     r.termine,
		TempsParManche: r.reglages.Temps,
	}
	liste := make([]Player, 0, len(r.players))
	dest := make([]*websocket.Conn, 0, len(r.players))
	for _, j := range r.players {
		liste = append(liste, *j)
		dest = append(dest, j.Conn)
		if j.Actif {
			etat.Actifs++
		}
		if j.Pret {
			etat.CompteurPrets++
		}
	}
	etat.Joueurs = liste
	etat.CompteurTotal = len(r.players)
	if r.validationActive {
		etat.ValidationActive = true
		pending := len(r.validationEntries) - r.validationIndex
		if pending < 0 {
			pending = 0
		}
		etat.ValidationPending = pending
		if r.validationIndex < len(r.validationEntries) {
			entry := r.validationEntries[r.validationIndex]
			copyApprovals := make(map[string]bool, len(entry.Approvals))
			for id, v := range entry.Approvals {
				copyApprovals[id] = v
			}
			etat.ValidationEntry = &ValidationDisplay{
				ID:        entry.ID,
				PlayerID:  entry.PlayerID,
				PlayerNom: entry.PlayerNom,
				Category:  entry.Category,
				Answer:    entry.Answer,
				Required:  entry.Required,
				Votes:     len(entry.Approvals),
				Approvals: copyApprovals,
				Accepted:  entry.Accepted,
				Completed: entry.Completed,
			}
		}
	}
	r.mu.RUnlock()

	persistPlayersSnapshot(r.code, liste)

	for _, c := range dest {
		if c != nil {
			c.WriteJSON(etat)
		}
	}
}
