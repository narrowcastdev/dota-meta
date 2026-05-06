package format

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/narrowcastdev/dota-meta/internal/analysis"
)

const (
	rankingTopN = 5
	tierTopN    = 5
	siteBase    = "https://dota.narrowcast.dev"
)

var bracketSlug = map[string]string{
	"Herald-Guardian": "herald_guardian",
	"Crusader-Archon": "crusader_archon",
	"Legend-Ancient":  "legend_ancient",
	"Divine":          "divine",
	"Immortal":        "immortal",
}

var tierSlug = map[analysis.Tier]string{
	analysis.TierMetaTyrant: "meta-tyrants",
	analysis.TierPocketPick: "pocket-picks",
	analysis.TierTrap:       "traps",
	analysis.TierDead:       "dead",
}

func bracketLink(name string) string {
	slug, ok := bracketSlug[name]
	if !ok {
		return siteBase
	}
	return siteBase + "/#" + slug
}

func tierLink(bracketName string, t analysis.Tier) string {
	b, ok := bracketSlug[bracketName]
	if !ok {
		return siteBase
	}
	ts, ok := tierSlug[t]
	if !ok {
		return siteBase + "/#" + b
	}
	return siteBase + "/#" + b + "-" + ts
}

var tierEmoji = map[analysis.Tier]string{
	analysis.TierMetaTyrant: "👑",
	analysis.TierPocketPick: "🎯",
	analysis.TierTrap:       "🪤",
	analysis.TierDead:       "💀",
}

var climbEmoji = map[analysis.ClimbTag]string{
	analysis.ClimbUp:        "🧠",
	analysis.ClimbDown:      "🪓",
	analysis.ClimbUniversal: "⚖️",
}

var momentumEmoji = map[analysis.MomentumTag]string{
	analysis.MomentumRising:  "🔥",
	analysis.MomentumFalling: "⚠️",
	analysis.MomentumHidden:  "💎",
	analysis.MomentumDying:   "📉",
}

// FormatRedditWithImages generates a Reddit post where per-bracket tables are
// replaced by image placeholders. The caller uploads the PNGs and substitutes
// the filenames with the real URLs before posting.
func FormatRedditWithImages(result analysis.FullAnalysis, date string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## [Weekly] Dota 2 hero rankings by bracket — %s\n\n", date)
	fmt.Fprintf(&b, "Patch **%s** · %s matches in the last week.\n\n", result.Patch, formatNumber(result.TotalMatches))
	b.WriteString("Tiers computed per-bracket from median WR and pick rate. Same hero can be a Meta Tyrant in Divine and a Trap in Herald. ≥1000 picks to qualify.\n\n")
	b.WriteString("**Legend:** Gold = Meta Tyrant · Blue = Pocket Pick · Red = Trap · Gray = Dead. WR bars centered at 50%.\n\n")
	b.WriteString("---\n\n")

	for _, ba := range result.Brackets {
		slug := bracketSlug[ba.Bracket.Name]
		fmt.Fprintf(&b, "### [%s](%s)\n\n", ba.Bracket.Name, bracketLink(ba.Bracket.Name))
		fmt.Fprintf(&b, "![%s](%s.png)\n\n", ba.Bracket.Name, slug)
	}

	writeMomentumSection(&b, result)

	b.WriteString("---\n\n")
	b.WriteString("Interactive site with full rankings, sparklines, and deltas: [dota.narrowcast.dev](https://dota.narrowcast.dev)\n\n")
	b.WriteString("Data from [STRATZ](https://stratz.com). Tool is open source: [github.com/narrowcastdev/dota-meta](https://github.com/narrowcastdev/dota-meta)\n")

	return b.String()
}

