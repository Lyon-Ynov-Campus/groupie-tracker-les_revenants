package main

import (
	"html/template"
	"log"
	"net/http"
)

type PageData struct {
	Letter     string
	Categories []string
}

var tmpl *template.Template

func main() {
	// Chargement du template HTML
	var err error
	tmpl, err = template.ParseFiles("templates/ptitbac.html")
	if err != nil {
		log.Fatalf("Erreur de chargement du template : %v", err)
	}

	// Route principale
	http.HandleFunc("/", petitBacHandler)

	log.Println("Serveur lancé sur http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Erreur serveur : %v", err)
	}
}

func petitBacHandler(w http.ResponseWriter, r *http.Request) {
	// Utilisation de la logique définie dans logic.go
	categories := GetCategories()
	letter := GetRandomLetter()

	data := PageData{
		Letter:     string(letter),
		Categories: categories,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
		log.Printf("Erreur template : %v", err)
		return
	}
}
