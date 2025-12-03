package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

const nombreJoueurs = 2

type ResultatJoueur struct {
	Indice int
	Numero int
	Nom    string
	Score  int
}

type DonneesPage struct {
	Lettre     string
	Categories []string
	Joueurs    []ResultatJoueur
	Soumis     bool
}

var modeleHTML *template.Template

func main() {
	var erreurChargement error
	modeleHTML, erreurChargement = template.ParseFiles("templates/ptitbac.html")
	if erreurChargement != nil {
		log.Fatalf("Erreur de chargement du template : %v", erreurChargement)
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", gestionnairePetitBac)

	log.Println("Serveur lance sur http://localhost:8080")
	if erreurEcoute := http.ListenAndServe(":8080", nil); erreurEcoute != nil {
		log.Fatalf("Erreur serveur : %v", erreurEcoute)
	}
}

func gestionnairePetitBac(reponse http.ResponseWriter, requete *http.Request) {
	categories := ObtenirCategories()
	lettreCourante := ObtenirLettreAleatoire()
	joueurs := make([]ResultatJoueur, nombreJoueurs)

	switch requete.Method {
	case http.MethodGet:
		for i := 0; i < nombreJoueurs; i++ {
			joueurs[i] = ResultatJoueur{
				Indice: i,
				Numero: i + 1,
			}
		}

	case http.MethodPost:
		if err := requete.ParseForm(); err != nil {
			http.Error(reponse, "Erreur de formulaire", http.StatusBadRequest)
			return
		}

		lettreFormulaire := strings.TrimSpace(requete.FormValue("lettre"))
		if lettreFormulaire != "" {
			lettreMajuscule := strings.ToUpper(lettreFormulaire)
			runes := []rune(lettreMajuscule)
			if len(runes) > 0 {
				lettreCourante = runes[0]
			}
		}

		for i := 0; i < nombreJoueurs; i++ {
			nom := requete.FormValue(fmt.Sprintf("joueur%d_nom", i))
			reponses := make(map[string]string)

			for j, categorie := range categories {
				nomChamp := fmt.Sprintf("joueur%d_categorie%d", i, j)
				reponses[categorie] = requete.FormValue(nomChamp)
			}

			score := 0
			for _, categorie := range categories {
				if EstValidePourLettre(reponses[categorie], lettreCourante) {
					score++
				}
			}

			joueurs[i] = ResultatJoueur{
				Indice: i,
				Numero: i + 1,
				Nom:    nom,
				Score:  score,
			}
		}

	default:
		http.Error(reponse, "Methode non supportee", http.StatusMethodNotAllowed)
		return
	}

	donnees := DonneesPage{
		Lettre:     string(lettreCourante),
		Categories: categories,
		Joueurs:    joueurs,
		Soumis:     requete.Method == http.MethodPost,
	}

	if err := modeleHTML.Execute(reponse, donnees); err != nil {
		http.Error(reponse, "Erreur interne", http.StatusInternalServerError)
		log.Printf("Erreur template : %v", err)
	}
}
