package analysis

import "testing"

func TestClassifyTiers_MedianSplit(t *testing.T) {
	stats := []HeroStat{
		{WinRate: 60, PickRate: 10},
		{WinRate: 58, PickRate: 2},
		{WinRate: 45, PickRate: 12},
		{WinRate: 43, PickRate: 1},
	}
	ClassifyTiers(stats)
	want := map[int]Tier{0: TierMetaTyrant, 1: TierPocketPick, 2: TierTrap, 3: TierDead}
	for i, w := range want {
		if stats[i].Tier != w {
			t.Errorf("stats[%d].Tier=%v, want %v", i, stats[i].Tier, w)
		}
	}
}

func TestClassifyTiers_Empty(t *testing.T) {
	var stats []HeroStat
	ClassifyTiers(stats)
}

func TestClassifyTiers_SingleHero(t *testing.T) {
	stats := []HeroStat{{WinRate: 50, PickRate: 5}}
	ClassifyTiers(stats)
	if stats[0].Tier != TierMetaTyrant {
		t.Errorf("got %v, want Meta Tyrant", stats[0].Tier)
	}
}
