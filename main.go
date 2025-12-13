package main

import (
	"html/template"
	"log"
	"net/http"

	blindtest "groupie-tracker/BlindTest"
	petitbac "groupie-tracker/PetitBac"
)

func main() {
	initDatabase()

	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", pageAccueil)
	http.HandleFunc("/login", pageLogin)
	http.HandleFunc("/register", pageRegister)
	http.HandleFunc("/logout", pageLogout)
	http.HandleFunc("/api/user", apiUserInfo)

	if err := petitbac.RegisterRoutes(requireAuth, petitBacUserResolver); err != nil {
		log.Fatal(err)
	}
	blindtest.RegisterRoutes(requireAuth)

	log.Println("SERVEUR PRET")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func pageAccueil(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	tmpl, err := template.ParseFiles("web/index.html")
	if err != nil {
		log.Println(err)
		return
	}
	tmpl.Execute(w, nil)
}

func petitBacUserResolver(r *http.Request) (*petitbac.UserInfo, error) {
	user, err := getUserFromSession(r)
	if err != nil {
		return nil, err
	}
	return &petitbac.UserInfo{ID: user.ID, Pseudo: user.Pseudo}, nil
}
