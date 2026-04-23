package stratz

import "sort"

const picksPerMatch = 10

// AggregatedHero is one hero's totals + weekly series inside an analysis bracket.
type AggregatedHero struct {
	HeroID      int
	Picks       int // cumulative across all returned weeks
	Wins        int
	LatestPicks int // most recent week only
	LatestWins  int
	WeeklyWR    []float64
	WeeklyPR    []float64 // % of bracket matches in that week, oldest first
}

// AggregateBrackets rolls up a list of STRATZ bracket responses into one
// AggregatedHero per heroID. Callers pass only brackets in a single analysis
// bucket.
func AggregateBrackets(responses []BracketResponse) map[int]AggregatedHero {
	type weekAgg struct{ picks, wins int }
	type acc struct {
		totalPicks, totalWins int
		weekly                map[int64]weekAgg
	}
	byHero := map[int]*acc{}
	weekTotalPicks := map[int64]int{}
	for _, br := range responses {
		for _, w := range br.Weeks {
			a, ok := byHero[w.HeroID]
			if !ok {
				a = &acc{weekly: map[int64]weekAgg{}}
				byHero[w.HeroID] = a
			}
			a.totalPicks += w.MatchCount
			a.totalWins += w.WinCount
			wa := a.weekly[w.Week]
			wa.picks += w.MatchCount
			wa.wins += w.WinCount
			a.weekly[w.Week] = wa
			weekTotalPicks[w.Week] += w.MatchCount
		}
	}
	out := make(map[int]AggregatedHero, len(byHero))
	for id, a := range byHero {
		weeks := make([]int64, 0, len(a.weekly))
		for w := range a.weekly {
			weeks = append(weeks, w)
		}
		sort.Slice(weeks, func(i, j int) bool { return weeks[i] < weeks[j] })
		series := make([]float64, 0, len(weeks))
		prSeries := make([]float64, 0, len(weeks))
		for _, w := range weeks {
			x := a.weekly[w]
			if x.picks == 0 {
				series = append(series, 0)
				prSeries = append(prSeries, 0)
				continue
			}
			series = append(series, float64(x.wins)/float64(x.picks)*100)
			totalMatches := weekTotalPicks[w] / picksPerMatch
			if totalMatches == 0 {
				prSeries = append(prSeries, 0)
				continue
			}
			prSeries = append(prSeries, float64(x.picks)/float64(totalMatches)*100)
		}
		var latestPicks, latestWins int
		if len(weeks) > 0 {
			last := a.weekly[weeks[len(weeks)-1]]
			latestPicks = last.picks
			latestWins = last.wins
		}
		out[id] = AggregatedHero{
			HeroID: id, Picks: a.totalPicks, Wins: a.totalWins,
			LatestPicks: latestPicks, LatestWins: latestWins,
			WeeklyWR: series, WeeklyPR: prSeries,
		}
	}
	return out
}
