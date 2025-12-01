package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	//"time"
)

func main() {
	// Initialisation du random
	//rand.Seed(time.Now().UnixNano())

	// Liste des catégories du Petit Bac
	categories := []string{
		"Artiste",
		"Album",
		"Groupe de musique",
		"Instrument de musique",
		"Featuring",
	}

	// Tirage d'une lettre aléatoire (A–Z)
	letter := rune('A' + rand.Intn(rand.Intn(26)))

	fmt.Println("===== PETIT BAC =====")
	fmt.Printf("La lettre tirée est : %c\n", letter)
	fmt.Println("Remplis une réponse par catégorie, en commençant par cette lettre.")
	fmt.Println("----------------------------------")

	// Map pour stocker les réponses de l'utilisateur
	answers := make(map[string]string)

	reader := bufio.NewReader(os.Stdin)

	for _, cat := range categories {
		for {
			fmt.Printf("%s (%c) : ", cat, letter)
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)

			// On accepte vide, mais si tu veux forcer la lettre :
			if text == "" {
				fmt.Println("Réponse vide, tu peux passer, mais tu peux aussi retenter")
				answers[cat] = ""
				break
			}

			// Vérifie que la réponse commence bien par la lettre (case insensitive)
			firstRune := []rune(strings.ToUpper(text))[0]
			if firstRune != letter {
				fmt.Printf("Ta réponse ne commence pas par '%c'. Tu veux réessayer ? (o/N) : ", letter)
				choice, _ := reader.ReadString('\n')
				choice = strings.TrimSpace(strings.ToLower(choice))
				if choice == "o" || choice == "oui" {
					continue
				}
			}

			answers[cat] = text
			break
		}
	}

	fmt.Println("\n===== RÉCAPITULATIF =====")
	fmt.Printf("Lettre jouée : %c\n\n", letter)

	score := 0
	for _, cat := range categories {
		resp := answers[cat]
		if resp == "" {
			fmt.Printf("- %s : (aucune réponse)\n", cat)
		} else {
			fmt.Printf("- %s : %s\n", cat, resp)
			score++
		}
	}

	fmt.Println("\n===== SCORE =====")
	fmt.Printf("Réponses remplies : %d / %d\n", score, len(categories))
	fmt.Println("Fin de la manche, merci d'avoir joué !")
}
