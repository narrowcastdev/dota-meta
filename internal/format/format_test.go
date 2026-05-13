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

func TestFormatReddit_PlainSections(t *testing.T) {
	full := analysis.FullAnalysis{
		Patch: "7.40b", TotalMatches: 100000,
		Brackets: []analysis.BracketAnalysis{{
			Bracket: analysis.Bracket{Name: "Divine"},
			Cores: []analysis.HeroStat{
				{Hero: stratz.Hero{ID: 1, DisplayName: "Visage"}, WinRate: 55, PickRate: 5, Picks: 12000, Tier: analysis.TierMetaTyrant},
				{Hero: stratz.Hero{ID: 2, DisplayName: "Sniper"}, WinRate: 56, PickRate: 1, Picks: 1500, Tier: analysis.TierPocketPick},
				{Hero: stratz.Hero{ID: 3, DisplayName: "PA"}, WinRate: 45, PickRate: 10, Picks: 20000, Tier: analysis.TierTrap},
			},
			Supports: []analysis.HeroStat{
				{Hero: stratz.Hero{ID: 4, DisplayName: "Lion"}, WinRate: 54, PickRate: 8, Picks: 18000},
			},
		}},
	}
	post := FormatReddit(full, "April 23, 2026")
	wants := []string{
		"Best heroes by bracket",
		"Sleeper picks",
		"Best support heroes",
		"Trap picks",
		"7.40b",
		"STRATZ",
		"Visage", "Sniper", "PA", "Lion",
	}
	for _, want := range wants {
		if !strings.Contains(post, want) {
			t.Errorf("missing %q in post", want)
		}
	}
	for _, banned := range []string{"👑", "🎯", "🪤", "💀", "🔥", "💎", "Meta Tyrant", "Momentum watch", "Tier"} {
		if strings.Contains(post, banned) {
			t.Errorf("unexpected %q in post — should be plain format", banned)
		}
	}
}

func TestFormatReddit_BracketDelta(t *testing.T) {
	full := analysis.FullAnalysis{
		Patch: "7.40b",
		Brackets: []analysis.BracketAnalysis{
			{
				Bracket: analysis.Bracket{Name: "Herald-Guardian"},
				Cores: []analysis.HeroStat{
					{Hero: stratz.Hero{ID: 1, DisplayName: "Sniper"}, WinRate: 55, PickRate: 5},
					{Hero: stratz.Hero{ID: 2, DisplayName: "Invoker"}, WinRate: 45, PickRate: 5},
				},
			},
			{
				Bracket: analysis.Bracket{Name: "Divine"},
				Cores: []analysis.HeroStat{
					{Hero: stratz.Hero{ID: 1, DisplayName: "Sniper"}, WinRate: 48, PickRate: 5},
					{Hero: stratz.Hero{ID: 2, DisplayName: "Invoker"}, WinRate: 53, PickRate: 5},
				},
			},
		},
	}
	post := FormatReddit(full, "April 23, 2026")
	if !strings.Contains(post, "Bracket delta") {
		t.Error("expected Bracket delta section")
	}
	if !strings.Contains(post, "Low bracket stompers") || !strings.Contains(post, "Sniper") {
		t.Error("expected Sniper as low-bracket stomper")
	}
	if !strings.Contains(post, "High skill ceiling") || !strings.Contains(post, "Invoker") {
		t.Error("expected Invoker as high skill ceiling")
	}
}
