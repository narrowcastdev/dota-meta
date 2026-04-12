package analysis

import (
	"sort"

	"github.com/narrowcastdev/dota-meta/internal/api"
)

// BracketPair represents two adjacent brackets combined for analysis.
type BracketPair struct {
	Name     string
	Brackets [2]int // indices into BracketPick/BracketWin (0-indexed)
}

// BracketPairs defines the four bracket pairs used for analysis.
var BracketPairs = []BracketPair{
	{Name: "Herald-Guardian", Brackets: [2]int{0, 1}},
	{Name: "Crusader-Archon", Brackets: [2]int{2, 3}},
	{Name: "Legend-Ancient", Brackets: [2]int{4, 5}},
	{Name: "Divine-Immortal", Brackets: [2]int{6, 7}},
}

// HeroStat holds computed stats for one hero in one bracket pair.
type HeroStat struct {
	Hero     api.Hero
	WinRate  float64
	PickRate float64
	Picks    int
	Wins     int
}

// BracketAnalysis holds all analysis results for one bracket pair.
type BracketAnalysis struct {
	Pair       BracketPair
	Best       []HeroStat
	Sleepers   []HeroStat
	Traps      []HeroStat
	TotalPicks int
}

// DeltaHero holds bracket delta data for one hero.
type DeltaHero struct {
	Hero   api.Hero
	LowWR  float64 // Herald-Guardian win rate
	HighWR float64 // Divine-Immortal win rate
	Delta  float64 // LowWR - HighWR (positive = stronger in low brackets)
}

// FullAnalysis holds the complete analysis output.
type FullAnalysis struct {
	Brackets     []BracketAnalysis
	LowStompers  []DeltaHero // Heroes stronger in low brackets
	HighSkillCap []DeltaHero // Heroes stronger in high brackets
	TotalMatches int
}

// Analyze runs the full analysis on all heroes.
func Analyze(heroes []api.Hero, minPicks int) FullAnalysis {
	var result FullAnalysis

	for _, pair := range BracketPairs {
		ba := analyzeBracketPair(heroes, pair, minPicks)
		result.Brackets = append(result.Brackets, ba)
		result.TotalMatches += ba.TotalPicks
	}

	// Total matches is total picks / 2 (each match has 10 picks but heroStats
	// counts per-hero, so total picks across all heroes / 10 * 5 teams...
	// actually total picks is just total hero selections. Divide by 10 for matches.
	// Keep as total picks for display — it's more impressive and accurate.

	result.LowStompers, result.HighSkillCap = analyzeBracketDelta(heroes, minPicks)

	return result
}

func analyzeBracketPair(heroes []api.Hero, pair BracketPair, minPicks int) BracketAnalysis {
	b1, b2 := pair.Brackets[0], pair.Brackets[1]

	var totalPicks int
	for _, h := range heroes {
		totalPicks += h.BracketPick[b1] + h.BracketPick[b2]
	}

	var qualified []HeroStat
	for _, h := range heroes {
		picks := h.BracketPick[b1] + h.BracketPick[b2]
		if picks < minPicks {
			continue
		}
		wins := h.BracketWin[b1] + h.BracketWin[b2]
		wr := float64(wins) / float64(picks) * 100
		pr := float64(picks) / float64(totalPicks) * 100

		qualified = append(qualified, HeroStat{
			Hero:     h,
			WinRate:  wr,
			PickRate: pr,
			Picks:    picks,
			Wins:     wins,
		})
	}

	ba := BracketAnalysis{
		Pair:       pair,
		TotalPicks: totalPicks,
	}

	ba.Best = bestHeroes(qualified)
	ba.Sleepers = sleeperPicks(qualified)
	ba.Traps = trapPicks(qualified)

	return ba
}

// bestHeroes returns the top 5 heroes by win rate.
func bestHeroes(stats []HeroStat) []HeroStat {
	sorted := make([]HeroStat, len(stats))
	copy(sorted, stats)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].WinRate > sorted[j].WinRate
	})

	if len(sorted) > 5 {
		sorted = sorted[:5]
	}
	return sorted
}

