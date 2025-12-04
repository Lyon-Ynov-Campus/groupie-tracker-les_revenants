package main

import (
	"math/rand"
	"strings"
	"time"
)

// Toutes tes fonctions ici…

func ObtenirCategories() []string {
	return []string{
		"Artistes",
		"Albums",
		"Groupe de musique",
		"Instrument de musiques",
		"Featuring",
	}
}

func ObtenirLettreAleatoire() rune {
	rand.Seed(time.Now().UnixNano())
	return rune('A' + rand.Intn(26))
}

func EstValidePourLettre(rep string, lettre rune) bool {
	rep = strings.TrimSpace(rep)
	if rep == "" {
		return false
	}
	first := []rune(strings.ToUpper(rep))[0]
	return first == lettre
}