// FormatReddit generates a Reddit-formatted markdown post from the analysis.
func FormatReddit(result analysis.FullAnalysis, date string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## [Weekly] Dota 2 hero rankings by bracket — %s\n\n", date)
	fmt.Fprintf(&b, "Patch **%s** · %s matches in the last week.\n\n", result.Patch, formatNumber(result.TotalMatches))
	b.WriteString("Tiers are computed per-bracket from median win rate and pick rate splits, so the same hero can be a 👑 Meta Tyrant in Divine and a 🪤 Trap in Herald. ≥1000 picks per bracket to qualify.\n\n")
	b.WriteString("**Legend:** 👑 Meta Tyrant · 🎯 Pocket Pick · 🪤 Trap · 💀 Dead · 🔥 rising · ⚠️ falling off · 💎 hidden gem · 📉 dying\n\n")
	b.WriteString("_ΔWR/ΔPR = change vs last snapshot. Momentum = 4-week trend (can point opposite direction)._\n\n")
	b.WriteString("---\n\n")

	for _, ba := range result.Brackets {
		writeBracket(&b, ba)
	}

	writeMomentumSection(&b, result)

	b.WriteString("---\n\n")
	b.WriteString("Interactive site with full rankings, sparklines, and deltas: [dota.narrowcast.dev](https://dota.narrowcast.dev)\n\n")
	b.WriteString("Data from [STRATZ](https://stratz.com). Tool is open source: [github.com/narrowcastdev/dota-meta](https://github.com/narrowcastdev/dota-meta)\n")

	return b.String()
}

func writeBracket(b *strings.Builder, ba analysis.BracketAnalysis) {
	fmt.Fprintf(b, "### [%s](%s)\n\n", ba.Bracket.Name, bracketLink(ba.Bracket.Name))

	cores := sortedBy(ba.Cores, byWinRate)
	supports := sortedBy(ba.Supports, byWinRate)

	writeRankingTable(b, "Top Cores", cores, rankingTopN)
	writeRankingTable(b, "Top Supports", supports, rankingTopN)

	all := append(append([]analysis.HeroStat{}, ba.Cores...), ba.Supports...)
	writeTierTable(b,
		fmt.Sprintf("[👑 Meta Tyrants (ban or first-pick)](%s)", tierLink(ba.Bracket.Name, analysis.TierMetaTyrant)),
		filterTier(all, analysis.TierMetaTyrant),
		func(x, y analysis.HeroStat) bool { return x.WinRate*x.PickRate > y.WinRate*y.PickRate })
	writeTierTable(b,
		fmt.Sprintf("[🎯 Pocket Picks (last-pick counters)](%s)", tierLink(ba.Bracket.Name, analysis.TierPocketPick)),
		filterTier(all, analysis.TierPocketPick), byWinRate)
	writeTierTable(b,
		fmt.Sprintf("[🪤 Traps (popular but losing)](%s)", tierLink(ba.Bracket.Name, analysis.TierTrap)),
		filterTier(all, analysis.TierTrap), byPickRate)

	b.WriteString("---\n\n")
}

func writeRankingTable(b *strings.Builder, title string, heroes []analysis.HeroStat, n int) {
	fmt.Fprintf(b, "**%s (top %d by WR)**\n\n", title, n)
	if len(heroes) == 0 {
		b.WriteString("_No qualifying heroes._\n\n")
		return
	}
	if len(heroes) > n {
		heroes = heroes[:n]
	}
	writeTable(b, heroes, true)
}

func writeTierTable(b *strings.Builder, title string, heroes []analysis.HeroStat, cmp func(a, b analysis.HeroStat) bool) {
	fmt.Fprintf(b, "**%s**\n\n", title)
	if len(heroes) == 0 {
		b.WriteString("_None this week._\n\n")
		return
	}
	sorted := sortedBy(heroes, cmp)
	if len(sorted) > tierTopN {
		sorted = sorted[:tierTopN]
	}
	writeTable(b, sorted, false)
}

