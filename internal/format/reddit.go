package format

import (
	"fmt"
	"sort"
	"strings"

	"github.com/narrowcastdev/dota-meta/internal/analysis"
)

const (
	rankingTopN = 5
	supportTopN = 5
	deltaTopN   = 5
	deltaMinWR  = 1.5 // percentage points; ignore tiny gaps
)

// FormatReddit generates a Reddit-formatted markdown post from the analysis.
// Plain markdown tables, no emoji — pre-STRATZ format with STRATZ attribution.
func FormatReddit(result analysis.FullAnalysis, date string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## [Weekly] Heroes that are secretly broken at your rank -- Week of %s\n\n", date)

	fmt.Fprintf(&b, "Every week I pull data from STRATZ (%s matches this week, patch %s) and break down which heroes are actually winning at each bracket. ",
		formatNumber(result.TotalMatches), result.Patch)
	b.WriteString("The meta looks pretty different depending on where you're playing — a hero dominating Herald might be throwing games in Immortal.\n\n")
	b.WriteString("Find your bracket below. Heroes need 1000+ picks to qualify so we're not looking at noise.\n\n")

	b.WriteString("---\n\n")

	b.WriteString("### Best heroes by bracket\n\n")
	b.WriteString("The highest win rate heroes at your rank right now.\n\n")
	for _, ba := range result.Brackets {
		fmt.Fprintf(&b, "**%s:**\n\n", ba.Bracket.Name)
		all := combinedSorted(ba, byWinRate)
		if len(all) == 0 {
			b.WriteString("No heroes qualified this week.\n\n")
			continue
		}
		if len(all) > rankingTopN {
			all = all[:rankingTopN]
		}
		writePlainTable(&b, all)
	}

	b.WriteString("### Sleeper picks\n\n")
	b.WriteString("These heroes are quietly winning but almost nobody is picking them. Free MMR if they fit your playstyle.\n\n")
	for _, ba := range result.Brackets {
		fmt.Fprintf(&b, "**%s:** ", ba.Bracket.Name)
		picks := tierFiltered(ba, analysis.TierPocketPick)
		picks = sortedBy(picks, byWinRate)
		if len(picks) > rankingTopN {
			picks = picks[:rankingTopN]
		}
		if len(picks) == 0 {
			b.WriteString("None this week.\n\n")
			continue
		}
		parts := make([]string, len(picks))
		for i, s := range picks {
			parts[i] = fmt.Sprintf("%s (%.1f%% WR, only %.1f%% pick rate)", s.Hero.DisplayName, s.WinRate, s.PickRate)
		}
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString("\n\n")
	}

	b.WriteString("### Best support heroes\n\n")
	b.WriteString("Top supports by win rate per bracket. Role tags come from STRATZ hero metadata, not actual in-game position — flex picks (Visage, Underlord, etc.) can land here since they're played pos 4/5 in some metas.\n\n")
	for _, ba := range result.Brackets {
		fmt.Fprintf(&b, "**%s:**\n\n", ba.Bracket.Name)
		sups := sortedBy(ba.Supports, byWinRate)
		if len(sups) == 0 {
			b.WriteString("No supports qualified this week.\n\n")
			continue
		}
		if len(sups) > supportTopN {
			sups = sups[:supportTopN]
		}
		writePlainTable(&b, sups)
	}

	b.WriteString("### Trap picks\n\n")
	b.WriteString("Popular heroes that are actually losing more than they win. You're seeing these every game but they're not pulling their weight.\n\n")
	for _, ba := range result.Brackets {
		fmt.Fprintf(&b, "**%s:** ", ba.Bracket.Name)
		traps := tierFiltered(ba, analysis.TierTrap)
		traps = sortedBy(traps, byPickRate)
		if len(traps) > rankingTopN {
			traps = traps[:rankingTopN]
		}
		if len(traps) == 0 {
			b.WriteString("None this week.\n\n")
			continue
		}
		parts := make([]string, len(traps))
		for i, s := range traps {
			parts[i] = fmt.Sprintf("%s (%.1f%% pick rate, %.1f%% WR)", s.Hero.DisplayName, s.PickRate, s.WinRate)
		}
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString("\n\n")
	}

	writeBracketDelta(&b, result)

	b.WriteString("---\n\n")
	b.WriteString("I also put together an interactive site where you can filter by bracket and sort all heroes: [dota.narrowcast.dev](https://dota.narrowcast.dev)\n\n")
	b.WriteString("See you next week. Let me know if this is useful — and if there's anything you'd want added like item builds, matchup data, or hero trends over time.\n\n")
	b.WriteString("Data from [STRATZ](https://stratz.com). Tool is open source: [github.com/narrowcastdev/dota-meta](https://github.com/narrowcastdev/dota-meta)\n")

	return b.String()
}

