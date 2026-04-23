package format

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/narrowcastdev/dota-meta/internal/analysis"
	"github.com/narrowcastdev/dota-meta/internal/api/stratz"
)

func TestFormatJSON_EmitsTierClimbMomentum(t *testing.T) {
	heroes := []stratz.Hero{{ID: 1, DisplayName: "Visage", ShortName: "visage", Roles: []string{"Carry"}}}
	now := time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC)
	full := analysis.FullAnalysis{
		Patch:        "7.40b",
		SnapshotDate: now,
		Brackets: []analysis.BracketAnalysis{{
			Bracket: analysis.Bracket{Name: "Divine"},
			Cores: []analysis.HeroStat{{
				Hero: heroes[0], Picks: 1200, Wins: 660, WinRate: 55, PickRate: 2.3,
				Tier: analysis.TierPocketPick, ClimbTag: analysis.ClimbUp,
				Momentum: analysis.MomentumHidden, WRHistory: []float64{53, 54, 54.5, 55},
			}},
		}},
	}
	out, err := FormatJSON(heroes, full)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	a := got["analysis"].(map[string]any)["brackets"].(map[string]any)["divine"].(map[string]any)
	cores := a["cores"].([]any)
	core0 := cores[0].(map[string]any)
	if core0["tier"] != "pocket-pick" {
		t.Errorf("tier=%v", core0["tier"])
	}
	if core0["climb"] != "high-skill" {
		t.Errorf("climb=%v", core0["climb"])
	}
	if core0["momentum"] != "hidden-gem" {
		t.Errorf("momentum=%v", core0["momentum"])
	}
	if !strings.Contains(string(out), `"wr_history"`) {
		t.Errorf("no wr_history in output")
	}
}

func TestFormatJSON_OmitsPriorSnapshotWhenNil(t *testing.T) {
	full := analysis.FullAnalysis{Patch: "7.40b", SnapshotDate: time.Now().UTC()}
	out, err := FormatJSON(nil, full)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), `"prior_snapshot"`) {
		t.Errorf("prior_snapshot should be omitted when nil")
	}
}

func TestFormatJSON_IncludesPriorSnapshotWhenSet(t *testing.T) {
	d := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	full := analysis.FullAnalysis{Patch: "7.40b", SnapshotDate: time.Now().UTC(), PriorSnapshot: &d}
	out, err := FormatJSON(nil, full)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"prior_snapshot": "2026-04-16"`) {
		t.Errorf("prior_snapshot missing: %s", out)
	}
}

func TestFormatReddit_HasTierSections(t *testing.T) {
	full := analysis.FullAnalysis{
		Patch: "7.40b", TotalMatches: 100000,
		Brackets: []analysis.BracketAnalysis{{
			Bracket: analysis.Bracket{Name: "Divine"},
			Cores: []analysis.HeroStat{
				{Hero: stratz.Hero{DisplayName: "Visage"}, WinRate: 55, PickRate: 5, Tier: analysis.TierMetaTyrant},
				{Hero: stratz.Hero{DisplayName: "Sniper"}, WinRate: 56, PickRate: 1, Tier: analysis.TierPocketPick},
				{Hero: stratz.Hero{DisplayName: "PA"}, WinRate: 45, PickRate: 10, Tier: analysis.TierTrap},
			},
		}},
	}
	post := FormatReddit(full, "April 23, 2026")
	for _, want := range []string{"Meta Tyrants", "Pocket Picks", "Traps", "Top Cores", "7.40b", "Visage", "Sniper", "PA", "👑", "🎯", "🪤"} {
		if !strings.Contains(post, want) {
			t.Errorf("missing %q in post", want)
		}
	}
}

func TestFormatReddit_MomentumSectionOnlyWhenPresent(t *testing.T) {
	full := analysis.FullAnalysis{
		Patch: "7.40b",
		Brackets: []analysis.BracketAnalysis{{
			Bracket: analysis.Bracket{Name: "Divine"},
			Cores: []analysis.HeroStat{
				{Hero: stratz.Hero{DisplayName: "Lina"}, WinRate: 55, PickRate: 5, Tier: analysis.TierMetaTyrant, Momentum: analysis.MomentumHidden},
			},
		}},
	}
	post := FormatReddit(full, "April 23, 2026")
	if !strings.Contains(post, "Momentum watch") {
		t.Error("expected Momentum watch section")
	}
	if !strings.Contains(post, "Hidden gems") {
		t.Error("expected Hidden gems subsection")
	}
	// No momentum → no section
	full.Brackets[0].Cores[0].Momentum = analysis.MomentumNone
	post = FormatReddit(full, "April 23, 2026")
	if strings.Contains(post, "Momentum watch") {
		t.Error("should not include Momentum section when empty")
	}
}