func writeTable(b *strings.Builder, heroes []analysis.HeroStat, showRank bool) {
	if showRank {
		b.WriteString("| # | Hero | Tier | WR | ΔWR | PR | ΔPR | Games | Momentum |\n")
		b.WriteString("|---|------|:----:|---:|----:|---:|----:|------:|:--------:|\n")
	} else {
		b.WriteString("| Hero | Tier | WR | ΔWR | PR | ΔPR | Games | Momentum |\n")
		b.WriteString("|------|:----:|---:|----:|---:|----:|------:|:--------:|\n")
	}
	for i, s := range heroes {
		row := ""
		if showRank {
			row = fmt.Sprintf("| %d ", i+1)
		}
		row += fmt.Sprintf("| **%s** | %s | %.1f%% | %s | %.1f%% | %s | %s | %s |\n",
			s.Hero.DisplayName,
			tierCell(s.Tier),
			s.WinRate,
			deltaCell(s.WRDelta),
			s.PickRate,
			deltaCell(s.PRDelta),
			gamesCell(s.Picks),
			momentumCell(s.Momentum),
		)
		b.WriteString(row)
	}
	b.WriteString("\n")
}

func tierCell(t analysis.Tier) string {
	if e, ok := tierEmoji[t]; ok {
		return e
	}
	return "—"
}

func climbCell(c analysis.ClimbTag) string {
	if e, ok := climbEmoji[c]; ok {
		return e
	}
	return "—"
}

func momentumCell(m analysis.MomentumTag) string {
	if e, ok := momentumEmoji[m]; ok {
		return e
	}
	return "—"
}

func deltaCell(v *float64) string {
	if v == nil {
		return "—"
	}
	if math.Abs(*v) < 0.05 {
		return "→ 0.0"
	}
	if *v > 0 {
		return fmt.Sprintf("▲ +%.1f", *v)
	}
	return fmt.Sprintf("▼ %.1f", *v)
}

func gamesCell(picks int) string {
	games := picks / 10 // 10 picks per match
	if games >= 1000 {
		return fmt.Sprintf("%.1fk", float64(games)/1000)
	}
	return fmt.Sprintf("%d", games)
}

func writeMomentumSection(b *strings.Builder, result analysis.FullAnalysis) {
	hasAny := false
	for _, ba := range result.Brackets {
		for _, list := range [][]analysis.HeroStat{ba.Cores, ba.Supports} {
			for _, s := range list {
				if s.Momentum == analysis.MomentumHidden || s.Momentum == analysis.MomentumDying {
					hasAny = true
				}
			}
		}
	}
	if !hasAny {
		return
	}

	b.WriteString("### Momentum watch\n\n")
	writeMomentumTable(b, "💎 Hidden gems — WR climbing, under the radar", result, analysis.MomentumHidden)
	writeMomentumTable(b, "📉 Dying — losing WR and players", result, analysis.MomentumDying)
}

const momentumPerBracket = 3

func writeMomentumTable(b *strings.Builder, title string, result analysis.FullAnalysis, tag analysis.MomentumTag) {
	type entry struct {
		bracket string
		heroes  []analysis.HeroStat
	}
	var groups []entry
	for _, ba := range result.Brackets {
		var matched []analysis.HeroStat
		for _, list := range [][]analysis.HeroStat{ba.Cores, ba.Supports} {
			for _, s := range list {
				if s.Momentum == tag {
					matched = append(matched, s)
				}
			}
		}
		if len(matched) == 0 {
			continue
		}
		sorted := sortedBy(matched, byWinRate)
		if len(sorted) > momentumPerBracket {
			sorted = sorted[:momentumPerBracket]
		}
		groups = append(groups, entry{ba.Bracket.Name, sorted})
	}
	if len(groups) == 0 {
		return
	}

	fmt.Fprintf(b, "**%s**\n\n", title)
	b.WriteString("| Bracket | Hero | WR | ΔWR | PR | ΔPR |\n")
	b.WriteString("|---------|------|---:|----:|---:|----:|\n")
	for _, g := range groups {
		for i, s := range g.heroes {
			bracket := g.bracket
			if i > 0 {
				bracket = ""
			}
			fmt.Fprintf(b, "| %s | **%s** | %.1f%% | %s | %.1f%% | %s |\n",
				bracket, s.Hero.DisplayName,
				s.WinRate, deltaCell(s.WRDelta),
				s.PickRate, deltaCell(s.PRDelta))
		}
	}
	b.WriteString("\n")
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
