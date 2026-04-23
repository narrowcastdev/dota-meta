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
	Hero     stratz.Hero
	WinRate  float64
	PickRate float64
	Picks    int
	Wins     int
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
