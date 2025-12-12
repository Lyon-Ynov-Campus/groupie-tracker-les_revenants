package petitbac

import "time"

func (s *salon) templateData() donneesPage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return donneesPage{
		Lettre:          string(s.lettreActu),
		Categories:      append([]string(nil), s.reglages.Categories...),
		TempsParManche:  s.reglages.Temps,
		NombreDeManches: s.reglages.Manches,
		SalonCode:       s.code,
	}
}

func (s *salon) applyConfig(reg reglageJeu) {
	s.mu.Lock()
	if len(reg.Categories) > 0 {
		s.reglages.Categories = reg.Categories
	}
	if reg.Temps >= 15 {
		s.reglages.Temps = reg.Temps
	}
	if reg.Manches > 0 {
		s.reglages.Manches = reg.Manches
	}
	s.mancheEnCours, s.attenteVotes, s.termine = false, false, false
	s.nbManches, s.tempsRest = 0, 0
	s.lettreActu = lettreAleatoire()
	for _, j := range s.joueurs {
		j.Score, j.Total, j.Actif, j.Pret = 0, 0, false, false
		j.Reponses = make(map[string]string)
	}
	s.mu.Unlock()
}

func (s *salon) demarrerManche(selection bool) {
	s.mu.Lock()
	if s.termine || (s.reglages.Manches > 0 && s.nbManches >= s.reglages.Manches) {
		s.finPartieLocked()
		s.mu.Unlock()
		s.envoyerEtat()
		return
	}

	actifs := 0
	for _, j := range s.joueurs {
		j.Score = 0
		j.Reponses = make(map[string]string)
		j.Actif = !selection || j.Pret
		if j.Actif {
			actifs++
		}
		j.Pret = false
	}

	if selection && actifs == 0 {
		s.attenteVotes = true
		s.tempsRest = 0
		s.mu.Unlock()
		s.envoyerEtat()
		return
	}

	s.nbManches++
	if s.reglages.Temps <= 0 {
		s.reglages.Temps = 90
	}
	s.lettreActu = lettreAleatoire()
	s.tempsRest = s.reglages.Temps
	s.mancheEnCours, s.attenteVotes, s.termine = true, false, false
	s.mu.Unlock()

	go s.compteRebours()
	s.envoyerEtat()
}

func (s *salon) compteRebours() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for range t.C {
		s.mu.Lock()
		if !s.mancheEnCours {
			s.mu.Unlock()
			return
		}
		if s.tempsRest > 0 {
			s.tempsRest--
		}
		if s.tempsRest == 0 {
			s.mancheEnCours = false
			s.mu.Unlock()
			s.scoresFin()
			s.modeAttente()
			s.envoyerEtat()
			return
		}
		s.mu.Unlock()
		s.envoyerEtat()
	}
}

func (s *salon) finMancheRemplie() {
	s.mu.Lock()
	if !s.mancheEnCours {
		s.mu.Unlock()
		return
	}
	s.mancheEnCours = false
	s.tempsRest = 0
	s.mu.Unlock()
	s.scoresFin()
	s.modeAttente()
	s.envoyerEtat()
}

func (s *salon) modeAttente() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.reglages.Manches > 0 && s.nbManches >= s.reglages.Manches {
		s.finPartieLocked()
		return
	}
	s.attenteVotes = true
	s.tempsRest = 0
	for _, j := range s.joueurs {
		j.Actif = false
		j.Pret = false
	}
}

func (s *salon) verifieVotes() bool {
	s.mu.Lock()
	if !s.attenteVotes || len(s.joueurs) == 0 || s.termine {
		s.mu.Unlock()
		return false
	}
	prets := 0
	for _, j := range s.joueurs {
		if j.Pret {
			prets++
		}
	}
	total := len(s.joueurs)
	s.mu.Unlock()

	if prets > 0 && float64(prets) >= float64(total)*0.66 {
		s.demarrerManche(true)
		return true
	}
	return false
}

func (s *salon) finPartieLocked() {
	s.mancheEnCours, s.attenteVotes, s.termine = false, false, true
	s.tempsRest = 0
	for _, j := range s.joueurs {
		j.Actif, j.Pret = false, false
	}
}
