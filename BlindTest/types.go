package blindtest

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	ID       string
	Username string
	Conn     *websocket.Conn
	Score    int
	Ready    bool
}

type Track struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Preview  string `json:"preview"`
	Album    string `json:"album"`
	Duration int    `json:"duration"`
}

type PlayerAnswer struct {
	FoundTitle  bool
	FoundArtist bool
	TimeTitle   time.Time
	TimeArtist  time.Time
}

type Room struct {
	ID              string
	Players         map[string]*Player
	CurrentTrack    *Track
	RoundNumber     int
	GameStarted     bool
	RoundStartTime  time.Time
	CorrectAnswers  map[string]bool
	PlayerAnswers   map[string]*PlayerAnswer
	Tracks          []Track
	CurrentTrackIdx int
	MaxRounds       int
	RoundTime       int
	Playlist        string
	mu              sync.RWMutex
}

type Message struct {
	Type      string                 `json:"type"`
	Username  string                 `json:"username,omitempty"`
	RoomID    string                 `json:"roomId,omitempty"`
	Answer    string                 `json:"answer,omitempty"`
	Playlist  string                 `json:"playlist,omitempty"`
	MaxRounds int                    `json:"maxRounds,omitempty"`
	RoundTime int                    `json:"roundTime,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

var (
	rooms   = make(map[string]*Room)
	roomsMu sync.RWMutex
)
