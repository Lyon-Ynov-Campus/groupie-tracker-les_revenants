package game

import (
	"blindTest/internal/ws"
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"math/rand"
	"strings"
	"time"
)

type BlindTestConfig struct {
	Playlist        string `json:"playlist"`
	ResponseTimeSec int    `json:"response_time_sec"`
	MaxRounds       int    `json:"max_rounds"`
}

type Track struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	PreviewURL string `json:"preview_url"`
}

type BlindTestRound struct {
	RoundIndex   int
	Track        Track
	StartTime    time.Time
	EndTime      time.Time
	Answers      map[int]string
	CorrectOrder []int
}

type BlindeTestEngine struct {
	db         *sql.DB
	hub        *ws.Hub
	gameID     int
	cfg        BlindTestConfig
	scoreboard map[int]int
}

func NewBlindTestEngine(db *sql.DB, hub *ws.Hub, gameID int, cfg BlindTestConfig, scoreboard map[int]int) *BlindeTestEngine {
	return &BlindeTestEngine{
		db:         db,
		hub:        hub,
		gameID:     gameID,
		cfg:        cfg,
		scoreboard: scoreboard,
	}
}

func CreateGame(db *sql.DB, roomID string, cfg BlindTestConfig) error {
	cfgJSON, _ := json.Marshal(cfg)
	_, err := db.Exec(`INSERT INTO games(room_id, config_json, max_rounds, current_round, started_at)
VALUES(?, ?, ?, 0, NULL)`, roomID, string(cfgJSON), cfg.MaxRounds)
	return err
}

func GetConfig(db *sql.DB, gameID int) (BlindTestConfig, error) {
	var raw string
	err := db.QueryRow(`SELECT config_json FROM games WHERE id = ?`, gameID).Scan(&raw)
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

func (bt *BlindeTestEngine) Run(ctx context.Context) {
	_, _ = bt.db.Exec(`UPDATE games SET started_at = ? WHERE id = ?`, time.Now(), bt.gameID)
	for round := 0; round < bt.cfg.MaxRounds; round++ {
		_ = bt.db.Exec(`UPDATE games SET current_round = ? WHERE id = ?`, round+1, bt.gameID)
		r := bt.startRound(round)

		bt.hub.BroadcastJSON(ws.Message{
			Type: "round.start",
			Payload: map[string]any{
				"round":       r.RoundIndex,
				"previewURL":  r.Track.PreviewURL,
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
				if in.Type == "answer.submit" {
					uid := int(in.UserID)
					ans := strings.TrimSpace(strings.ToLower(in.Payload["answer"].(string)))
					r.Answers[uid] = ans
					title := strings.TrimSpace(strings.ToLower(r.Track.Title))
					if isCorrect(title, ans) {
						if !contains(r.CorrectOrder, uid) {
							r.CorrectOrder = append(r.CorrectOrder, uid)
							bt.hub.BroadcastJSON(ws.Message{
								Type: "answer.update",
								Payload: map[string]any{
									"userID": uid,
									"status": "correct",
								},
							})
						}
					} else {
						bt.hub.BroadcastJSON(ws.Message{
							Type: "answer.update",
							Payload: map[string]any{
								"userID": uid,
								"status": "incorrect",
							},
						})
					}
				}
			}
		}

		points := scoreBlindTest(r.CorrectOrder)
		for uid, pts := range points {
			bt.scoreboard[uid] += pts
			_, _ = bt.db.Exec(`INSERT INTO scores(game_id, user_id, points) VALUES (?, ?, ?)`, bt.gameID, uid, pts)
			bt.hubBroadcastJSON(ws.Message{
				Type: "score.update",
				Payload: map[string]any{
					"userID": uid,
					"points": pts,
					"total":  bt.scoreboard[uid],
				},
			})
		}

		bt.hub.BroadcastJSON(ws.Message{
			Type: "round.end",
			Payload: map[string]any{
				"round": r.RoundIndex,
			}})

		time.Sleep(2 * time.Second)
	}

	_, _ = bt.db.Exec(`UPDATE rooms SET status = 'finished' WHERE id = (SELECT room_id FROM games WHERE id = ?)`, bt.gameID)
	bt.hub.BroadcastJSON(ws.Message{
		Type: "scoreboard.show",
		Payload: map[string]any{
			"scores": bt.scoreboard,
		},
	})
}

func (bt *BlindeTestEngine) startRound(index int) *BlindTestRound {
	track := pickRandomTrack(bt.cfg.Playlist)
	r := &BlindTestRound{
		RoundIndex:   index + 1,
		Track:        track,
		StartTime:    time.Now(),
		Answers:      map[int]string{},
		CorrectOrder: []int{},
	}
	payload, _ := json.Marshal(r.Track)
	_, _ = bt.db.Exec(`INSERT INTO rounds(game_id, round_index, payload_json, started_at) VALUES (?, ?, ?, ?)`,
		bt.gameID, r.RoundIndex, string(payload), r.StartTime)
	return r
}

func scoreBlindTest(correctOrder []int) map[int]int {
	base := 5
	points := map[int]int{}
	for i, uid := range correctUserIDs {
		p[uid] = int(math.Max(float64(base-i), 1))
	}
	return p
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
	s = strings.ReplaceAll(s, "'", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func pickRandomTrack(playlist string) Track {
	tracks := []Track{
		{ID: "1", Title: "Song A", Artist: "Artist 1", PreviewURL: "https://example.com/preview1.mp3"},
		{ID: "2", Title: "Song B", Artist: "Artist 2", PreviewURL: "https://example.com/preview2.mp3"},
		{ID: "3", Title: "Song C", Artist: "Artist 3", PreviewURL: "https://example.com/preview3.mp3"},
	}
	idx := rand.Intn(len(tracks))
	return tracks[idx]
}
