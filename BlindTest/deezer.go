package blindtest

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
)

func fetchTracksFromDeezer(playlist string, limit int) ([]Track, error) {
	if playlist == "generale" {
		return fetchMixedGenreTracks(limit)
	}

	if playlist == "francaise" {
		return fetchFrenchTracks(limit)
	}

	genreID := getGenreID(playlist)

	url := fmt.Sprintf("https://api.deezer.com/chart/%d/tracks?limit=100", genreID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			Title   string `json:"title"`
			Preview string `json:"preview"`
			Artist  struct {
				Name string `json:"name"`
			} `json:"artist"`
			Album struct {
				Title string `json:"title"`
			} `json:"album"`
			Duration int `json:"duration"`
		} `json:"data"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	tracks := make([]Track, 0)
	for _, item := range result.Data {
		if item.Preview != "" {
			tracks = append(tracks, Track{
				Title:    item.Title,
				Artist:   item.Artist.Name,
				Preview:  item.Preview,
				Album:    item.Album.Title,
				Duration: item.Duration,
			})
		}
	}

	rand.Shuffle(len(tracks), func(i, j int) {
		tracks[i], tracks[j] = tracks[j], tracks[i]
	})

	if len(tracks) > limit {
		tracks = tracks[:limit]
	}

	return tracks, nil
}

func fetchMixedGenreTracks(limit int) ([]Track, error) {
	allGenres := []string{"pop", "rock", "rap", "electronic", "indie", "classic", "country", "jazz", "blues", "reggae", "rnb", "soul", "metal", "alternative", "techno"}

	tracksPerGenre := 10
	allTracks := make([]Track, 0)

	for _, genre := range allGenres {
		tracks, err := fetchTracksFromGenre(genre, tracksPerGenre)
		if err == nil {
			allTracks = append(allTracks, tracks...)
		}
	}

	rand.Shuffle(len(allTracks), func(i, j int) {
		allTracks[i], allTracks[j] = allTracks[j], allTracks[i]
	})

	if len(allTracks) > limit {
		allTracks = allTracks[:limit]
	}

	return allTracks, nil
}

func fetchFrenchTracks(limit int) ([]Track, error) {
	frenchArtists := []string{
		"Stromae", "Angèle", "Orelsan", "Edith Piaf", "Charles Aznavour",
		"Indila", "Ninho", "Aya Nakamura", "Jul", "Soprano",
		"Jacques Brel", "Dalida", "Claude François", "Johnny Hallyday",
		"Zaz", "Céline Dion", "GIMS", "PNL", "Booba", "Nekfeu",
		"Damso", "SCH", "Naps", "Maître Gims", "Black M",
		"Louane", "Kendji Girac", "Vitaa", "Slimane", "Dadju",
	}

	allTracks := make([]Track, 0)
	tracksPerArtist := 5

	for _, artist := range frenchArtists {
		url := fmt.Sprintf("https://api.deezer.com/search?q=artist:\"%s\"&limit=%d", artist, tracksPerArtist)

		resp, err := http.Get(url)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var result struct {
			Data []struct {
				Title   string `json:"title"`
				Preview string `json:"preview"`
				Artist  struct {
					Name string `json:"name"`
				} `json:"artist"`
				Album struct {
					Title string `json:"title"`
				} `json:"album"`
				Duration int `json:"duration"`
			} `json:"data"`
		}

		err = json.Unmarshal(body, &result)
		if err != nil {
			continue
		}

		for _, item := range result.Data {
			if item.Preview != "" {
				allTracks = append(allTracks, Track{
					Title:    item.Title,
					Artist:   item.Artist.Name,
					Preview:  item.Preview,
					Album:    item.Album.Title,
					Duration: item.Duration,
				})
			}
		}

		if len(allTracks) >= limit*2 {
			break
		}
	}

	rand.Shuffle(len(allTracks), func(i, j int) {
		allTracks[i], allTracks[j] = allTracks[j], allTracks[i]
	})

	if len(allTracks) > limit {
		allTracks = allTracks[:limit]
	}

	return allTracks, nil
}

func fetchTracksFromGenre(genre string, limit int) ([]Track, error) {
	genreID := getGenreID(genre)
	if genreID == 0 {
		return []Track{}, nil
	}

	url := fmt.Sprintf("https://api.deezer.com/chart/%d/tracks?limit=%d", genreID, limit)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			Title   string `json:"title"`
			Preview string `json:"preview"`
			Artist  struct {
				Name string `json:"name"`
			} `json:"artist"`
			Album struct {
				Title string `json:"title"`
			} `json:"album"`
			Duration int `json:"duration"`
		} `json:"data"`
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	tracks := make([]Track, 0)
	for _, item := range result.Data {
		if item.Preview != "" {
			tracks = append(tracks, Track{
				Title:    item.Title,
				Artist:   item.Artist.Name,
				Preview:  item.Preview,
				Album:    item.Album.Title,
				Duration: item.Duration,
			})
		}
	}

	return tracks, nil
}

func getGenreID(playlist string) int {
	genreMap := map[string]int{
		"pop":         132,
		"rock":        152,
		"rap":         116,
		"electronic":  106,
		"indie":       85,
		"classic":     98,
		"country":     2,
		"jazz":        129,
		"blues":       153,
		"reggae":      144,
		"rnb":         165,
		"soul":        169,
		"metal":       464,
		"alternative": 85,
		"latin":       197,
		"techno":      140,
	}

	if id, exists := genreMap[playlist]; exists {
		return id
	}

	return 0
}
