package blindtest

import "time"

func handleAnswer(room *Room, player *Player, answer string) {
	room.mu.Lock()
	defer room.mu.Unlock()

	if room.CorrectAnswers[player.ID] {
		return
	}

	if !room.GameStarted || room.CurrentTrack == nil {
		return
	}

	normalizedAnswer := normalizeString(answer)
	normalizedTitle := normalizeString(room.CurrentTrack.Title)
	normalizedArtist := normalizeString(room.CurrentTrack.Artist)

	foundTitle := contains(normalizedAnswer, normalizedTitle) || contains(normalizedTitle, normalizedAnswer)
	foundArtist := contains(normalizedAnswer, normalizedArtist) || contains(normalizedArtist, normalizedAnswer)

	if !foundTitle && !foundArtist {
		return
	}

	if room.PlayerAnswers[player.ID] == nil {
		room.PlayerAnswers[player.ID] = &PlayerAnswer{}
	}

	playerAnswer := room.PlayerAnswers[player.ID]
	hadBothBefore := playerAnswer.FoundTitle && playerAnswer.FoundArtist

	if foundTitle && !playerAnswer.FoundTitle {
		playerAnswer.FoundTitle = true
		playerAnswer.TimeTitle = time.Now()
	}

	if foundArtist && !playerAnswer.FoundArtist {
		playerAnswer.FoundArtist = true
		playerAnswer.TimeArtist = time.Now()
	}

	hasBothNow := playerAnswer.FoundTitle && playerAnswer.FoundArtist

	var points int
	var color string
	var answerType string

	if hasBothNow && !hadBothBefore {
		room.CorrectAnswers[player.ID] = true

		var elapsed float64
		if foundTitle && foundArtist {
			elapsed = time.Since(room.RoundStartTime).Seconds()
			answerType = "both"
		} else if foundTitle {
			elapsed = time.Since(room.RoundStartTime).Seconds()
			answerType = "title_completing"
		} else {
			elapsed = time.Since(room.RoundStartTime).Seconds()
			answerType = "artist_completing"
		}

		points = calculatePoints(elapsed)
		player.Score += points
		color = "green"

		for _, p := range room.Players {
			p.Conn.WriteJSON(Message{
				Type: "correct_answer",
				Data: map[string]interface{}{
					"username":    player.Username,
					"points":      points,
					"answerType":  answerType,
					"color":       color,
					"foundTitle":  playerAnswer.FoundTitle,
					"foundArtist": playerAnswer.FoundArtist,
				},
			})
		}
	} else if !hadBothBefore {
		elapsed := time.Since(room.RoundStartTime).Seconds()
		points = calculatePoints(elapsed) / 2

		player.Score += points

		if foundTitle {
			answerType = "title_partial"
			color = "orange"
		} else {
			answerType = "artist_partial"
			color = "orange"
		}

		for _, p := range room.Players {
			p.Conn.WriteJSON(Message{
				Type: "correct_answer",
				Data: map[string]interface{}{
					"username":    player.Username,
					"points":      points,
					"answerType":  answerType,
					"color":       color,
					"foundTitle":  playerAnswer.FoundTitle,
					"foundArtist": playerAnswer.FoundArtist,
				},
			})
		}
	}
}

func normalizeString(s string) string {
	result := ""
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if r >= 'A' && r <= 'Z' {
				result += string(r + 32)
			} else {
				result += string(r)
			}
		}
	}
	return result
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func calculatePoints(elapsed float64) int {
	if elapsed < 5 {
		return 1000
	} else if elapsed < 10 {
		return 800
	} else if elapsed < 15 {
		return 600
	} else if elapsed < 20 {
		return 400
	} else if elapsed < 25 {
		return 200
	}
	return 100
}
