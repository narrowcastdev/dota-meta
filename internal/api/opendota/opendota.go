package opendota

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const heroStatsURL = "https://api.opendota.com/api/heroStats"

// Hero represents a single hero's stats from the OpenDota API.
type Hero struct {
	ID            int      `json:"id"`
	LocalizedName string   `json:"localized_name"`
	PrimaryAttr   string   `json:"primary_attr"`
	AttackType    string   `json:"attack_type"`
	Roles         []string `json:"roles"`
	Img           string   `json:"img"`
	BracketPick   [8]int   // Indexed 0-7 for brackets 1-8
	BracketWin    [8]int   // Indexed 0-7 for brackets 1-8
}

// WinRate returns the win rate percentage for a given bracket index (0-7).
func (h Hero) WinRate(bracket int) float64 {
	if h.BracketPick[bracket] == 0 {
		return 0
	}
	return float64(h.BracketWin[bracket]) / float64(h.BracketPick[bracket]) * 100
}

// PickRate returns the pick rate percentage for a given bracket index (0-7)
// relative to totalPicks in that bracket.
func (h Hero) PickRate(bracket int, totalPicks int) float64 {
	if totalPicks == 0 {
		return 0
	}
	return float64(h.BracketPick[bracket]) / float64(totalPicks) * 100
}

// FetchHeroStats fetches hero statistics from the OpenDota API.
func FetchHeroStats() ([]Hero, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(heroStatsURL)
	if err != nil {
		return nil, fmt.Errorf("fetching hero stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("OpenDota API returned %d: %s", resp.StatusCode, string(body))
	}

	return ParseHeroStats(resp.Body)
}

// ParseHeroStats parses hero stats JSON from a reader.
// Separated from FetchHeroStats for testability.
func ParseHeroStats(r io.Reader) ([]Hero, error) {
	var raw []map[string]json.RawMessage
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding hero stats: %w", err)
	}

	heroes := make([]Hero, 0, len(raw))
	for _, entry := range raw {
		var hero Hero
		if v, ok := entry["id"]; ok {
			json.Unmarshal(v, &hero.ID)
		}
		if v, ok := entry["localized_name"]; ok {
			json.Unmarshal(v, &hero.LocalizedName)
		}
		if v, ok := entry["primary_attr"]; ok {
			json.Unmarshal(v, &hero.PrimaryAttr)
		}
		if v, ok := entry["attack_type"]; ok {
			json.Unmarshal(v, &hero.AttackType)
		}
		if v, ok := entry["roles"]; ok {
			json.Unmarshal(v, &hero.Roles)
		}
		if v, ok := entry["img"]; ok {
			json.Unmarshal(v, &hero.Img)
		}

		for b := 1; b <= 8; b++ {
			pickKey := fmt.Sprintf("%d_pick", b)
			winKey := fmt.Sprintf("%d_win", b)
			if v, ok := entry[pickKey]; ok {
				json.Unmarshal(v, &hero.BracketPick[b-1])
			}
			if v, ok := entry[winKey]; ok {
				json.Unmarshal(v, &hero.BracketWin[b-1])
			}
		}

		heroes = append(heroes, hero)
	}

	if len(heroes) == 0 {
		return nil, fmt.Errorf("no hero data returned — API may be temporarily unavailable")
	}

	return heroes, nil
}
