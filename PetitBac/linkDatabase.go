package petitbac

import (
	"database/sql"
	"encoding/json"
	"log"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

var (
<<<<<<< HEAD
	dbOnce sync.Once
	dbErr  error
	pbDB   *sql.DB
=======
	pbDB   *sql.DB
	dbOnce sync.Once
	dbErr  error
>>>>>>> v1seb
)

type dbPlayer struct {
	Pseudo string `json:"pseudo"`
	Score  int    `json:"score"`
	Room   string `json:"room"`
}

func initPetitBacStore() error {
	dbOnce.Do(func() {
		pbDB, dbErr = sql.Open("sqlite", "./blindtest.db")
		if dbErr != nil {
			return
		}
		if err := createPetitBacTables(pbDB); err != nil {
			dbErr = err
		}
	})
	return dbErr
}

<<<<<<< HEAD
func createPetitBacTables(database *sql.DB) error {
	statements := []string{
=======
func createPetitBacTables(db *sql.DB) error {
	stmts := []string{
>>>>>>> v1seb
		`CREATE TABLE IF NOT EXISTS petitbac_rooms (
			code TEXT PRIMARY KEY,
			host TEXT NOT NULL,
			categories TEXT,
			round_time INTEGER,
			rounds INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS petitbac_players (
			room_code TEXT NOT NULL,
			pseudo TEXT NOT NULL,
			total_score INTEGER DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (room_code, pseudo)
		);`,
	}
<<<<<<< HEAD
	for _, stmt := range statements {
		if _, err := database.Exec(stmt); err != nil {
=======
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
>>>>>>> v1seb
			return err
		}
	}
	return nil
}

func persistRoomConfiguration(code string, reg reglageJeu, host string) {
	if pbDB == nil {
		return
	}
	host = strings.TrimSpace(host)
	if host == "" {
		host = "Anonyme"
	}
	catsJSON, _ := json.Marshal(reg.Categories)
	_, err := pbDB.Exec(`INSERT INTO petitbac_rooms(code, host, categories, round_time, rounds, updated_at)
		VALUES(?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(code) DO UPDATE SET
			host=excluded.host,
			categories=excluded.categories,
			round_time=excluded.round_time,
			rounds=excluded.rounds,
			updated_at=CURRENT_TIMESTAMP;`,
		code, host, string(catsJSON), reg.Temps, reg.Manches)
	if err != nil {
		log.Println("PetitBac: impossible d'enregistrer la configuration:", err)
	}
}

func recordPlayerEntry(roomCode, pseudo string) {
	if pbDB == nil {
		return
	}
	pseudo = strings.TrimSpace(pseudo)
	if pseudo == "" {
		return
	}
	_, err := pbDB.Exec(`INSERT INTO petitbac_players(room_code, pseudo, total_score, updated_at)
		VALUES(?, ?, 0, CURRENT_TIMESTAMP)
		ON CONFLICT(room_code, pseudo) DO UPDATE SET updated_at=CURRENT_TIMESTAMP;`,
		roomCode, pseudo)
	if err != nil {
		log.Println("PetitBac: impossible d'enregistrer le joueur:", err)
	}
}

func persistPlayersSnapshot(roomCode string, joueurs []joueurDonnees) {
	if pbDB == nil {
		return
	}
	tx, err := pbDB.Begin()
	if err != nil {
<<<<<<< HEAD
		log.Println("PetitBac: impossible de demarrer la transaction de sauvegarde:", err)
=======
		log.Println("PetitBac: impossible de démarrer la sauvegarde:", err)
>>>>>>> v1seb
		return
	}
	stmt, err := tx.Prepare(`INSERT INTO petitbac_players(room_code, pseudo, total_score, updated_at)
		VALUES(?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(room_code, pseudo) DO UPDATE SET
			total_score=excluded.total_score,
			updated_at=CURRENT_TIMESTAMP;`)
	if err != nil {
		tx.Rollback()
<<<<<<< HEAD
		log.Println("PetitBac: prepare snapshot:", err)
=======
		log.Println("PetitBac: préparation snapshot impossible:", err)
>>>>>>> v1seb
		return
	}
	defer stmt.Close()
	for _, j := range joueurs {
<<<<<<< HEAD
		if strings.TrimSpace(j.Nom) == "" {
			continue
		}
		if _, err := stmt.Exec(roomCode, j.Nom, j.Total); err != nil {
=======
		name := strings.TrimSpace(j.Nom)
		if name == "" {
			continue
		}
		if _, err := stmt.Exec(roomCode, name, j.Total); err != nil {
>>>>>>> v1seb
			log.Println("PetitBac: snapshot joueur:", err)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Println("PetitBac: commit snapshot:", err)
	}
}

func fetchRoomPlayers(roomCode string) ([]dbPlayer, error) {
	if pbDB == nil {
		return nil, nil
	}
	rows, err := pbDB.Query(`SELECT pseudo, total_score FROM petitbac_players WHERE room_code = ? ORDER BY total_score DESC, pseudo ASC`, roomCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
<<<<<<< HEAD
	var results []dbPlayer
=======
	var players []dbPlayer
>>>>>>> v1seb
	for rows.Next() {
		var p dbPlayer
		if err := rows.Scan(&p.Pseudo, &p.Score); err != nil {
			return nil, err
		}
		p.Room = roomCode
<<<<<<< HEAD
		results = append(results, p)
	}
	return results, rows.Err()
=======
		players = append(players, p)
	}
	return players, rows.Err()
>>>>>>> v1seb
}

func isRoomHost(roomCode, pseudo string) bool {
	if pbDB == nil {
		return true
	}
	pseudo = strings.TrimSpace(pseudo)
	if pseudo == "" {
		return false
	}
	var host string
<<<<<<< HEAD
	err := pbDB.QueryRow(`SELECT host FROM petitbac_rooms WHERE code = ?`, roomCode).Scan(&host)
	if err != nil {
=======
	if err := pbDB.QueryRow(`SELECT host FROM petitbac_rooms WHERE code = ?`, roomCode).Scan(&host); err != nil {
>>>>>>> v1seb
		return false
	}
	return strings.EqualFold(strings.TrimSpace(host), pseudo)
}
