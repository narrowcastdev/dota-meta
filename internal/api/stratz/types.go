package stratz

// Bracket is the STRATZ rank-bracket enum (see RankBracket in their schema).
type Bracket string

const (
	BracketHerald   Bracket = "HERALD"
	BracketGuardian Bracket = "GUARDIAN"
	BracketCrusader Bracket = "CRUSADER"
	BracketArchon   Bracket = "ARCHON"
	BracketLegend   Bracket = "LEGEND"
	BracketAncient  Bracket = "ANCIENT"
	BracketDivine   Bracket = "DIVINE"
	BracketImmortal Bracket = "IMMORTAL"
)

// AllBrackets returns the 8 brackets in ascending order.
func AllBrackets() []Bracket {
	return []Bracket{
		BracketHerald, BracketGuardian, BracketCrusader, BracketArchon,
		BracketLegend, BracketAncient, BracketDivine, BracketImmortal,
	}
}

// HeroWeekStat is one weekly (hero, bracket) aggregate.
type HeroWeekStat struct {
	HeroID     int   `json:"heroId"`
	Week       int64 `json:"week"`
	MatchCount int   `json:"matchCount"`
	WinCount   int   `json:"winCount"`
}

// BracketResponse is STRATZ's reply for one bracket.
type BracketResponse struct {
	Bracket Bracket
	Weeks   []HeroWeekStat
}

// Hero is the bracket-agnostic hero catalog entry (from constants.heroes).
type Hero struct {
	ID               int      `json:"id"`
	ShortName        string   `json:"shortName"`
	DisplayName      string   `json:"displayName"`
	Roles            []string `json:"roles"`
	PrimaryAttribute string   `json:"primaryAttribute"`
	AttackType       string   `json:"attackType"`
}
