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
