package main

import (
	"math/rand"
	"strings"
)

// ObtenirCategories retourne la liste des categories du Petit Bac
func ObtenirCategories() []string {
	return []string{
		"Artiste",
		"Album",
		"Groupe de musique",
		"Instrument de musique",
		"Featuring",
	}
}

// ObtenirLettreAleatoire retourne une lettre aleatoire entre 'A' et 'Z'
func ObtenirLettreAleatoire() rune {
	return rune('A' + rand.Intn(26))
}

// CalculerScoreReponses calcule le nombre de reponses non vides
func CalculerScoreReponses(reponses map[string]string, categories []string) int {
	score := 0
	for _, categorie := range categories {
		if strings.TrimSpace(reponses[categorie]) != "" {
			score++
		}
	}
	return score
}

// EstValidePourLettre verifie qu'une reponse commence par la lettre donnee (sans tenir compte de la casse)
func EstValidePourLettre(reponse string, lettre rune) bool {
	reponse = strings.TrimSpace(reponse)
	if reponse == "" {
		return false
	}
	premiereRune := []rune(strings.ToUpper(reponse))[0]
	return premiereRune == lettre
}

// CalculerScoresCollectifs attribue 0/1/2 points selon la presence et l'unicite des reponses.
// Une reponse n'est consideree valide que si elle obtient au moins 2/3 de validations (joueurs actifs).
func CalculerScoresCollectifs(reponsesJoueurs []map[string]string, categories []string, lettre rune) []int {
	scores := make([]int, len(reponsesJoueurs))
	if len(reponsesJoueurs) == 0 {
		return scores
	}

	seuilValidations := (2*len(reponsesJoueurs) + 2) / 3 // ceil(2/3 * n)

	for _, categorie := range categories {
		occurences := make(map[string]int)
		reponsesValides := make([]bool, len(reponsesJoueurs))
		reponsesNormalisees := make([]string, len(reponsesJoueurs))

		for indiceJoueur, reponses := range reponsesJoueurs {
			reponse := reponses[categorie]
			if EstValidePourLettre(reponse, lettre) {
				normalisee := strings.ToUpper(strings.TrimSpace(reponse))
				occurences[normalisee]++
				reponsesValides[indiceJoueur] = true
				reponsesNormalisees[indiceJoueur] = normalisee
			}
		}

		for indiceJoueur := range reponsesJoueurs {
			if !reponsesValides[indiceJoueur] {
				continue
			}
			if occurences[reponsesNormalisees[indiceJoueur]] < seuilValidations {
				continue
			}
			if occurences[reponsesNormalisees[indiceJoueur]] == 1 {
				scores[indiceJoueur] += 2
			} else {
				scores[indiceJoueur]++
			}
		}
	}

	return scores
}
