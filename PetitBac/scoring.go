package petitbac

import (
	"math/rand"
	"strings"
)

func (r *Room) scoresFin() {
	r.mu.Lock()
	if len(r.players) == 0 {
		r.mu.Unlock()
		return
	}
	tous := []map[string]string{}
	ordre := []*Player{}
	for _, j := range r.players {
		if j.Actif {
			tous = append(tous, j.Reponses)
			ordre = append(ordre, j)
		} else {
			j.Score = 0
		}
	}
	cats := append([]string(nil), r.reglages.Categories...)
	lettre := r.lettreActu
	r.mu.Unlock()

	points := scoresCollectifs(tous, cats, lettre)

	r.mu.Lock()
	for i, j := range ordre {
		if i < len(points) {
			j.Score = points[i]
			j.Total += points[i]
		}
	}
	r.mu.Unlock()
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

func reponseValide(texte string, lettre rune) bool {
	texte = strings.TrimSpace(texte)
	if texte == "" {
		return false
	}
	texte = strings.ToUpper(texte)
	valeurs := []rune(texte)
	if len(valeurs) == 0 {
		return false
	}
	return valeurs[0] == lettre
}

func scoresCollectifs(reponsesJoueurs []map[string]string, categories []string, lettre rune) []int {
	resultats := make([]int, len(reponsesJoueurs))
	if len(reponsesJoueurs) == 0 {
		return resultats
	}

	seuil := (2*len(reponsesJoueurs) + 2) / 3

	for _, cat := range categories {
		occur := make(map[string]int)
		ok := make([]bool, len(reponsesJoueurs))
		objets := make([]string, len(reponsesJoueurs))

		for i := 0; i < len(reponsesJoueurs); i++ {
			reponses := reponsesJoueurs[i]
			texte := reponses[cat]
			if reponseValide(texte, lettre) {
				forme := strings.ToUpper(strings.TrimSpace(texte))
				occur[forme]++
				ok[i] = true
				objets[i] = forme
			}
		}

		for i := 0; i < len(reponsesJoueurs); i++ {
			if !ok[i] {
				continue
			}
			if occur[objets[i]] < seuil {
				continue
			}
			if occur[objets[i]] == 1 {
				resultats[i] += 2
			} else {
				resultats[i]++
			}
		}
	}

	return resultats
}
