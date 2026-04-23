package format

import (
	"fmt"
	"sort"
	"strings"

	"github.com/narrowcastdev/dota-meta/internal/analysis"
)

// FormatReddit generates a Reddit-formatted markdown post from the analysis.
func FormatReddit(result analysis.FullAnalysis, date string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## [Weekly] Heroes that are secretly broken at your rank -- Week of %s\n\n", date)
	fmt.Fprintf(&b, "Patch %s · %s matches analyzed.\n\n", result.Patch, formatNumber(result.TotalMatches))
	b.WriteString("Tiers are computed per-bracket from median win rate and pick rate splits — so the same hero can be a Meta Tyrant in Divine and a Trap in Herald. ≥1000 picks per bracket to qualify.\n\n---\n\n")

	for _, ba := range result.Brackets {
		fmt.Fprintf(&b, "### %s\n\n", ba.Bracket.Name)
		all := append(append([]analysis.HeroStat{}, ba.Cores...), ba.Supports...)
		writeSection(&b, "Ban list (Meta Tyrants)", filterTier(all, analysis.TierMetaTyrant),
			func(a, b analysis.HeroStat) bool { return a.WinRate*a.PickRate > b.WinRate*b.PickRate })
		writeSection(&b, "Pocket picks (sleepers)", filterTier(all, analysis.TierPocketPick),
			func(a, b analysis.HeroStat) bool { return a.WinRate > b.WinRate })
		writeSection(&b, "Stop picking (Traps)", filterTier(all, analysis.TierTrap),
			func(a, b analysis.HeroStat) bool { return a.PickRate > b.PickRate })
	}

	rising := collectRising(result)
	if len(rising) > 0 {
		b.WriteString("### Rising this week\n\n")
		for _, s := range rising {
			fmt.Fprintf(&b, "- **%s** — %.1f%% WR, %.1f%% PR (slope up in both)\n", s.Hero.DisplayName, s.WinRate, s.PickRate)
		}
		b.WriteString("\n")
	}

	b.WriteString("---\n\n")
	b.WriteString("Interactive site with all tiers, sparklines, and deltas: [dota.narrowcast.dev](https://dota.narrowcast.dev)\n\n")
	b.WriteString("Data from [STRATZ](https://stratz.com). Tool is open source: [github.com/narrowcastdev/dota-meta](https://github.com/narrowcastdev/dota-meta)\n")

	return b.String()
}

func filterTier(stats []analysis.HeroStat, tier analysis.Tier) []analysis.HeroStat {
	var out []analysis.HeroStat
	for _, s := range stats {
		if s.Tier == tier {
			out = append(out, s)
		}
	}
	return out
}

func writeSection(b *strings.Builder, title string, heroes []analysis.HeroStat, cmp func(a, b analysis.HeroStat) bool) {
	fmt.Fprintf(b, "**%s:** ", title)
	if len(heroes) == 0 {
		b.WriteString("none this week.\n\n")
		return
	}
	sort.Slice(heroes, func(i, j int) bool { return cmp(heroes[i], heroes[j]) })
	if len(heroes) > 3 {
		heroes = heroes[:3]
	}
	parts := make([]string, len(heroes))
	for i, s := range heroes {
		parts[i] = fmt.Sprintf("%s (%.1f%% WR, %.1f%% PR)", s.Hero.DisplayName, s.WinRate, s.PickRate)
	}
	b.WriteString(strings.Join(parts, ", "))
	b.WriteString("\n\n")
}

func collectRising(result analysis.FullAnalysis) []analysis.HeroStat {
	seen := map[string]bool{}
	var out []analysis.HeroStat
	for _, ba := range result.Brackets {
		for _, list := range [][]analysis.HeroStat{ba.Cores, ba.Supports} {
			for _, s := range list {
				if s.Momentum != analysis.MomentumRising {
					continue
				}
				k := ba.Bracket.Name + "|" + s.Hero.DisplayName
				if seen[k] {
					continue
				}
				seen[k] = true
				out = append(out, s)
			}
		}
	}
	return out
}

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
