package petitbac

import "time"

func (r *Room) templateData() PageData {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return PageData{
		Lettre:          string(r.lettreActu),
		Categories:      append([]string(nil), r.reglages.Categories...),
		TempsParManche:  r.reglages.Temps,
		NombreDeManches: r.reglages.Manches,
		SalonCode:       r.code,
	}
}

func (r *Room) applyConfig(reg GameConfig) {
	r.mu.Lock()
	if len(reg.Categories) > 0 {
		r.reglages.Categories = reg.Categories
	}
	if reg.Temps >= 15 {
		r.reglages.Temps = reg.Temps
	}
	if reg.Manches > 0 {
		r.reglages.Manches = reg.Manches
	}
	r.mancheEnCours, r.attenteVotes, r.termine = false, false, false
	r.nbManches, r.tempsRest = 0, 0
	r.lettreActu = lettreAleatoire()
	for _, j := range r.players {
		j.Score, j.Total, j.Actif, j.Pret = 0, 0, false, false
		j.Reponses = make(map[string]string)
	}
	r.mu.Unlock()
}

func (r *Room) demarrerManche(selection bool) {
	r.mu.Lock()
	if r.termine || (r.reglages.Manches > 0 && r.nbManches >= r.reglages.Manches) {
		r.finPartieLocked()
		r.mu.Unlock()
		r.envoyerEtat()
		return
	}

	actifs := 0
	for _, j := range r.players {
		j.Score = 0
		j.Reponses = make(map[string]string)
		j.Actif = !selection || j.Pret
		if j.Actif {
			actifs++
		}
		j.Pret = false
	}

	if selection && actifs == 0 {
		r.attenteVotes = true
		r.tempsRest = 0
		r.mu.Unlock()
		r.envoyerEtat()
		return
	}

	r.nbManches++
	if r.reglages.Temps <= 0 {
		r.reglages.Temps = 90
	}
	r.lettreActu = lettreAleatoire()
	r.tempsRest = r.reglages.Temps
	r.mancheEnCours, r.attenteVotes, r.termine = true, false, false
	r.validationActive = false
	r.validationEntries = nil
	r.validationIndex = 0
	r.mu.Unlock()

	go r.compteRebours()
	r.envoyerEtat()
}

func (r *Room) compteRebours() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		r.mu.Lock()
		if !r.mancheEnCours {
			r.mu.Unlock()
			return
		}
		if r.tempsRest > 0 {
			r.tempsRest--
		}
		if r.tempsRest == 0 {
			r.mancheEnCours = false
			r.mu.Unlock()
			r.startValidationPhase()
			return
		}
		r.mu.Unlock()
		r.envoyerEtat()
	}
}

func (r *Room) finMancheRemplie() {
	r.mu.Lock()
	if !r.mancheEnCours {
		r.mu.Unlock()
		return
	}
	r.mancheEnCours = false
	r.tempsRest = 0
	r.mu.Unlock()
	r.startValidationPhase()
}

func (r *Room) modeAttente() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.reglages.Manches > 0 && r.nbManches >= r.reglages.Manches {
		r.finPartieLocked()
		return
	}
	r.attenteVotes = true
	r.tempsRest = 0
	r.validationActive = false
	r.validationEntries = nil
	r.validationIndex = 0
	for _, j := range r.players {
		j.Actif = false
		j.Pret = false
	}
}

func (r *Room) verifieVotes() bool {
	r.mu.Lock()
	if !r.attenteVotes || len(r.players) == 0 || r.termine {
		r.mu.Unlock()
		return false
	}
	prets := 0
	for _, j := range r.players {
		if j.Pret {
			prets++
		}
	}
	total := len(r.players)
	r.mu.Unlock()

	if prets > 0 && float64(prets) >= float64(total)*0.66 {
		r.demarrerManche(true)
		return true
	}
	return false
}

func (r *Room) finPartieLocked() {
	r.mancheEnCours, r.attenteVotes, r.termine = false, false, true
	r.tempsRest = 0
	for _, j := range r.players {
		j.Actif, j.Pret = false, false
	}
}