func writePlainTable(b *strings.Builder, heroes []analysis.HeroStat) {
	b.WriteString("| Hero | Win Rate | Pick Rate | Games |\n")
	b.WriteString("|------|----------|-----------|-------|\n")
	for _, s := range heroes {
		fmt.Fprintf(b, "| %s | %.1f%% | %.1f%% | %s |\n",
			s.Hero.DisplayName, s.WinRate, s.PickRate, formatNumber(s.Picks/10))
	}
	b.WriteString("\n")
}

// writeBracketDelta emits the "same hero, different bracket" section by
// comparing Herald-Guardian vs Divine win rates per hero.
func writeBracketDelta(b *strings.Builder, result analysis.FullAnalysis) {
	type pair struct {
		hero            string
		lowWR, highWR   float64
		hasLow, hasHigh bool
	}
	pairs := map[int]*pair{}
	for _, ba := range result.Brackets {
		var target string
		switch ba.Bracket.Name {
		case "Herald-Guardian":
			target = "low"
		case "Divine":
			target = "high"
		default:
			continue
		}
		for _, list := range [][]analysis.HeroStat{ba.Cores, ba.Supports} {
			for _, s := range list {
				p, ok := pairs[s.Hero.ID]
				if !ok {
					p = &pair{hero: s.Hero.DisplayName}
					pairs[s.Hero.ID] = p
				}
				if target == "low" {
					p.lowWR = s.WinRate
					p.hasLow = true
				} else {
					p.highWR = s.WinRate
					p.hasHigh = true
				}
			}
		}
	}

	type delta struct {
		hero string
		low  float64
		high float64
		diff float64 // low - high; positive = stomper, negative = high-skill
	}
	var stompers, ceilings []delta
	for _, p := range pairs {
		if !(p.hasLow && p.hasHigh) {
			continue
		}
		d := delta{hero: p.hero, low: p.lowWR, high: p.highWR, diff: p.lowWR - p.highWR}
		if d.diff >= deltaMinWR {
			stompers = append(stompers, d)
		} else if -d.diff >= deltaMinWR {
			ceilings = append(ceilings, d)
		}
	}
	sort.Slice(stompers, func(i, j int) bool { return stompers[i].diff > stompers[j].diff })
	sort.Slice(ceilings, func(i, j int) bool { return ceilings[i].diff < ceilings[j].diff })
	if len(stompers) == 0 && len(ceilings) == 0 {
		return
	}

	b.WriteString("### Bracket delta\n\n")
	b.WriteString("Same hero, completely different results depending on rank.\n\n")

	b.WriteString("**Low bracket stompers** — these heroes feast in Herald-Guardian but fall off hard at Divine:\n\n")
	if len(stompers) == 0 {
		b.WriteString("None this week.\n\n")
	} else {
		if len(stompers) > deltaTopN {
			stompers = stompers[:deltaTopN]
		}
		b.WriteString("| Hero | Herald-Guardian WR | Divine WR | Gap |\n")
		b.WriteString("|------|-------------------|-----------|-----|\n")
		for _, d := range stompers {
			fmt.Fprintf(b, "| %s | %.1f%% | %.1f%% | +%.1f%% |\n", d.hero, d.low, d.high, d.diff)
		}
		b.WriteString("\n")
	}

	b.WriteString("**High skill ceiling** — these heroes look bad in low ranks but become monsters at Divine:\n\n")
	if len(ceilings) == 0 {
		b.WriteString("None this week.\n\n")
	} else {
		if len(ceilings) > deltaTopN {
			ceilings = ceilings[:deltaTopN]
		}
		b.WriteString("| Hero | Herald-Guardian WR | Divine WR | Gap |\n")
		b.WriteString("|------|-------------------|-----------|-----|\n")
		for _, d := range ceilings {
			fmt.Fprintf(b, "| %s | %.1f%% | %.1f%% | +%.1f%% |\n", d.hero, d.low, d.high, -d.diff)
		}
		b.WriteString("\n")
	}
}

func combinedSorted(ba analysis.BracketAnalysis, cmp func(a, b analysis.HeroStat) bool) []analysis.HeroStat {
	all := append(append([]analysis.HeroStat{}, ba.Cores...), ba.Supports...)
	return sortedBy(all, cmp)
}

func tierFiltered(ba analysis.BracketAnalysis, t analysis.Tier) []analysis.HeroStat {
	var out []analysis.HeroStat
	for _, list := range [][]analysis.HeroStat{ba.Cores, ba.Supports} {
		for _, s := range list {
			if s.Tier == t {
				out = append(out, s)
			}
		}
	}
	return out
}

func sortedBy(stats []analysis.HeroStat, cmp func(a, b analysis.HeroStat) bool) []analysis.HeroStat {
	out := append([]analysis.HeroStat{}, stats...)
	sort.Slice(out, func(i, j int) bool { return cmp(out[i], out[j]) })
	return out
}

func byWinRate(a, b analysis.HeroStat) bool  { return a.WinRate > b.WinRate }
func byPickRate(a, b analysis.HeroStat) bool { return a.PickRate > b.PickRate }

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}
