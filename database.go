package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func initDatabase() {
	var err error
	db, err = sql.Open("sqlite", "./blindtest.db")
	if err != nil {
		log.Fatal("❌ Erreur lors de l'ouverture de la base de données:", err)
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		pseudo TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatal("❌ Erreur lors de la création de la table users:", err)
	}

	log.Println("✅ Base de données SQLite initialisée")
}
