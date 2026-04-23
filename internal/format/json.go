package format

import (
	"encoding/json"
	"time"

	"github.com/narrowcastdev/dota-meta/internal/analysis"
	"github.com/narrowcastdev/dota-meta/internal/api/stratz"
)

type jsonOutput struct {
	Generated        string       `json:"generated"`
	Patch            string       `json:"patch,omitempty"`
	SnapshotDate     string       `json:"snapshot_date,omitempty"`
	PriorSnapshot    string       `json:"prior_snapshot,omitempty"`
	SnapshotsInPatch int          `json:"snapshots_in_patch"`
	Heroes           []jsonHero   `json:"heroes"`
	Analysis         jsonAnalysis `json:"analysis"`
}

type jsonHero struct {
	Name        string                     `json:"name"`
	ShortName   string                     `json:"short_name"`
	PrimaryAttr string                     `json:"primary_attr"`
	AttackType  string                     `json:"attack_type"`
	Roles       []string                   `json:"roles"`
	Brackets    map[string]jsonBracketStat `json:"brackets"`
}

type jsonBracketStat struct {
	Picks    int     `json:"picks"`
	Wins     int     `json:"wins"`
	WinRate  float64 `json:"win_rate"`
	PickRate float64 `json:"pick_rate"`
}

type jsonAnalysis struct {
	Brackets map[string]jsonBracketAnalysis `json:"brackets"`
}

type jsonBracketAnalysis struct {
	Name     string         `json:"name"`
	Cores    []jsonHeroStat `json:"cores"`
	Supports []jsonHeroStat `json:"supports"`
}

type jsonHeroStat struct {
	Name      string    `json:"name"`
	ShortName string    `json:"short_name"`
	Tier      string    `json:"tier"`
	Climb     string    `json:"climb,omitempty"`
	WinRate   float64   `json:"win_rate"`
	PickRate  float64   `json:"pick_rate"`
	Picks     int       `json:"picks"`
	WRDelta   *float64  `json:"wr_delta,omitempty"`
	PRDelta   *float64  `json:"pr_delta,omitempty"`
	Momentum  string    `json:"momentum,omitempty"`
	WRHistory []float64 `json:"wr_history,omitempty"`
}

var bracketKeyMap = map[string]string{
	"Herald-Guardian": "herald_guardian",
	"Crusader-Archon": "crusader_archon",
	"Legend-Ancient":  "legend_ancient",
	"Divine":          "divine",
	"Immortal":        "immortal",
}

// FormatJSON generates a JSON output suitable for the static site.
func FormatJSON(heroes []stratz.Hero, result analysis.FullAnalysis) ([]byte, error) {
	output := jsonOutput{
		Generated:        time.Now().UTC().Format(time.RFC3339),
		Patch:            result.Patch,
		SnapshotDate:     result.SnapshotDate.Format("2006-01-02"),
		SnapshotsInPatch: result.SnapshotsInPatch,
		Heroes:           buildJSONHeroes(heroes, result),
		Analysis:         buildJSONAnalysis(result),
	}
	if result.PriorSnapshot != nil {
		output.PriorSnapshot = result.PriorSnapshot.Format("2006-01-02")
	}
	return json.MarshalIndent(output, "", "  ")
}

func buildJSONHeroes(heroes []stratz.Hero, result analysis.FullAnalysis) []jsonHero {
	statLookup := map[int]map[string]jsonBracketStat{}
	for _, ba := range result.Brackets {
		key := bracketKeyMap[ba.Bracket.Name]
		add := func(stats []analysis.HeroStat) {
			for _, s := range stats {
				m, ok := statLookup[s.Hero.ID]
				if !ok {
					m = map[string]jsonBracketStat{}
					statLookup[s.Hero.ID] = m
				}
				m[key] = jsonBracketStat{
					Picks:    s.Picks,
					Wins:     s.Wins,
					WinRate:  s.WinRate,
					PickRate: s.PickRate,
				}
			}
		}
		add(ba.Cores)
		add(ba.Supports)
	}
	out := make([]jsonHero, 0, len(heroes))
	for _, h := range heroes {
		out = append(out, jsonHero{
			Name:        h.DisplayName,
			ShortName:   h.ShortName,
			PrimaryAttr: h.PrimaryAttribute,
			AttackType:  h.AttackType,
			Roles:       h.Roles,
			Brackets:    statLookup[h.ID],
		})
	}
	return out
}

func buildJSONAnalysis(result analysis.FullAnalysis) jsonAnalysis {
	ja := jsonAnalysis{Brackets: make(map[string]jsonBracketAnalysis, len(result.Brackets))}
	for _, ba := range result.Brackets {
		key := bracketKeyMap[ba.Bracket.Name]
		ja.Brackets[key] = jsonBracketAnalysis{
			Name:     ba.Bracket.Name,
			Cores:    heroStatsToJSON(ba.Cores),
			Supports: heroStatsToJSON(ba.Supports),
		}
	}
	return ja
}

func heroStatsToJSON(stats []analysis.HeroStat) []jsonHeroStat {
	out := make([]jsonHeroStat, 0, len(stats))
	for _, s := range stats {
		js := jsonHeroStat{
			Name:      s.Hero.DisplayName,
			ShortName: s.Hero.ShortName,
			Tier:      s.Tier.String(),
			WinRate:   s.WinRate,
			PickRate:  s.PickRate,
			Picks:     s.Picks,
			WRDelta:   s.WRDelta,
			PRDelta:   s.PRDelta,
			WRHistory: s.WRHistory,
		}
		if s.ClimbTag != analysis.ClimbUnknown {
			js.Climb = string(s.ClimbTag)
		}
		if s.Momentum != analysis.MomentumNone {
			js.Momentum = string(s.Momentum)
		}
		out = append(out, js)
	}
	return out
}
