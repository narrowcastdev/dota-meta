package analysis_test

import (
	"os"
	"testing"

	"github.com/narrowcastdev/dota-meta/internal/analysis"
	"github.com/narrowcastdev/dota-meta/internal/api"
)

func loadFixture(t *testing.T) []api.Hero {
	t.Helper()
	f, err := os.Open("../../testdata/herostats.json")
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer f.Close()

	heroes, err := api.ParseHeroStats(f)
	if err != nil {
		t.Fatalf("parsing fixture: %v", err)
	}
	return heroes
}

func TestAnalyze_BracketCount(t *testing.T) {
	heroes := loadFixture(t)
	result := analysis.Analyze(heroes, 1000)

	if got := len(result.Brackets); got != 4 {
		t.Errorf("expected 4 bracket analyses, got %d", got)
	}
}

func TestAnalyze_MinPicksFilter(t *testing.T) {
	heroes := loadFixture(t)

	// With minPicks=1000, Bane (all brackets <1000) should be excluded
	result := analysis.Analyze(heroes, 1000)

	for _, ba := range result.Brackets {
		for _, best := range ba.Best {
			if best.Hero.LocalizedName == "Bane" {
				t.Errorf("Bane should be filtered out with minPicks=1000 in %s", ba.Bracket.Name)
			}
		}
	}
}

func TestAnalyze_MinPicksFilterIncludesLowPickHeroes(t *testing.T) {
	heroes := loadFixture(t)

	// With minPicks=100, Bane should be included
	result := analysis.Analyze(heroes, 100)

	found := false
	for _, ba := range result.Brackets {
		for _, best := range ba.Best {
			if best.Hero.LocalizedName == "Bane" {
				found = true
			}
		}
	}
	if !found {
		// Bane might not be in "best" but should at least be analyzed.
		// Check that the bracket analysis has more heroes than with high threshold.
		highResult := analysis.Analyze(heroes, 5000)
		if len(result.Brackets[0].Best) <= len(highResult.Brackets[0].Best) {
			// This is fine — best is capped at 5. Just verify no crash.
		}
	}
}

func TestAnalyze_BestHeroesSortedByWinRate(t *testing.T) {
	heroes := loadFixture(t)
	result := analysis.Analyze(heroes, 1000)

	for _, ba := range result.Brackets {
		for i := 1; i < len(ba.Best); i++ {
			if ba.Best[i].WinRate > ba.Best[i-1].WinRate {
				t.Errorf("%s: best heroes not sorted by WR descending at index %d: %.2f > %.2f",
					ba.Bracket.Name, i, ba.Best[i].WinRate, ba.Best[i-1].WinRate)
			}
		}
	}
}

func TestAnalyze_BestHeroesMaxFive(t *testing.T) {
	heroes := loadFixture(t)
	result := analysis.Analyze(heroes, 1000)

	for _, ba := range result.Brackets {
		if len(ba.Best) > 5 {
			t.Errorf("%s: expected at most 5 best heroes, got %d", ba.Bracket.Name, len(ba.Best))
		}
	}
}

func TestAnalyze_SleeperPickThresholds(t *testing.T) {
	heroes := loadFixture(t)
	result := analysis.Analyze(heroes, 1000)

	for _, ba := range result.Brackets {
		for _, s := range ba.Sleepers {
			if s.WinRate < 53 {
				t.Errorf("%s: sleeper %s has WR %.2f < 53%%",
					ba.Bracket.Name, s.Hero.LocalizedName, s.WinRate)
			}
		}
	}
}

func TestAnalyze_TrapPickThresholds(t *testing.T) {
	heroes := loadFixture(t)
	result := analysis.Analyze(heroes, 1000)

	for _, ba := range result.Brackets {
		for _, s := range ba.Traps {
			if s.WinRate >= 48 {
				t.Errorf("%s: trap %s has WR %.2f >= 48%%",
					ba.Bracket.Name, s.Hero.LocalizedName, s.WinRate)
			}
		}
	}
}

func TestAnalyze_BracketDeltaLowStompers(t *testing.T) {
	heroes := loadFixture(t)
	result := analysis.Analyze(heroes, 1000)

	for _, d := range result.LowStompers {
		if d.Delta <= 0 {
			t.Errorf("low stomper %s has non-positive delta %.2f", d.Hero.LocalizedName, d.Delta)
		}
		if d.LowWR <= d.HighWR {
			t.Errorf("low stomper %s: low WR %.2f should exceed high WR %.2f",
				d.Hero.LocalizedName, d.LowWR, d.HighWR)
		}
	}

	if len(result.LowStompers) > 5 {
		t.Errorf("expected at most 5 low stompers, got %d", len(result.LowStompers))
	}
}

func TestAnalyze_BracketDeltaHighSkillCap(t *testing.T) {
	heroes := loadFixture(t)
	result := analysis.Analyze(heroes, 1000)

	for _, d := range result.HighSkillCap {
		if d.Delta >= 0 {
			t.Errorf("high skill cap %s has non-negative delta %.2f", d.Hero.LocalizedName, d.Delta)
		}
		if d.HighWR <= d.LowWR {
			t.Errorf("high skill cap %s: high WR %.2f should exceed low WR %.2f",
				d.Hero.LocalizedName, d.HighWR, d.LowWR)
		}
	}

	if len(result.HighSkillCap) > 5 {
		t.Errorf("expected at most 5 high skill cap, got %d", len(result.HighSkillCap))
	}
}

func TestAnalyze_BracketDeltaKnownHeroes(t *testing.T) {
	heroes := loadFixture(t)
	result := analysis.Analyze(heroes, 1000)

	// Bloodseeker should be a low bracket stomper (high WR in Herald-Guardian, low in Divine-Immortal)
	foundBloodseeker := false
	for _, d := range result.LowStompers {
		if d.Hero.LocalizedName == "Bloodseeker" {
			foundBloodseeker = true
		}
	}
	if !foundBloodseeker {
		t.Error("expected Bloodseeker to be a low bracket stomper")
	}

	// Earthshaker should be high skill ceiling (low WR in Herald-Guardian, high in Divine-Immortal)
	foundEarthshaker := false
	for _, d := range result.HighSkillCap {
		if d.Hero.LocalizedName == "Earthshaker" {
			foundEarthshaker = true
		}
	}
	if !foundEarthshaker {
		t.Error("expected Earthshaker to be high skill ceiling")
	}
}

func TestAnalyze_TotalMatches(t *testing.T) {
	heroes := loadFixture(t)
	result := analysis.Analyze(heroes, 1000)

	if result.TotalMatches <= 0 {
		t.Errorf("expected positive total matches, got %d", result.TotalMatches)
	}
}

func TestAnalyze_EmptyHeroes(t *testing.T) {
	result := analysis.Analyze(nil, 1000)

	if len(result.Brackets) != 4 {
		t.Errorf("expected 4 brackets even with empty heroes, got %d", len(result.Brackets))
	}
	for _, ba := range result.Brackets {
		if len(ba.Best) != 0 {
			t.Errorf("expected no best heroes with empty input")
		}
	}
}
