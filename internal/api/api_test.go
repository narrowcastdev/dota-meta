package api_test

import (
	"math"
	"os"
	"strings"
	"testing"

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

func TestParseHeroStats_FixtureCount(t *testing.T) {
	heroes := loadFixture(t)
	if got := len(heroes); got != 10 {
		t.Errorf("expected 10 heroes, got %d", got)
	}
}

func TestParseHeroStats_Fields(t *testing.T) {
	heroes := loadFixture(t)

	// Find Anti-Mage (id=1)
	var am api.Hero
	for _, h := range heroes {
		if h.ID == 1 {
			am = h
			break
		}
	}
	if am.ID == 0 {
		t.Fatal("Anti-Mage not found in fixture")
	}

	if am.LocalizedName != "Anti-Mage" {
		t.Errorf("expected name Anti-Mage, got %q", am.LocalizedName)
	}
	if am.PrimaryAttr != "agi" {
		t.Errorf("expected primary_attr agi, got %q", am.PrimaryAttr)
	}
	if am.AttackType != "Melee" {
		t.Errorf("expected attack_type Melee, got %q", am.AttackType)
	}
	if len(am.Roles) != 3 {
		t.Errorf("expected 3 roles, got %d", len(am.Roles))
	}
}

func TestParseHeroStats_BracketMapping(t *testing.T) {
	heroes := loadFixture(t)

	// Anti-Mage bracket 1: pick=5000, win=2800
	var am api.Hero
	for _, h := range heroes {
		if h.ID == 1 {
			am = h
			break
		}
	}

	if am.BracketPick[0] != 5000 {
		t.Errorf("bracket 1 pick: expected 5000, got %d", am.BracketPick[0])
	}
	if am.BracketWin[0] != 2800 {
		t.Errorf("bracket 1 win: expected 2800, got %d", am.BracketWin[0])
	}

	// Bracket 8: pick=2500, win=1150
	if am.BracketPick[7] != 2500 {
		t.Errorf("bracket 8 pick: expected 2500, got %d", am.BracketPick[7])
	}
	if am.BracketWin[7] != 1150 {
		t.Errorf("bracket 8 win: expected 1150, got %d", am.BracketWin[7])
	}
}

func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestHero_WinRate(t *testing.T) {
	hero := api.Hero{
		BracketPick: [8]int{5000, 0, 0, 0, 0, 0, 0, 0},
		BracketWin:  [8]int{2800, 0, 0, 0, 0, 0, 0, 0},
	}

	wr := hero.WinRate(0)
	if !almostEqual(wr, 56.0, 0.01) {
		t.Errorf("expected WinRate ~56.0, got %.2f", wr)
	}

	// Zero picks should return 0
	if got := hero.WinRate(1); got != 0 {
		t.Errorf("expected 0 for zero picks, got %.2f", got)
	}
}

func TestHero_PickRate(t *testing.T) {
	hero := api.Hero{
		BracketPick: [8]int{5000, 0, 0, 0, 0, 0, 0, 0},
	}

	pr := hero.PickRate(0, 100000)
	if !almostEqual(pr, 5.0, 0.01) {
		t.Errorf("expected PickRate ~5.0, got %.2f", pr)
	}

	// Zero total should return 0
	if got := hero.PickRate(0, 0); got != 0 {
		t.Errorf("expected 0 for zero total, got %.2f", got)
	}
}

func TestParseHeroStats_EmptyResponse(t *testing.T) {
	_, err := api.ParseHeroStats(strings.NewReader("[]"))
	if err == nil {
		t.Fatal("expected error for empty response")
	}
	if !strings.Contains(err.Error(), "no hero data") {
		t.Errorf("expected 'no hero data' error, got: %v", err)
	}
}

func TestParseHeroStats_InvalidJSON(t *testing.T) {
	_, err := api.ParseHeroStats(strings.NewReader("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
