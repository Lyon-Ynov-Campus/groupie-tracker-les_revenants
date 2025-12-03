package rooms

type Room struct {
	ID        string
	Code      string
	GameType  string
	Status    string
	PlayerIDs []int
	MaxRounds int
	Config    map[string]any
}

func IsRoomReady(r Room) bool {
	if len(r.PlayerIDs) < 2 {
		return false
	}

	if r.GameType != "blindtest" {
		return false
	}

	_, ok := r.Config["playlist"]
	return ok
}

func ValidPlaylist(p string) bool {
	switch p {
	case "Rock", "Pop", "Jazz", "Classical", "HipHop":
		return true
	default:
		return false
	}
}
