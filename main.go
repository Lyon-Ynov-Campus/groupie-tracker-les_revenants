package main

import (
	"log"
	"net/http"
	"text/template"
)

func main() {
	// 1. Servir les fichiers statiques (CSS, JS, Images)
	// Cela permet d'accéder à /static/css/style.css ou /static/img/logo.png
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// 2. Route pour la Landing Page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// On parse le template HTML
		tmpl, err := template.ParseFiles("web/templates/index.html")
		if err != nil {
			http.Error(w, "Erreur interne : Impossible de charger la page", 500)
			log.Println("Erreur Template:", err)
			return
		}
		// On l'envoie au navigateur
		tmpl.Execute(w, nil)
	})

	log.Println("🌸 Serveur lancé sur http://localhost:8080 (Hello Kitty Style activated)")
	http.ListenAndServe(":8080", nil)
}