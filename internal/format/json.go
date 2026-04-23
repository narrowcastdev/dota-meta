package format

import (
	"encoding/json"
	"time"

	"github.com/narrowcastdev/dota-meta/internal/analysis"
	"github.com/narrowcastdev/dota-meta/internal/api/opendota"
)

type jsonOutput struct {
	Generated string       `json:"generated"`
	Heroes    []jsonHero   `json:"heroes"`
	Analysis  jsonAnalysis `json:"analysis"`
}

type jsonHero struct {
	Name        string                     `json:"name"`
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
	Delta    jsonDelta                      `json:"delta"`
}

type jsonBracketAnalysis struct {
	Best     []jsonHeroStat `json:"best"`
	Sleepers []jsonHeroStat `json:"sleepers"`
	Traps    []jsonHeroStat `json:"traps"`
	Supports []jsonHeroStat `json:"supports"`
}

type jsonHeroStat struct {
	Name     string  `json:"name"`
	WinRate  float64 `json:"win_rate"`
	PickRate float64 `json:"pick_rate"`
	Picks    int     `json:"picks"`
}

type jsonDelta struct {
	LowStompers      []jsonDeltaHero `json:"low_stompers"`
	HighSkillCeiling []jsonDeltaHero `json:"high_skill_ceiling"`
}

type jsonDeltaHero struct {
	Name   string  `json:"name"`
	LowWR  float64 `json:"low_wr"`
	HighWR float64 `json:"high_wr"`
	Delta  float64 `json:"delta"`
}

var bracketKeyMap = map[string]string{
	"Herald-Guardian": "herald_guardian",
	"Crusader-Archon": "crusader_archon",
	"Legend-Ancient":  "legend_ancient",
	"Divine":          "divine",
}

// FormatJSON generates a JSON output suitable for the static site.
func FormatJSON(heroes []opendota.Hero, result analysis.FullAnalysis) ([]byte, error) {
	bracketMatches := make(map[string]int, len(result.Brackets))
	for _, ba := range result.Brackets {
		bracketMatches[ba.Bracket.Name] = ba.Matches()
	}

	output := jsonOutput{
		Generated: time.Now().UTC().Format(time.RFC3339),
		Heroes:    buildJSONHeroes(heroes, bracketMatches),
		Analysis:  buildJSONAnalysis(result),
	}

	return json.MarshalIndent(output, "", "  ")
}

func buildJSONHeroes(heroes []opendota.Hero, bracketMatches map[string]int) []jsonHero {
	out := make([]jsonHero, 0, len(heroes))
	for _, h := range heroes {
		jh := jsonHero{
			Name:        h.LocalizedName,
			PrimaryAttr: h.PrimaryAttr,
			AttackType:  h.AttackType,
			Roles:       h.Roles,
			Brackets:    make(map[string]jsonBracketStat),
		}

		for _, bracket := range analysis.Brackets {
			key := bracketKeyMap[bracket.Name]
			var picks, wins int
			for _, i := range bracket.Indices {
				picks += h.BracketPick[i]
				wins += h.BracketWin[i]
			}

			var wr, pr float64
			if picks > 0 {
				wr = float64(wins) / float64(picks) * 100
			}
			if matches := bracketMatches[bracket.Name]; matches > 0 {
				pr = float64(picks) / float64(matches) * 100
			}

			jh.Brackets[key] = jsonBracketStat{
				Picks:    picks,
				Wins:     wins,
				WinRate:  wr,
				PickRate: pr,
			}
		}

		out = append(out, jh)
	}
	return out
}

func buildJSONAnalysis(result analysis.FullAnalysis) jsonAnalysis {
	ja := jsonAnalysis{
		Brackets: make(map[string]jsonBracketAnalysis),
	}

	for _, ba := range result.Brackets {
		key := bracketKeyMap[ba.Bracket.Name]
		ja.Brackets[key] = jsonBracketAnalysis{
			Best:     heroStatsToJSON(ba.Best),
			Sleepers: heroStatsToJSON(ba.Sleepers),
			Traps:    heroStatsToJSON(ba.Traps),
			Supports: heroStatsToJSON(ba.Supports),
		}
	}

	ja.Delta = jsonDelta{
		LowStompers:      deltaHeroesToJSON(result.LowStompers),
		HighSkillCeiling: deltaHeroesToJSON(result.HighSkillCap),
	}

	return ja
}

func heroStatsToJSON(stats []analysis.HeroStat) []jsonHeroStat {
	out := make([]jsonHeroStat, 0, len(stats))
	for _, s := range stats {
		out = append(out, jsonHeroStat{
			Name:     s.Hero.LocalizedName,
			WinRate:  s.WinRate,
			PickRate: s.PickRate,
			Picks:    s.Picks,
		})
	}
	return out
}

func deltaHeroesToJSON(deltas []analysis.DeltaHero) []jsonDeltaHero {
	out := make([]jsonDeltaHero, 0, len(deltas))
	for _, d := range deltas {
		out = append(out, jsonDeltaHero{
			Name:   d.Hero.LocalizedName,
			LowWR:  d.LowWR,
			HighWR: d.HighWR,
			Delta:  d.Delta,
		})
	}
	return out
}
