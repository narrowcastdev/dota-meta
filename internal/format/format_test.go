package format_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/narrowcastdev/dota-meta/internal/analysis"
	"github.com/narrowcastdev/dota-meta/internal/api"
	"github.com/narrowcastdev/dota-meta/internal/format"
)

func loadFixtureAndAnalyze(t *testing.T) ([]api.Hero, analysis.FullAnalysis) {
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

	result := analysis.Analyze(heroes, 1000)
	return heroes, result
}

func TestFormatReddit_ContainsTitle(t *testing.T) {
	_, result := loadFixtureAndAnalyze(t)
	post := format.FormatReddit(result, "April 12, 2026")

	if !strings.Contains(post, "Week of April 12, 2026") {
		t.Error("post should contain the date in the title")
	}
}

func TestFormatReddit_ContainsBracketSections(t *testing.T) {
	_, result := loadFixtureAndAnalyze(t)
	post := format.FormatReddit(result, "April 12, 2026")

	for _, name := range []string{"Herald-Guardian", "Crusader-Archon", "Legend-Ancient", "Divine"} {
		if !strings.Contains(post, name) {
			t.Errorf("post should contain bracket %q", name)
		}
	}
}

func TestFormatReddit_ContainsSections(t *testing.T) {
	_, result := loadFixtureAndAnalyze(t)
	post := format.FormatReddit(result, "April 12, 2026")

	for _, section := range []string{
		"Best heroes by bracket",
		"Best support heroes",
		"Sleeper picks",
		"Trap picks",
		"Bracket delta",
		"Low bracket stompers",
		"High skill ceiling",
		"Free MMR",
	} {
		if !strings.Contains(post, section) {
			t.Errorf("post should contain section %q", section)
		}
	}
}

func TestFormatReddit_ContainsFooter(t *testing.T) {
	_, result := loadFixtureAndAnalyze(t)
	post := format.FormatReddit(result, "April 12, 2026")

	if !strings.Contains(post, "dota.narrowcast.dev") {
		t.Error("post should contain static site link")
	}
	if !strings.Contains(post, "OpenDota") {
		t.Error("post should contain OpenDota attribution")
	}
}

func TestFormatReddit_ContainsHeroNames(t *testing.T) {
	_, result := loadFixtureAndAnalyze(t)
	post := format.FormatReddit(result, "April 12, 2026")

	// Bloodseeker should appear (high WR in low brackets, known stomper)
	if !strings.Contains(post, "Bloodseeker") {
		t.Error("post should contain Bloodseeker")
	}
}

func TestFormatReddit_ContainsTableHeaders(t *testing.T) {
	_, result := loadFixtureAndAnalyze(t)
	post := format.FormatReddit(result, "April 12, 2026")

	if !strings.Contains(post, "| Hero | Win Rate | Pick Rate | Games |") {
		t.Error("post should contain best heroes table header")
	}
}

func TestFormatJSON_ValidJSON(t *testing.T) {
	heroes, result := loadFixtureAndAnalyze(t)
	data, err := format.FormatJSON(heroes, result)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
}

func TestFormatJSON_HasRequiredFields(t *testing.T) {
	heroes, result := loadFixtureAndAnalyze(t)
	data, err := format.FormatJSON(heroes, result)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, field := range []string{"generated", "heroes", "analysis"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("JSON output missing field %q", field)
		}
	}
}

func TestFormatJSON_HeroCount(t *testing.T) {
	heroes, result := loadFixtureAndAnalyze(t)
	data, err := format.FormatJSON(heroes, result)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	var parsed struct {
		Heroes []json.RawMessage `json:"heroes"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// All 10 heroes should be in the output (JSON includes all, not just qualified)
	if got := len(parsed.Heroes); got != 10 {
		t.Errorf("expected 10 heroes in JSON, got %d", got)
	}
}

func TestFormatJSON_AnalysisBrackets(t *testing.T) {
	heroes, result := loadFixtureAndAnalyze(t)
	data, err := format.FormatJSON(heroes, result)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	var parsed struct {
		Analysis struct {
			Brackets map[string]json.RawMessage `json:"brackets"`
			Delta    json.RawMessage            `json:"delta"`
		} `json:"analysis"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"herald_guardian", "crusader_archon", "legend_ancient", "divine"} {
		if _, ok := parsed.Analysis.Brackets[key]; !ok {
			t.Errorf("analysis missing bracket %q", key)
		}
	}

	if parsed.Analysis.Delta == nil {
		t.Error("analysis missing delta")
	}
}
