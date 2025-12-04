package main

import (
	"log"
	"net/http"
	"text/template"
)

func main() {
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("web/templates/index.html")
		if err != nil {
			http.Error(w, "Erreur interne : Impossible de charger la page", 500)
			log.Println("Erreur Template:", err)
			return
		}
		tmpl.Execute(w, nil)
	})

	log.Println("🌸 Serveur lancé sur http://localhost:8080 (Hello Kitty Style activated)")
	http.ListenAndServe(":8080", nil)
}