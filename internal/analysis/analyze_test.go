package analysis

import (
	"testing"

	"github.com/narrowcastdev/dota-meta/internal/api/stratz"
)

func TestAnalyze_SplitsCoresAndSupports(t *testing.T) {
	heroes := []stratz.Hero{
		{ID: 1, DisplayName: "Anti-Mage", Roles: []string{"Carry"}},
		{ID: 2, DisplayName: "Crystal Maiden", Roles: []string{"Support", "Disabler"}},
		{ID: 3, DisplayName: "Wraith King", Roles: []string{"Carry", "Support"}},
	}
	resp := []stratz.BracketResponse{{
		Bracket: stratz.BracketDivine,
		Weeks: []stratz.HeroWeekStat{
			{HeroID: 1, Week: 100, MatchCount: 2000, WinCount: 1100},
			{HeroID: 2, Week: 100, MatchCount: 2000, WinCount: 1050},
			{HeroID: 3, Week: 100, MatchCount: 2000, WinCount: 900},
		},
	}}
	got := Analyze(heroes, resp, 1000)
	divine := findBracket(t, got, "Divine")
	coreNames := names(divine.Cores)
	suppNames := names(divine.Supports)
	if !hasName(coreNames, "Anti-Mage") {
		t.Errorf("AM should be core: %v", coreNames)
	}
	if !hasName(coreNames, "Wraith King") {
		t.Errorf("WK (flex) should be core: %v", coreNames)
	}
	if !hasName(suppNames, "Crystal Maiden") {
		t.Errorf("CM should be support: %v", suppNames)
	}
}

func TestAnalyze_HonorsMinPicks(t *testing.T) {
	heroes := []stratz.Hero{{ID: 1, DisplayName: "Anti-Mage", Roles: []string{"Carry"}}}
	resp := []stratz.BracketResponse{{
		Bracket: stratz.BracketDivine,
		Weeks: []stratz.HeroWeekStat{
			{HeroID: 1, Week: 100, MatchCount: 500, WinCount: 250},
		},
	}}
	got := Analyze(heroes, resp, 1000)
	divine := findBracket(t, got, "Divine")
	if len(divine.Cores)+len(divine.Supports) != 0 {
		t.Errorf("expected 0 qualified, got %d cores %d supports", len(divine.Cores), len(divine.Supports))
	}
}

func findBracket(t *testing.T, a FullAnalysis, name string) BracketAnalysis {
	t.Helper()
	for _, b := range a.Brackets {
		if b.Bracket.Name == name {
			return b
		}
	}
	t.Fatalf("bracket %q not found", name)
	return BracketAnalysis{}
}
func names(stats []HeroStat) []string {
	out := make([]string, 0, len(stats))
	for _, s := range stats {
		out = append(out, s.Hero.DisplayName)
	}
	return out
}
func hasName(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
