package stratz

import (
	"math"
	"testing"
)

func TestAggregateBrackets_SingleBracket(t *testing.T) {
	resp := []BracketResponse{{
		Bracket: BracketDivine,
		Weeks: []HeroWeekStat{
			{HeroID: 1, Week: 100, MatchCount: 100, WinCount: 55},
			{HeroID: 1, Week: 101, MatchCount: 200, WinCount: 110},
		},
	}}
	got := AggregateBrackets(resp)
	h := got[1]
	if h.Picks != 300 || h.Wins != 165 {
		t.Errorf("totals=%d/%d, want 300/165", h.Picks, h.Wins)
	}
	if h.LatestPicks != 200 || h.LatestWins != 110 {
		t.Errorf("latest=%d/%d, want 200/110 (week 101)", h.LatestPicks, h.LatestWins)
	}
	if len(h.WeeklyWR) != 2 {
		t.Fatalf("weekly len=%d, want 2", len(h.WeeklyWR))
	}
	if math.Abs(h.WeeklyWR[0]-55.0) > 1e-6 {
		t.Errorf("wr0=%f", h.WeeklyWR[0])
	}
}

func TestAggregateBrackets_MergesTwoBrackets(t *testing.T) {
	a := BracketResponse{Bracket: BracketHerald, Weeks: []HeroWeekStat{
		{HeroID: 1, Week: 100, MatchCount: 100, WinCount: 50},
	}}
	b := BracketResponse{Bracket: BracketGuardian, Weeks: []HeroWeekStat{
		{HeroID: 1, Week: 100, MatchCount: 100, WinCount: 60},
	}}
	got := AggregateBrackets([]BracketResponse{a, b})
	if got[1].Picks != 200 || got[1].Wins != 110 {
		t.Errorf("merge totals=%d/%d, want 200/110", got[1].Picks, got[1].Wins)
	}
	if math.Abs(got[1].WeeklyWR[0]-55.0) > 1e-6 {
		t.Errorf("merged wr=%f, want 55.0", got[1].WeeklyWR[0])
	}
}

func TestAggregateBrackets_PopulatesWeeklyPR(t *testing.T) {
	resp := []BracketResponse{{
		Bracket: BracketDivine,
		Weeks: []HeroWeekStat{
			{HeroID: 1, Week: 100, MatchCount: 100, WinCount: 50},
			{HeroID: 2, Week: 100, MatchCount: 900, WinCount: 450},
		},
	}}
	got := AggregateBrackets(resp)
	// Week 100: total picks = 1000 → matches ≈ 100. Hero 1 PR = 100/100 = 100%.
	if len(got[1].WeeklyPR) != 1 {
		t.Fatalf("len(WeeklyPR)=%d", len(got[1].WeeklyPR))
	}
	if got[1].WeeklyPR[0] != 100 {
		t.Errorf("hero 1 PR=%v want 100", got[1].WeeklyPR[0])
	}
}
