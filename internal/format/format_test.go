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
	if core0["climb"] != "scales-up" {
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