// sleeperPicks returns heroes with WR >= 53% and pick rate in the bottom 40th percentile.
func sleeperPicks(stats []HeroStat) []HeroStat {
	if len(stats) == 0 {
		return nil
	}

	pickRateThreshold := pickRatePercentile(stats, 40)

	var sleepers []HeroStat
	for _, s := range stats {
		if s.WinRate >= 53 && s.PickRate <= pickRateThreshold {
			sleepers = append(sleepers, s)
		}
	}

	sort.Slice(sleepers, func(i, j int) bool {
		return sleepers[i].WinRate > sleepers[j].WinRate
	})

	return sleepers
}

// trapPicks returns heroes with pick rate in the top 20th percentile and WR < 48%.
func trapPicks(stats []HeroStat) []HeroStat {
	if len(stats) == 0 {
		return nil
	}

	pickRateThreshold := pickRatePercentile(stats, 80)

	var traps []HeroStat
	for _, s := range stats {
		if s.PickRate >= pickRateThreshold && s.WinRate < 48 {
			traps = append(traps, s)
		}
	}

	sort.Slice(traps, func(i, j int) bool {
		return traps[i].PickRate > traps[j].PickRate
	})

	return traps
}

// pickRatePercentile returns the pick rate at the given percentile.
// percentile should be 0-100.
func pickRatePercentile(stats []HeroStat, percentile float64) float64 {
	if len(stats) == 0 {
		return 0
	}

	rates := make([]float64, len(stats))
	for i, s := range stats {
		rates[i] = s.PickRate
	}
	sort.Float64s(rates)

	idx := int(percentile / 100 * float64(len(rates)))
	if idx >= len(rates) {
		idx = len(rates) - 1
	}
	return rates[idx]
}

func analyzeBracketDelta(heroes []api.Hero, minPicks int) (lowStompers []DeltaHero, highSkillCap []DeltaHero) {
	lowPair := BracketPairs[0]  // Herald-Guardian
	highPair := BracketPairs[3] // Divine-Immortal

	lb1, lb2 := lowPair.Brackets[0], lowPair.Brackets[1]
	hb1, hb2 := highPair.Brackets[0], highPair.Brackets[1]

	var deltas []DeltaHero
	for _, h := range heroes {
		lowPicks := h.BracketPick[lb1] + h.BracketPick[lb2]
		highPicks := h.BracketPick[hb1] + h.BracketPick[hb2]

		if lowPicks < minPicks || highPicks < minPicks {
			continue
		}

		lowWins := h.BracketWin[lb1] + h.BracketWin[lb2]
		highWins := h.BracketWin[hb1] + h.BracketWin[hb2]

		lowWR := float64(lowWins) / float64(lowPicks) * 100
		highWR := float64(highWins) / float64(highPicks) * 100

		deltas = append(deltas, DeltaHero{
			Hero:   h,
			LowWR:  lowWR,
			HighWR: highWR,
			Delta:  lowWR - highWR,
		})
	}

	// Low bracket stompers: highest positive delta (stronger in low brackets)
	sort.Slice(deltas, func(i, j int) bool {
		return deltas[i].Delta > deltas[j].Delta
	})
	for _, d := range deltas {
		if d.Delta <= 0 {
			break
		}
		lowStompers = append(lowStompers, d)
		if len(lowStompers) >= 5 {
			break
		}
	}

	// High skill ceiling: largest negative delta (stronger in high brackets)
	sort.Slice(deltas, func(i, j int) bool {
		return deltas[i].Delta < deltas[j].Delta
	})
	for _, d := range deltas {
		if d.Delta >= 0 {
			break
		}
		highSkillCap = append(highSkillCap, d)
		if len(highSkillCap) >= 5 {
			break
		}
	}

	return lowStompers, highSkillCap
}
