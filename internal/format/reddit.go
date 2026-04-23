package format

import (
	"fmt"
	"strings"

	"github.com/narrowcastdev/dota-meta/internal/analysis"
)

// FormatReddit generates a Reddit-formatted markdown post from the analysis.
func FormatReddit(result analysis.FullAnalysis, date string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## [Weekly] Heroes that are secretly broken at your rank -- Week of %s\n\n", date)

	fmt.Fprintf(&b, "Every week I pull data from OpenDota (%s matches this week) and break down which heroes are actually winning at each bracket. ", formatNumber(result.TotalMatches))
	b.WriteString("The meta looks pretty different depending on where you're playing — a hero dominating Herald might be throwing games in Immortal.\n\n")
	b.WriteString("Find your bracket below. Heroes need 1000+ picks to qualify so we're not looking at noise.\n\n")

	b.WriteString("---\n\n")

	b.WriteString("### Best heroes by bracket\n\n")
	b.WriteString("The highest win rate heroes at your rank right now.\n\n")
	for _, ba := range result.Brackets {
		fmt.Fprintf(&b, "**%s:**\n\n", ba.Bracket.Name)
		if len(ba.Best) == 0 {
			b.WriteString("No heroes qualified this week.\n\n")
			continue
		}
		b.WriteString("| Hero | Win Rate | Pick Rate | Games |\n")
		b.WriteString("|------|----------|-----------|-------|\n")
		for _, s := range ba.Best {
			fmt.Fprintf(&b, "| %s | %.1f%% | %.1f%% | %s |\n", s.Hero.LocalizedName, s.WinRate, s.PickRate, formatNumber(s.Picks))
		}
		b.WriteString("\n")
	}

	b.WriteString("### Sleeper picks\n\n")
	b.WriteString("These heroes are quietly winning but almost nobody is picking them. Free MMR if they fit your playstyle.\n\n")
	for _, ba := range result.Brackets {
		fmt.Fprintf(&b, "**%s:** ", ba.Bracket.Name)
		if len(ba.Sleepers) == 0 {
			b.WriteString("None this week.\n\n")
			continue
		}
		parts := make([]string, len(ba.Sleepers))
		for i, s := range ba.Sleepers {
			parts[i] = fmt.Sprintf("%s (%.1f%% WR, only %.1f%% pick rate)", s.Hero.LocalizedName, s.WinRate, s.PickRate)
		}
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString("\n\n")
	}

	b.WriteString("### Best support heroes\n\n")
	b.WriteString("Top 5 supports by win rate per bracket. Role tags come from OpenDota's hero metadata, not actual in-game position — flex picks (Visage, Underlord, etc.) can legitimately land here since they're played pos 4/5 in some metas.\n\n")
	for _, ba := range result.Brackets {
		fmt.Fprintf(&b, "**%s:**\n\n", ba.Bracket.Name)
		if len(ba.Supports) == 0 {
			b.WriteString("No supports qualified this week.\n\n")
			continue
		}
		b.WriteString("| Hero | Win Rate | Pick Rate | Games |\n")
		b.WriteString("|------|----------|-----------|-------|\n")
		for _, s := range ba.Supports {
			fmt.Fprintf(&b, "| %s | %.1f%% | %.1f%% | %s |\n", s.Hero.LocalizedName, s.WinRate, s.PickRate, formatNumber(s.Picks))
		}
		b.WriteString("\n")
	}

	b.WriteString("### Trap picks\n\n")
	b.WriteString("Popular heroes that are actually losing more than they win. You're seeing these every game but they're not pulling their weight.\n\n")
	for _, ba := range result.Brackets {
		fmt.Fprintf(&b, "**%s:** ", ba.Bracket.Name)
		if len(ba.Traps) == 0 {
			b.WriteString("None this week.\n\n")
			continue
		}
		parts := make([]string, len(ba.Traps))
		for i, s := range ba.Traps {
			parts[i] = fmt.Sprintf("%s (%.1f%% pick rate, %.1f%% WR)", s.Hero.LocalizedName, s.PickRate, s.WinRate)
		}
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString("\n\n")
	}

	b.WriteString("### Bracket delta\n\n")
	b.WriteString("Same hero, completely different results depending on rank.\n\n")

	b.WriteString("**Low bracket stompers** — these heroes feast in Herald-Guardian but fall off hard at Divine:\n\n")
	if len(result.LowStompers) == 0 {
		b.WriteString("None this week.\n\n")
	} else {
		b.WriteString("| Hero | Herald-Guardian WR | Divine WR | Gap |\n")
		b.WriteString("|------|-------------------|-----------|-----|\n")
		for _, d := range result.LowStompers {
			fmt.Fprintf(&b, "| %s | %.1f%% | %.1f%% | +%.1f%% |\n", d.Hero.LocalizedName, d.LowWR, d.HighWR, d.Delta)
		}
		b.WriteString("\n")
	}

	b.WriteString("**High skill ceiling** — these heroes look bad in low ranks but become monsters at Divine:\n\n")
	if len(result.HighSkillCap) == 0 {
		b.WriteString("None this week.\n\n")
	} else {
		b.WriteString("| Hero | Herald-Guardian WR | Divine WR | Gap |\n")
		b.WriteString("|------|-------------------|-----------|-----|\n")
		for _, d := range result.HighSkillCap {
			fmt.Fprintf(&b, "| %s | %.1f%% | %.1f%% | +%.1f%% |\n", d.Hero.LocalizedName, d.LowWR, d.HighWR, -d.Delta)
		}
		b.WriteString("\n")
	}

	b.WriteString("---\n\n")
	b.WriteString("I also put together an interactive site where you can filter by bracket and sort all heroes: [dota.narrowcast.dev](https://dota.narrowcast.dev)\n\n")
	b.WriteString("See you next week. Let me know if this is useful — and if there's anything you'd want added like item builds, matchup data, or hero trends over time.\n\n")
	b.WriteString("Data from [OpenDota](https://www.opendota.com). Tool is open source: [github.com/narrowcastdev/dota-meta](https://github.com/narrowcastdev/dota-meta)\n")

	return b.String()
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
