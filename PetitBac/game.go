package petitbac

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/gorilla/websocket"
)

var (
	tplJeu   *template.Template
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func RegisterRoutes(authMiddleware func(http.HandlerFunc) http.HandlerFunc) error {
	var err error
	tplJeu, err = template.ParseFiles("PetitBac/templates/ptitbac.html")
	if err != nil {
		return fmt.Errorf("impossible de charger PetitBac/templates/ptitbac.html: %w", err)
	}

	http.HandleFunc("/PetitBac", authMiddleware(pageJeu))
	http.HandleFunc("/ws", socketJeu)
	http.HandleFunc("/config", configJeu)
	registerSalonHandlers(authMiddleware)

	fsJeu := http.FileServer(http.Dir("PetitBac/Pstatic"))
	http.Handle("/Pstatic/", http.StripPrefix("/Pstatic/", fsJeu))

	salons.defaultSalon().demarrerManche(false)
	return nil
}
