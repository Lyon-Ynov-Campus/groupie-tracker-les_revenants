package main

import (
	"log"
	"net/http"
	"text/template"
)

func main() {
	// 1. Servir les fichiers statiques (CSS, IMG)
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// 2. Route pour la page d'accueil (Port 8080)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("web/templates/index.html")
		if err != nil {
			http.Error(w, "Erreur interne : Impossible de charger la page", 500)
			log.Println("Erreur Template:", err)
			return
		}
		tmpl.Execute(w, nil)
	})

	// 3. On lance UNIQUEMENT le port 8080 ici
	log.Println("🌸 Serveur Principal lancé sur http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}