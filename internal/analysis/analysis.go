package analysis

import (
	"sort"

	"github.com/narrowcastdev/dota-meta/internal/api/opendota"
)

// picksPerMatch is how many hero picks OpenDota counts per match (5 per team x 2).
const picksPerMatch = 10

// Bracket groups one or more OpenDota skill brackets into a single analysis bucket.
type Bracket struct {
	Name    string
	Indices []int // 0-indexed into Hero.BracketPick/BracketWin (0=Herald ... 7=Immortal)
}

// Brackets defines the analysis buckets. The OpenDota heroStats endpoint's
// bracket 8 (Immortal) is empty, so Divine (index 6) is effectively the top
// bucket; index 7 is kept in the sum as a no-op in case the feed ever fills it.
var Brackets = []Bracket{
	{Name: "Herald-Guardian", Indices: []int{0, 1}},
	{Name: "Crusader-Archon", Indices: []int{2, 3}},
	{Name: "Legend-Ancient", Indices: []int{4, 5}},
	{Name: "Divine", Indices: []int{6, 7}},
}

// HeroStat holds computed stats for one hero in one bracket.
type HeroStat struct {
	Hero     opendota.Hero
	WinRate  float64
	PickRate float64 // % of matches in this bracket where the hero was picked
	Picks    int
	Wins     int
}

// BracketAnalysis holds all analysis results for one bracket.
type BracketAnalysis struct {
	Bracket    Bracket
	Best       []HeroStat
	Sleepers   []HeroStat
	Traps      []HeroStat
	Supports   []HeroStat // Top 5 heroes tagged with "Support" role by win rate
	TotalPicks int        // sum of all hero picks in this bracket (≈ 10 × matches)
}

// Matches returns the approximate match count for this bracket.
func (ba BracketAnalysis) Matches() int {
	return ba.TotalPicks / picksPerMatch
}

// DeltaHero holds bracket delta data for one hero.
type DeltaHero struct {
	Hero   opendota.Hero
	LowWR  float64 // Herald-Guardian win rate
	HighWR float64 // Immortal win rate
	Delta  float64 // LowWR - HighWR (positive = stronger in low brackets)
}

// FullAnalysis holds the complete analysis output.
type FullAnalysis struct {
	Brackets     []BracketAnalysis
	LowStompers  []DeltaHero // Heroes stronger in low brackets
	HighSkillCap []DeltaHero // Heroes stronger in high brackets
	TotalMatches int         // sum of matches across all buckets
}

// Analyze runs the full analysis on all heroes.
func Analyze(heroes []opendota.Hero, minPicks int) FullAnalysis {
	var result FullAnalysis

	for _, bracket := range Brackets {
		ba := analyzeBracket(heroes, bracket, minPicks)
		result.Brackets = append(result.Brackets, ba)
		result.TotalMatches += ba.Matches()
	}

	result.LowStompers, result.HighSkillCap = analyzeBracketDelta(heroes, minPicks)

	return result
}

func sumIndices(values [8]int, indices []int) int {
	var total int
	for _, i := range indices {
		total += values[i]
	}
	return total
}

func analyzeBracket(heroes []opendota.Hero, bracket Bracket, minPicks int) BracketAnalysis {
	var totalPicks int
	for _, h := range heroes {
		totalPicks += sumIndices(h.BracketPick, bracket.Indices)
	}

	matches := totalPicks / picksPerMatch

	var qualified []HeroStat
	for _, h := range heroes {
		picks := sumIndices(h.BracketPick, bracket.Indices)
		if picks < minPicks {
			continue
		}
		wins := sumIndices(h.BracketWin, bracket.Indices)
		wr := float64(wins) / float64(picks) * 100

		var pr float64
		if matches > 0 {
			pr = float64(picks) / float64(matches) * 100
		}

		qualified = append(qualified, HeroStat{
			Hero:     h,
			WinRate:  wr,
			PickRate: pr,
			Picks:    picks,
			Wins:     wins,
		})
	}

	ba := BracketAnalysis{
		Bracket:    bracket,
		TotalPicks: totalPicks,
	}

	ba.Best = bestHeroes(qualified)
	ba.Sleepers = sleeperPicks(qualified)
	ba.Traps = trapPicks(qualified)
	ba.Supports = bestSupports(qualified)

	return ba
}

// bestSupports returns top 5 heroes by win rate tagged Support but not Carry.
// OpenDota tags flex heroes like Wraith King with both roles; excluding Carry
// filters those out since the Support section is meant for pos 4/5 picks.
func bestSupports(stats []HeroStat) []HeroStat {
	var filtered []HeroStat
	for _, s := range stats {
		var isSupport, isCarry bool
		for _, r := range s.Hero.Roles {
			if r == "Support" {
				isSupport = true
			}
			if r == "Carry" {
				isCarry = true
			}
		}
		if isSupport && !isCarry {
			filtered = append(filtered, s)
		}
	}
	return bestHeroes(filtered)
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

func analyzeBracketDelta(heroes []opendota.Hero, minPicks int) (lowStompers []DeltaHero, highSkillCap []DeltaHero) {
	lowBracket := Brackets[0]                // Herald-Guardian
	highBracket := Brackets[len(Brackets)-1] // Immortal

	var deltas []DeltaHero
	for _, h := range heroes {
		lowPicks := sumIndices(h.BracketPick, lowBracket.Indices)
		highPicks := sumIndices(h.BracketPick, highBracket.Indices)

		if lowPicks < minPicks || highPicks < minPicks {
			continue
		}

		lowWins := sumIndices(h.BracketWin, lowBracket.Indices)
		highWins := sumIndices(h.BracketWin, highBracket.Indices)

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
