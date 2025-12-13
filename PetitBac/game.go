package petitbac

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/gorilla/websocket"
)

type UserInfo struct {
	ID     int
	Pseudo string
}

var (
	tplJeu              *template.Template
	tplHome             *template.Template
	tplCreateCategories *template.Template
	tplCreateTime       *template.Template
	tplJoinRoom         *template.Template
	tplWaiting          *template.Template
	userResolver        func(*http.Request) (*UserInfo, error)
	upgrader            = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func RegisterRoutes(
	authMiddleware func(http.HandlerFunc) http.HandlerFunc,
	resolver func(*http.Request) (*UserInfo, error),
) error {
	var err error
	userResolver = resolver

	if tplJeu, err = template.ParseFiles("PetitBac/templates/ptitbac.html"); err != nil {
		return fmt.Errorf("impossible de charger PetitBac/templates/ptitbac.html: %w", err)
	}
	if tplHome, err = template.ParseFiles("PetitBac/templates/ptitbac_home.html"); err != nil {
		return fmt.Errorf("impossible de charger PetitBac/templates/ptitbac_home.html: %w", err)
	}
	if tplCreateCategories, err = template.ParseFiles("PetitBac/templates/ptitbac_create_categories.html"); err != nil {
		return fmt.Errorf("impossible de charger PetitBac/templates/ptitbac_create_categories.html: %w", err)
	}
	if tplCreateTime, err = template.ParseFiles("PetitBac/templates/ptitbac_create_time.html"); err != nil {
		return fmt.Errorf("impossible de charger PetitBac/templates/ptitbac_create_time.html: %w", err)
	}
	if tplJoinRoom, err = template.ParseFiles("PetitBac/templates/ptitbac_join_room.html"); err != nil {
		return fmt.Errorf("impossible de charger PetitBac/templates/ptitbac_join_room.html: %w", err)
	}
	if tplWaiting, err = template.ParseFiles("PetitBac/templates/ptitbac_waiting.html"); err != nil {
		return fmt.Errorf("impossible de charger PetitBac/templates/ptitbac_waiting.html: %w", err)
	}
	if err := initPetitBacStore(); err != nil {

		return fmt.Errorf("initialisation base PetitBac: %w", err)

		return fmt.Errorf("initialisation base Petit Bac: %w", err)

	}

	http.HandleFunc("/PetitBac", authMiddleware(pagePetitBacHome))
	http.HandleFunc("/PetitBac/create/categories", authMiddleware(pageCreateCategories))
	http.HandleFunc("/PetitBac/create/time", authMiddleware(pageCreateTime))
	http.HandleFunc("/PetitBac/join", authMiddleware(pageJoinSalon))
	http.HandleFunc("/PetitBac/wait", authMiddleware(pageWaitingRoom))
	http.HandleFunc("/PetitBac/play", authMiddleware(pageJeu))
	http.HandleFunc("/PetitBac/rooms/players", authMiddleware(handleRoomPlayers))
	http.HandleFunc("/PetitBac/rooms/start", authMiddleware(handleStartGame))
	http.HandleFunc("/ws", socketJeu)
	http.HandleFunc("/config", configJeu)
	registerSalonHandlers(authMiddleware)

	fsJeu := http.FileServer(http.Dir("PetitBac/Pstatic"))
	http.Handle("/Pstatic/", http.StripPrefix("/Pstatic/", fsJeu))

	salons.defaultSalon().demarrerManche(false)
	return nil
}
