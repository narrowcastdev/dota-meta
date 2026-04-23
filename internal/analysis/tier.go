package analysis

import "sort"

// Tier is one of four quadrants derived from WR vs PR medians.
type Tier int

const (
	TierDead Tier = iota
	TierTrap
	TierPocketPick
	TierMetaTyrant
)

func (t Tier) String() string {
	switch t {
	case TierMetaTyrant:
		return "meta-tyrant"
	case TierPocketPick:
		return "pocket-pick"
	case TierTrap:
		return "trap"
	default:
		return "dead"
	}
}

// ClassifyTiers mutates stats in place, assigning Tier based on within-slice
// median WR and median PR splits. ">=" median counts as "high".
func ClassifyTiers(stats []HeroStat) {
	if len(stats) == 0 {
		return
	}
	wr := make([]float64, len(stats))
	pr := make([]float64, len(stats))
	for i, s := range stats {
		wr[i] = s.WinRate
		pr[i] = s.PickRate
	}
	sort.Float64s(wr)
	sort.Float64s(pr)
	medWR := median(wr)
	medPR := median(pr)
	for i := range stats {
		highWR := stats[i].WinRate >= medWR
		highPR := stats[i].PickRate >= medPR
		switch {
		case highWR && highPR:
			stats[i].Tier = TierMetaTyrant
		case highWR && !highPR:
			stats[i].Tier = TierPocketPick
		case !highWR && highPR:
			stats[i].Tier = TierTrap
		default:
			stats[i].Tier = TierDead
		}
	}
}

func median(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}
