package analysis

import (
	"time"

	"github.com/narrowcastdev/dota-meta/internal/api/stratz"
)

// picksPerMatch is how many hero picks count per match (5 per team x 2).
const picksPerMatch = 10

// Bracket is an analysis bucket — one or more STRATZ ranks grouped together.
type Bracket struct {
	Name   string
	Stratz []stratz.Bracket
}

// Brackets defines the 5 analysis buckets. Immortal is now its own bucket
// (STRATZ separates it from Divine, unlike OpenDota).
var Brackets = []Bracket{
	{Name: "Herald-Guardian", Stratz: []stratz.Bracket{stratz.BracketHerald, stratz.BracketGuardian}},
	{Name: "Crusader-Archon", Stratz: []stratz.Bracket{stratz.BracketCrusader, stratz.BracketArchon}},
	{Name: "Legend-Ancient", Stratz: []stratz.Bracket{stratz.BracketLegend, stratz.BracketAncient}},
	{Name: "Divine", Stratz: []stratz.Bracket{stratz.BracketDivine}},
	{Name: "Immortal", Stratz: []stratz.Bracket{stratz.BracketImmortal}},
}

// HeroStat holds computed stats for one hero in one bracket.
type HeroStat struct {
	Hero      stratz.Hero
	WinRate   float64
	PickRate  float64
	Picks     int
	Wins      int
	Tier      Tier
	WRHistory []float64
}

// BracketAnalysis holds all analysis results for one bracket.
type BracketAnalysis struct {
	Bracket    Bracket
	Cores      []HeroStat
	Supports   []HeroStat
	TotalPicks int
}

// Matches returns the approximate match count for this bracket.
func (ba BracketAnalysis) Matches() int {
	return ba.TotalPicks / picksPerMatch
}

// DeltaHero holds bracket delta data for one hero.
type DeltaHero struct {
	Hero   stratz.Hero
	LowWR  float64
	HighWR float64
	Delta  float64
}

// FullAnalysis holds the complete analysis output.
type FullAnalysis struct {
	Brackets         []BracketAnalysis
	TotalMatches     int
	Patch            string
	SnapshotDate     time.Time
	PriorSnapshot    *time.Time
	SnapshotsInPatch int
}

// Analyze builds a FullAnalysis from STRATZ hero catalog + per-STRATZ-bracket
// weekly responses. Missing brackets produce empty buckets (no panic).
func Analyze(heroes []stratz.Hero, responses []stratz.BracketResponse, minPicks int) FullAnalysis {
	byID := make(map[int]stratz.Hero, len(heroes))
	for _, h := range heroes {
		byID[h.ID] = h
	}

	respByBracket := make(map[stratz.Bracket]stratz.BracketResponse, len(responses))
	for _, r := range responses {
		respByBracket[r.Bracket] = r
	}

	var full FullAnalysis
	for _, b := range Brackets {
		bucketResps := make([]stratz.BracketResponse, 0, len(b.Stratz))
		for _, sb := range b.Stratz {
			if r, ok := respByBracket[sb]; ok {
				bucketResps = append(bucketResps, r)
			}
		}
		agg := stratz.AggregateBrackets(bucketResps)
		ba := analyzeBracket(b, byID, agg, minPicks)
		full.Brackets = append(full.Brackets, ba)
		full.TotalMatches += ba.Matches()
	}
	return full
}

func analyzeBracket(bracket Bracket, byID map[int]stratz.Hero, agg map[int]stratz.AggregatedHero, minPicks int) BracketAnalysis {
	var totalPicks int
	for _, h := range agg {
		totalPicks += h.Picks
	}
	matches := totalPicks / picksPerMatch
	ba := BracketAnalysis{Bracket: bracket, TotalPicks: totalPicks}
	for id, a := range agg {
		if a.Picks < minPicks {
			continue
		}
		hero, ok := byID[id]
		if !ok {
			continue
		}
		wr := 0.0
		if a.Picks > 0 {
			wr = float64(a.Wins) / float64(a.Picks) * 100
		}
		pr := 0.0
		if matches > 0 {
			pr = float64(a.Picks) / float64(matches) * 100
		}
		hs := HeroStat{
			Hero:      hero,
			Picks:     a.Picks,
			Wins:      a.Wins,
			WinRate:   wr,
			PickRate:  pr,
			WRHistory: a.WeeklyWR,
		}
		if isSupport(hero) {
			ba.Supports = append(ba.Supports, hs)
		} else {
			ba.Cores = append(ba.Cores, hs)
		}
	}
	ClassifyTiers(ba.Cores)
	ClassifyTiers(ba.Supports)
	return ba
}

func isSupport(h stratz.Hero) bool {
	var support, carry bool
	for _, r := range h.Roles {
		if r == "Support" {
			support = true
		}
		if r == "Carry" {
			carry = true
		}
	}
	return support && !carry
}
