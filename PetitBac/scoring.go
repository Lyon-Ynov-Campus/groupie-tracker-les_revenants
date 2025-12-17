package petitbac

import (
	"math/rand"
	"strings"
)

func (r *Room) startValidationPhase() {
	var finalize bool

	r.mu.Lock()
	if r.validationActive {
		r.mu.Unlock()
		return
	}
	entries := r.buildValidationEntriesLocked()
	if len(entries) == 0 {
		r.validationActive = false
		r.validationEntries = nil
		r.validationIndex = 0
		finalize = true
	} else {
		r.validationEntries = entries
		r.validationIndex = 0
		r.validationActive = true
		finalize = r.ensureValidationProgressLocked()
	}
	r.mu.Unlock()

	if finalize {
		r.modeAttente()
	}
	r.envoyerEtat()
}

func (r *Room) buildValidationEntriesLocked() []*validationEntry {
	active := []*Player{}
	for _, p := range r.players {
		if p.Actif {
			active = append(active, p)
		} else {
			p.Score = 0
		}
	}
	if len(active) == 0 {
		return nil
	}
	requiredBase := len(active) - 1
	if requiredBase < 0 {
		requiredBase = 0
	}
	index := 0
	entries := make([]*validationEntry, 0)
	for _, p := range active {
		displayName := p.Nom
		if strings.TrimSpace(displayName) == "" {
			displayName = "Anonyme"
		}
		for _, cat := range r.reglages.Categories {
			answer := strings.TrimSpace(p.Reponses[cat])
			if answer == "" {
				continue
			}
			entry := &validationEntry{
				ID:        index,
				PlayerID:  p.ID,
				PlayerNom: displayName,
				Category:  cat,
				Answer:    answer,
				Approvals: make(map[string]bool),
				Required:  requiredBase,
			}
			entries = append(entries, entry)
			index++
		}
	}
	return entries
}

func (r *Room) ensureValidationProgressLocked() bool {
	for r.validationActive && r.validationIndex < len(r.validationEntries) {
		entry := r.validationEntries[r.validationIndex]
		if entry.Completed {
			r.validationIndex++
			continue
		}
		if entry.Required <= 0 {
			entry.Completed = true
			entry.Accepted = true
			if target, ok := r.players[entry.PlayerID]; ok {
				target.Score++
				target.Total++
			}
			r.validationIndex++
			continue
		}
		return false
	}
	if r.validationIndex >= len(r.validationEntries) {
		r.validationActive = false
		r.validationEntries = nil
		r.validationIndex = 0
		return true
	}
	return false
}

func (r *Room) handleValidationVote(playerID string, approve bool, validationID int) {
	var finalize bool

	r.mu.Lock()
	finalize = r.applyValidationVoteLocked(playerID, approve, validationID)
	r.mu.Unlock()

	if finalize {
		r.modeAttente()
	}
	r.envoyerEtat()
}

func (r *Room) applyValidationVoteLocked(playerID string, approve bool, validationID int) bool {
	if !r.validationActive || r.validationIndex >= len(r.validationEntries) {
		return false
	}
	current := r.validationEntries[r.validationIndex]
	if current.Completed || current.ID != validationID || current.PlayerID == playerID || current.Required <= 0 {
		return false
	}
	voter, ok := r.players[playerID]
	if !ok || !voter.Actif {
		return false
	}
	if _, voted := current.Approvals[playerID]; voted {
		return false
	}
	if !approve {
		current.Completed = true
		current.Accepted = false
		return r.advanceValidationLocked()
	}
	current.Approvals[playerID] = true
	if len(current.Approvals) >= current.Required {
		current.Completed = true
		current.Accepted = true
		if target, ok := r.players[current.PlayerID]; ok {
			target.Score++
			target.Total++
		}
		return r.advanceValidationLocked()
	}
	return false
}

func (r *Room) advanceValidationLocked() bool {
	r.validationIndex++
	return r.ensureValidationProgressLocked()
}

func (r *Room) adjustValidationOnLeaveLocked(playerID string, wasActive bool) bool {
	if !r.validationActive || r.validationIndex >= len(r.validationEntries) {
		return false
	}
	current := r.validationEntries[r.validationIndex]
	if current.PlayerID == playerID {
		current.Completed = true
		current.Accepted = false
		return r.advanceValidationLocked()
	}
	if !wasActive || current.Required <= 0 {
		return false
	}
	if _, voted := current.Approvals[playerID]; voted {
		delete(current.Approvals, playerID)
	}
	current.Required--
	if current.Required < 0 {
		current.Required = 0
	}
	if current.Required == 0 {
		current.Completed = true
		current.Accepted = len(current.Approvals) == 0
		if current.Accepted {
			if target, ok := r.players[current.PlayerID]; ok {
				target.Score++
				target.Total++
			}
		}
		return r.advanceValidationLocked()
	}
	if len(current.Approvals) >= current.Required {
		current.Completed = true
		current.Accepted = true
		if target, ok := r.players[current.PlayerID]; ok {
			target.Score++
			target.Total++
		}
		return r.advanceValidationLocked()
	}
	return false
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
