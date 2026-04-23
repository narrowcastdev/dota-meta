package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// snapshotSummary is the minimal shape we need from a prior snapshot —
// (bracket, heroName) → WR/PR. Uses display-name key for stability across runs.
type snapshotSummary struct {
	Date  time.Time
	Patch string
	Stats map[string]map[string]struct{ WR, PR float64 }
}

// LoadLatestPriorSnapshot returns the newest snapshot in dir strictly older
// than today's date (date comparison, not time). Returns nil, nil if none
// exist.
func LoadLatestPriorSnapshot(dir string, today time.Time) (*snapshotSummary, error) {
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	type dated struct {
		date time.Time
		path string
	}
	var candidates []dated
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "data-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		datePart := strings.TrimSuffix(strings.TrimPrefix(name, "data-"), ".json")
		d, err := time.Parse("2006-01-02", datePart)
		if err != nil {
			continue
		}
		if !d.Before(todayDate) {
			continue
		}
		candidates = append(candidates, dated{d, filepath.Join(dir, name)})
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].date.After(candidates[j].date) })
	newest := candidates[0]
	data, err := os.ReadFile(newest.path)
	if err != nil {
		return nil, err
	}
	return parseSnapshot(newest.date, data)
}

func parseSnapshot(date time.Time, data []byte) (*snapshotSummary, error) {
	var shape struct {
		Patch    string `json:"patch"`
		Analysis struct {
			Brackets map[string]struct {
				Name  string `json:"name"`
				Cores []struct {
					Name     string  `json:"name"`
					WinRate  float64 `json:"win_rate"`
					PickRate float64 `json:"pick_rate"`
				} `json:"cores"`
				Supports []struct {
					Name     string  `json:"name"`
					WinRate  float64 `json:"win_rate"`
					PickRate float64 `json:"pick_rate"`
				} `json:"supports"`
			} `json:"brackets"`
		} `json:"analysis"`
	}
	if err := json.Unmarshal(data, &shape); err != nil {
		return nil, fmt.Errorf("parsing snapshot: %w", err)
	}
	out := &snapshotSummary{Date: date, Patch: shape.Patch, Stats: map[string]map[string]struct{ WR, PR float64 }{}}
	for _, b := range shape.Analysis.Brackets {
		m := map[string]struct{ WR, PR float64 }{}
		for _, s := range b.Cores {
			m[s.Name] = struct{ WR, PR float64 }{s.WinRate, s.PickRate}
		}
		for _, s := range b.Supports {
			m[s.Name] = struct{ WR, PR float64 }{s.WinRate, s.PickRate}
		}
		out.Stats[b.Name] = m
	}
	return out, nil
}

// ApplyDeltas fills WRDelta/PRDelta on every HeroStat in the analysis using
// the given prior snapshot. Does nothing if prior is nil.
func ApplyDeltas(full *FullAnalysis, prior *snapshotSummary) {
	if prior == nil {
		return
	}
	full.PriorSnapshot = &prior.Date
	for bi := range full.Brackets {
		apply := func(stats []HeroStat, bracketName string) {
			bracketMap, ok := prior.Stats[bracketName]
			if !ok {
				return
			}
			for si := range stats {
				old, ok := bracketMap[stats[si].Hero.DisplayName]
				if !ok {
					continue
				}
				wrD := stats[si].WinRate - old.WR
				prD := stats[si].PickRate - old.PR
				stats[si].WRDelta = &wrD
				stats[si].PRDelta = &prD
			}
		}
		apply(full.Brackets[bi].Cores, full.Brackets[bi].Bracket.Name)
		apply(full.Brackets[bi].Supports, full.Brackets[bi].Bracket.Name)
	}
}
