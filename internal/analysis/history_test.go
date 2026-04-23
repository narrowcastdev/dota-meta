package analysis

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/narrowcastdev/dota-meta/internal/api/stratz"
)

func writeSnapshot(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadLatestPriorSnapshot_PicksNewestBeforeToday(t *testing.T) {
	dir := t.TempDir()
	writeSnapshot(t, dir, "data-2026-04-10.json", `{"patch":"7.40a","analysis":{"brackets":{"d":{"name":"Divine","cores":[{"name":"AM","win_rate":50,"pick_rate":5}],"supports":[]}}}}`)
	writeSnapshot(t, dir, "data-2026-04-15.json", `{"patch":"7.40b","analysis":{"brackets":{"d":{"name":"Divine","cores":[{"name":"AM","win_rate":52,"pick_rate":6}],"supports":[]}}}}`)
	writeSnapshot(t, dir, "data-2026-04-23.json", `{"patch":"7.40b","analysis":{"brackets":{"d":{"name":"Divine","cores":[{"name":"AM","win_rate":99,"pick_rate":9}],"supports":[]}}}}`)
	today, _ := time.Parse("2006-01-02", "2026-04-23")
	got, err := LoadLatestPriorSnapshot(dir, today)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("got nil")
	}
	if got.Patch != "7.40b" {
		t.Errorf("patch=%q want 7.40b", got.Patch)
	}
	if got.Stats["Divine"]["AM"].WR != 52 {
		t.Errorf("wr=%v want 52", got.Stats["Divine"]["AM"].WR)
	}
}

func TestLoadLatestPriorSnapshot_NoDir(t *testing.T) {
	today := time.Now()
	got, err := LoadLatestPriorSnapshot(filepath.Join(t.TempDir(), "nope"), today)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("want nil")
	}
}

func TestApplyDeltas_FillsDeltas(t *testing.T) {
	full := FullAnalysis{Brackets: []BracketAnalysis{{
		Bracket: Bracket{Name: "Divine"},
		Cores:   []HeroStat{{Hero: stratz.Hero{DisplayName: "AM"}, WinRate: 55, PickRate: 7}},
	}}}
	prior := &snapshotSummary{
		Date: time.Now().AddDate(0, 0, -7),
		Stats: map[string]map[string]struct{ WR, PR float64 }{
			"Divine": {"AM": {WR: 53, PR: 6}},
		},
	}
	ApplyDeltas(&full, prior)
	got := full.Brackets[0].Cores[0]
	if got.WRDelta == nil || *got.WRDelta != 2 {
		t.Errorf("wrDelta=%v want 2", got.WRDelta)
	}
	if got.PRDelta == nil || *got.PRDelta != 1 {
		t.Errorf("prDelta=%v want 1", got.PRDelta)
	}
}
