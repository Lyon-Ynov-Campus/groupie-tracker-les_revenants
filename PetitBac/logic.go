package main

import (
	"math/rand"
	"strings"
)


// GetCategories retourne la liste des catégories du Petit Bac
func GetCategories() []string {
	return []string{
		"Artiste",
		"Album",
		"Groupe de musique",
		"Instrument de musique",
		"Featuring",
	}
}

// GetRandomLetter retourne une lettre aléatoire entre 'A' et 'Z'
func GetRandomLetter() rune {
	return rune('A' + rand.Intn(26))
}

// ScoreAnswers calcule le nombre de réponses non vides
func ScoreAnswers(answers map[string]string, categories []string) int {
	score := 0
	for _, cat := range categories {
		if strings.TrimSpace(answers[cat]) != "" {
			score++
		}
	}
	return score
}

// IsValidForLetter vérifie qu'une réponse commence par la lettre donnée (sans tenir compte de la casse)
func IsValidForLetter(answer string, letter rune) bool {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return false
	}
	firstRune := []rune(strings.ToUpper(answer))[0]
	return firstRune == letter
}
