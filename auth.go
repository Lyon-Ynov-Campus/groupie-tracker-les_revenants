package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       int
	Pseudo   string
	Email    string
	Password string
}

type Session struct {
	UserID    int
	ExpiresAt time.Time
}

var sessions = make(map[string]Session)

func pageRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("web/register.html")
		if err != nil {
			log.Println(err)
			http.Error(w, "Erreur serveur", 500)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == http.MethodPost {
		pseudo := strings.TrimSpace(r.FormValue("pseudo"))
		email := strings.TrimSpace(r.FormValue("email"))
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")

		if pseudo == "" || email == "" || password == "" {
			http.Error(w, "Tous les champs sont requis", 400)
			return
		}

		if !isValidEmail(email) {
			http.Error(w, "Adresse email invalide", 400)
			return
		}

		if !isValidPassword(password) {
			http.Error(w, "Le mot de passe doit contenir au moins 12 caractères, une majuscule, une minuscule, un chiffre et un caractère spécial", 400)
			return
		}

		if password != confirmPassword {
			http.Error(w, "Les mots de passe ne correspondent pas", 400)
			return
		}

		var existingUser User
		err := db.QueryRow("SELECT id FROM users WHERE pseudo = ? OR email = ?", pseudo, email).Scan(&existingUser.ID)
		if err != sql.ErrNoRows {
			http.Error(w, "Ce pseudo ou cet email est déjà utilisé", 400)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Println("Erreur hashage mot de passe:", err)
			http.Error(w, "Erreur serveur", 500)
			return
		}

		result, err := db.Exec("INSERT INTO users (pseudo, email, password) VALUES (?, ?, ?)", pseudo, email, string(hashedPassword))
		if err != nil {
			log.Println("Erreur insertion utilisateur:", err)
			http.Error(w, "Erreur lors de la création du compte", 500)
			return
		}

		userID, err := result.LastInsertId()
		if err != nil {
			log.Println("Erreur récupération ID utilisateur:", err)
			http.Error(w, "Erreur serveur", 500)
			return
		}

		sessionToken := generateSessionToken()
		sessions[sessionToken] = Session{
			UserID:    int(userID),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Path:     "/",
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func pageLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("web/login.html")
		if err != nil {
			log.Println(err)
			http.Error(w, "Erreur serveur", 500)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	if r.Method == http.MethodPost {
		identifier := strings.TrimSpace(r.FormValue("identifier"))
		password := r.FormValue("password")

		if identifier == "" || password == "" {
			http.Error(w, "Tous les champs sont requis", 400)
			return
		}

		var user User
		err := db.QueryRow("SELECT id, pseudo, email, password FROM users WHERE pseudo = ? OR email = ?", identifier, identifier).Scan(&user.ID, &user.Pseudo, &user.Email, &user.Password)
		if err != nil {
			http.Error(w, "Identifiant ou mot de passe incorrect", 401)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err != nil {
			http.Error(w, "Identifiant ou mot de passe incorrect", 401)
			return
		}

		sessionToken := generateSessionToken()
		sessions[sessionToken] = Session{
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Path:     "/",
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func pageLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		delete(sessions, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return false
	}

	session, exists := sessions[cookie.Value]
	if !exists {
		return false
	}

	if time.Now().After(session.ExpiresAt) {
		delete(sessions, cookie.Value)
		return false
	}

	return true
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAuthenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func isValidPassword(password string) bool {
	if len(password) < 12 {
		return false
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password)

	return hasUpper && hasLower && hasNumber && hasSpecial
}

func generateSessionToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Println("Erreur génération token:", err)
		return ""
	}
	return hex.EncodeToString(b)
}

func getUserFromSession(r *http.Request) (*User, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return nil, err
	}

	session, exists := sessions[cookie.Value]
	if !exists {
		return nil, sql.ErrNoRows
	}

	var user User
	err = db.QueryRow("SELECT id, pseudo, email FROM users WHERE id = ?", session.UserID).Scan(&user.ID, &user.Pseudo, &user.Email)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func apiUserInfo(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromSession(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"authenticated": false})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": true,
		"pseudo":        user.Pseudo,
		"email":         user.Email,
	})
}
