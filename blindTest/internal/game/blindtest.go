package game

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"math/rand"
	"strings"
	"time"

	"blindTest/internal/ws"
)

// Configuration
type BlindTestConfig struct {
	Playlist        string `json:"playlist"`
	ResponseTimeSec int    `json:"response_time_sec"`
	MaxRounds       int    `json:"max_rounds"`
}

// Track metadata (Spotify-like)
type Track struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	PreviewURL string `json:"preview_url"`
}

// Round state
type BlindTestRound struct {
	RoundIndex int
	Track      Track
	StartTime  time.Time
	EndTime    time.Time
	// userID -> answer
	Answers map[int]string
	// received correct answers in order
	CorrectOrder []int
}

type BlindTestEngine struct {
	db     *sql.DB
	hub    *ws.Hub
	gameID int
	cfg    BlindTestConfig
	// scoreboardActualPointInGame ref
	scoreboard map[int]int
}

func NewBlindTestEngine(db *sql.DB, hub *ws.Hub, gameID int, cfg BlindTestConfig, scoreboard map[int]int) *BlindTestEngine {
	return &BlindTestEngine{
		db:         db,
		hub:        hub,
		gameID:     gameID,
		cfg:        cfg,
		scoreboard: scoreboard,
	}
}

// Create game row
func CreateGame(db *sql.DB, roomID string, cfg BlindTestConfig) error {
	cfgJSON, _ := json.Marshal(cfg)
	_, err := db.Exec(`INSERT INTO games(room_id, config_json, max_rounds, current_round, started_at)
VALUES(?, ?, ?, 0, NULL)`, roomID, string(cfgJSON), cfg.MaxRounds)
	return err
}

func GetConfig(db *sql.DB, gameID int) (BlindTestConfig, error) {
	var raw string
	err := db.QueryRow(`SELECT config_json FROM games WHERE id=?`, gameID).Scan(&raw)
	if err != nil {
		return BlindTestConfig{}, err
	}
	var cfg BlindTestConfig
	_ = json.Unmarshal([]byte(raw), &cfg)
	if cfg.ResponseTimeSec == 0 {
		cfg.ResponseTimeSec = 37
	}
	return cfg, nil
}

func (bt *BlindTestEngine) Run(ctx context.Context) {
	// Mark start
	_, _ = bt.db.Exec(`UPDATE games SET started_at=? WHERE id=?`, time.Now(), bt.gameID)

	for round := 0; round < bt.cfg.MaxRounds; round++ {
		_ = bt.db.Exec(`UPDATE games SET current_round = ? WHERE id = ?`, round+1, bt.gameID)
		r := bt.startRound(round)

		bt.hub.BroadcastJSON(ws.Message{
			Type: "round.start",
			Payload: map[string]any{
				"round":       r.RoundIndex,
				"previewUrl":  r.Track.PreviewURL,
				"responseSec": bt.cfg.ResponseTimeSec,
			},
		})

		deadline := r.StartTime.Add(time.Duration(bt.cfg.ResponseTimeSec) * time.Second)
		timer := time.NewTimer(time.Until(deadline))
	recvLoop:
		for {
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				break recvLoop
			case in := <-bt.hub.Input:
				// Expect answer.submit
				if in.Type == "answer.submit" {
					uid := int(in.UserID)
					ans := strings.TrimSpace(strings.ToLower(in.Payload["answer"].(string)))
					r.Answers[uid] = ans
					// Check correctness
					title := strings.TrimSpace(strings.ToLower(r.Track.Title))
					if isCorrect(title, ans) {
						// avoid duplicates
						if !contains(r.CorrectOrder, uid) {
							r.CorrectOrder = append(r.CorrectOrder, uid)
							bt.hub.BroadcastJSON(ws.Message{
								Type:    "answer.update",
								Payload: map[string]any{"userID": uid, "status": "correct"},
							})
						}
					} else {
						bt.hub.BroadcastJSON(ws.Message{
							Type:    "answer.update",
							Payload: map[string]any{"userID": uid, "status": "wrong"},
						})
					}
				}
			}
		}

		// Score distribution
		points := scoreBlindTest(r.CorrectOrder)
		for uid, p := range points {
			bt.scoreboard[uid] += p
			// persist
			_, _ = bt.db.Exec(`INSERT INTO scores(game_id, user_id, points) VALUES (?, ?, ?)`, bt.gameID, uid, p)
			bt.hub.BroadcastJSON(ws.Message{
				Type: "score.update",
				Payload: map[string]any{
					"userID": uid,
					"points": p,
					"total":  bt.scoreboard[uid],
				},
			})
		}

		bt.hub.BroadcastJSON(ws.Message{Type: "round.end", Payload: map[string]any{"round": r.RoundIndex}})

		// Small pause between rounds
		time.Sleep(2 * time.Second)
	}

	// End game
	_, _ = bt.db.Exec(`UPDATE rooms SET status='finished' WHERE id=(SELECT room_id FROM games WHERE id=?)`, bt.gameID)
	bt.hub.BroadcastJSON(ws.Message{
		Type: "scoreboard.show",
		Payload: map[string]any{
			"scores": bt.scoreboard,
		},
	})
}

func (bt *BlindTestEngine) startRound(index int) *BlindTestRound {
	track := pickRandomTrack(bt.cfg.Playlist)
	r := &BlindTestRound{
		RoundIndex:   index + 1,
		Track:        track,
		StartTime:    time.Now(),
		Answers:      map[int]string{},
		CorrectOrder: []int{},
	}
	// Persist round payload
	payload, _ := json.Marshal(track)
	_, _ = bt.db.Exec(`INSERT INTO rounds(game_id, round_index, payload_json, started_at)
VALUES (?, ?, ?, ?)`, bt.gameID, r.RoundIndex, string(payload), r.StartTime)
	return r
}

// --- Utilities ---

func scoreBlindTest(correctUserIDs []int) map[int]int {
	base := 5 // premier correct = 5 pts, puis 4, 3, 2, 1
	pts := map[int]int{}
	for i, uid := range correctUserIDs {
		pts[uid] = int(math.Max(float64(base-i), 1))
	}
	return pts
}

func contains(arr []int, v int) bool {
	for _, x := range arr {
		if x == v {
			return true
		}
	}
	return false
}

func isCorrect(correctTitle, answer string) bool {
	// Tolérance simple: égalité après normalisation, ou answer contenue dans le titre
	ct := normalize(correctTitle)
	a := normalize(answer)
	return ct == a || strings.Contains(ct, a)
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "(", " ")
	s = strings.ReplaceAll(s, ")", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

// Dummy picker (à remplacer par Spotify API)
func pickRandomTrack(playlist string) Track {
	tracks := []Track{
		{ID: "1", Title: "Smells Like Teen Spirit", Artist: "Nirvana", PreviewURL: "https://p.scdn.co/mp3-preview/demo1.mp3"},
		{ID: "2", Title: "Lose Yourself", Artist: "Eminem", PreviewURL: "https://p.scdn.co/mp3-preview/demo2.mp3"},
		{ID: "3", Title: "Billie Jean", Artist: "Michael Jackson", PreviewURL: "https://p.scdn.co/mp3-preview/demo3.mp3"},
		{ID: "4", Title: "Boulevard of Broken Dreams", Artist: "Green Day", PreviewURL: "https://p.scdn.co/mp3-preview/demo4.mp3"},
	}
	// Simple filter by playlist hint
	idx := rand.Intn(len(tracks))
	return tracks[idx]
}
