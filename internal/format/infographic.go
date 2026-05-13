package format

import (
	"fmt"
	"image"
	"math"
	"sort"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/narrowcastdev/dota-meta/internal/analysis"
)

// BracketImage holds a rendered bracket infographic and its metadata.
type BracketImage struct {
	Name  string
	Slug  string
	Image image.Image
}

const (
	biW      = 800
	biH      = 500
	biPad    = 24.0
	biGap    = 16.0
	biIconSz = 24
	biRow    = 38.0
)

const (
	cBg     = "#0d1117"
	cBorder = "#30363d"
	cText   = "#e6edf3"
	cMuted  = "#8b949e"
	cGold   = "#f0c040"
	cCyan   = "#58a6ff"
	cCoral  = "#f47067"
	cGray   = "#484f58"
	cGreen  = "#3fb950"
	cRed    = "#f85149"
	cAccent = "#1f6feb"
	cBarBg  = "#21262d"
)

func tierColor(t analysis.Tier) string {
	switch t {
	case analysis.TierMetaTyrant:
		return cGold
	case analysis.TierPocketPick:
		return cCyan
	case analysis.TierTrap:
		return cCoral
	default:
		return cGray
	}
}

type fontCache struct {
	head, label, body, bodyB, small font.Face
}

func parseFonts() (*fontCache, error) {
	regular, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("parse regular font: %w", err)
	}
	bold, err := truetype.Parse(gobold.TTF)
	if err != nil {
		return nil, fmt.Errorf("parse bold font: %w", err)
	}
	mk := func(f *truetype.Font, size float64) font.Face {
		return truetype.NewFace(f, &truetype.Options{Size: size, DPI: 72})
	}
	return &fontCache{
		head:  mk(bold, 20),
		label: mk(bold, 12),
		body:  mk(regular, 14),
		bodyB: mk(bold, 14),
		small: mk(regular, 12),
	}, nil
}

type bracketRenderer struct {
	dc    *gg.Context
	fonts *fontCache
	icons map[string]image.Image
}

// FormatBracketImages renders one infographic per bracket. icons is optional (nil OK).
func FormatBracketImages(result analysis.FullAnalysis, date string, icons map[string]image.Image) ([]BracketImage, error) {
	fonts, err := parseFonts()
	if err != nil {
		return nil, err
	}

	scaled := make(map[string]image.Image, len(icons))
	for name, img := range icons {
		scaled[name] = prepareIcon(img, biIconSz)
	}

	var out []BracketImage
	for _, ba := range result.Brackets {
		slug := bracketSlug[ba.Bracket.Name]
		if slug == "" {
			slug = ba.Bracket.Name
		}

		dc := gg.NewContext(biW, biH)
		r := &bracketRenderer{dc: dc, fonts: fonts, icons: scaled}
		r.render(ba, result.Patch, date)

		out = append(out, BracketImage{
			Name:  ba.Bracket.Name,
			Slug:  slug,
			Image: dc.Image(),
		})
	}

	return out, nil
}

func (r *bracketRenderer) render(ba analysis.BracketAnalysis, patch, date string) {
	dc := r.dc
	w := float64(biW)

	dc.SetHexColor(cBg)
	dc.Clear()

	dc.SetHexColor(cAccent)
	dc.DrawRectangle(0, 0, w, 3)
	dc.Fill()

	dc.SetFontFace(r.fonts.head)
	dc.SetHexColor(cText)
	dc.DrawString(ba.Bracket.Name, biPad, 28)

	dc.SetFontFace(r.fonts.small)
	dc.SetHexColor(cMuted)
	matchStr := fmt.Sprintf("%s matches", formatNumber(ba.Matches()))
	mw, _ := dc.MeasureString(matchStr)
	dc.DrawString(matchStr, w-biPad-mw, 28)

	sub := fmt.Sprintf("Patch %s · %s · WR from 50%% center", patch, date)
	dc.DrawString(sub, biPad, 48)

	r.drawLegend(biPad, 68)

	dc.SetHexColor(cBorder)
	dc.SetLineWidth(1)
	dc.DrawLine(biPad, 82, w-biPad, 82)
	dc.Stroke()

	colW := (w - 2*biPad - biGap) / 2
	leftX := biPad
	rightX := biPad + colW + biGap
	contentTop := 92.0

	cores := sortedBy(ba.Cores, byWinRate)
	leftY := r.drawSection(leftX, contentTop, colW, "TOP CORES", cGold, cores, 5)

	all := mergedByWR(ba.Cores, ba.Supports)
	traps := filterTier(all, analysis.TierTrap)
	sort.Slice(traps, func(i, j int) bool { return traps[i].PickRate > traps[j].PickRate })
	_ = r.drawSection(leftX, leftY+4, colW, "POPULAR BUT LOSING", cCoral, traps, 2)

	supports := sortedBy(ba.Supports, byWinRate)
	rightY := r.drawSection(rightX, contentTop, colW, "TOP SUPPORTS", cCyan, supports, 5)

	shown := collectShown(cores, 5, supports, 5, traps, 2)
	trending := filterMomentumExcluding(all, shown, analysis.MomentumRising, analysis.MomentumHidden)
	if len(trending) > 0 {
		_ = r.drawSection(rightX, rightY+4, colW, "TRENDING", cGreen, trending, 2)
	}

	r.drawFooter()
}

func (r *bracketRenderer) drawLegend(x, y float64) {
	dc := r.dc
	type entry struct {
		color, name string
	}
	items := []entry{
		{cGold, "Meta Tyrant"},
		{cCyan, "Pocket Pick"},
		{cCoral, "Trap"},
		{cGray, "Dead"},
	}

	dc.SetFontFace(r.fonts.small)
	cx := x
	for _, it := range items {
		dc.SetHexColor(it.color)
		dc.DrawCircle(cx+4, y, 4)
		dc.Fill()

		dc.SetHexColor(cMuted)
		dc.DrawString(it.name, cx+12, y+4)

		tw, _ := dc.MeasureString(it.name)
		cx += 12 + tw + 16
	}
}

func (r *bracketRenderer) drawSection(x, y, w float64, title, color string, heroes []analysis.HeroStat, maxN int) float64 {
	dc := r.dc

	dc.SetFontFace(r.fonts.label)
	dc.SetHexColor(color)
	dc.DrawString(title, x, y+10)

	cursor := y + 18

	n := maxN
	if n > len(heroes) {
		n = len(heroes)
	}
	for i := 0; i < n; i++ {
		r.drawHeroRow(x, cursor, w, heroes[i])
		cursor += biRow
	}

	return cursor + 4
}

func (r *bracketRenderer) drawHeroRow(x, y, w float64, s analysis.HeroStat) {
	dc := r.dc
	iconF := float64(biIconSz)

	if icon, ok := r.icons[s.Hero.ShortName]; ok {
		r.drawCircularIcon(x, y, icon)
	} else {
		dc.SetHexColor(tierColor(s.Tier))
		dc.DrawCircle(x+iconF/2, y+iconF/2, 5)
		dc.Fill()
	}

	textX := x + iconF + 6
	dc.SetFontFace(r.fonts.body)
	dc.SetHexColor(cText)
	dc.DrawString(s.Hero.DisplayName, textX, y+16)

	rightEdge := x + w
	deltaW := 0.0
	if s.WRDelta != nil && math.Abs(*s.WRDelta) >= 0.05 {
		dc.SetFontFace(r.fonts.small)
		var ds string
		if *s.WRDelta > 0 {
			dc.SetHexColor(cGreen)
			ds = fmt.Sprintf("+%.1f", *s.WRDelta)
		} else {
			dc.SetHexColor(cRed)
			ds = fmt.Sprintf("%.1f", *s.WRDelta)
		}
		dw, _ := dc.MeasureString(ds)
		dc.DrawString(ds, rightEdge-dw, y+16)
		deltaW = dw + 4
	}

	dc.SetFontFace(r.fonts.bodyB)
	dc.SetHexColor(cText)
	wrText := fmt.Sprintf("%.1f%%", s.WinRate)
	wrW, _ := dc.MeasureString(wrText)
	dc.DrawString(wrText, rightEdge-deltaW-wrW, y+16)

	barY := y + 28
	barH := 5.0

	dc.SetHexColor(cBarBg)
	dc.DrawRoundedRectangle(x, barY, w, barH, 2)
	dc.Fill()

	centerX := x + w/2
	deviation := (s.WinRate - 50) / 8.0
	if deviation > 1 {
		deviation = 1
	}
	if deviation < -1 {
		deviation = -1
	}
	fillW := math.Abs(deviation) * (w / 2)
	if fillW > 2 {
		dc.SetHexColor(tierColor(s.Tier))
		if deviation >= 0 {
			dc.DrawRectangle(centerX, barY, fillW, barH)
		} else {
			dc.DrawRectangle(centerX-fillW, barY, fillW, barH)
		}
		dc.Fill()
	}

	dc.SetHexColor(cGray)
	dc.SetLineWidth(1)
	dc.DrawLine(centerX, barY, centerX, barY+barH)
	dc.Stroke()
}

func (r *bracketRenderer) drawCircularIcon(x, y float64, icon image.Image) {
	dc := r.dc
	half := float64(biIconSz) / 2

	dc.DrawImage(icon, int(x), int(y))

	dc.SetHexColor(cBorder)
	dc.SetLineWidth(1.5)
	dc.DrawCircle(x+half, y+half, half)
	dc.Stroke()
}

func (r *bracketRenderer) drawFooter() {
	dc := r.dc
	w := float64(biW)
	h := float64(biH)

	dc.SetHexColor(cBorder)
	dc.SetLineWidth(1)
	dc.DrawLine(biPad, h-40, w-biPad, h-40)
	dc.Stroke()

	dc.SetFontFace(r.fonts.small)
	dc.SetHexColor(cMuted)
	dc.DrawStringAnchored("Data from STRATZ · github.com/narrowcastdev/dota-meta",
		w/2, h-24, 0.5, 0.5)

	dc.SetFontFace(r.fonts.label)
	dc.SetHexColor(cAccent)
	dc.DrawStringAnchored("dota.narrowcast.dev", w/2, h-8, 0.5, 0.5)
}

func mergedByWR(cores, supports []analysis.HeroStat) []analysis.HeroStat {
	all := make([]analysis.HeroStat, 0, len(cores)+len(supports))
	all = append(all, cores...)
	all = append(all, supports...)
	sort.Slice(all, func(i, j int) bool { return all[i].WinRate > all[j].WinRate })
	return all
}

func collectShown(cores []analysis.HeroStat, coreMax int, supports []analysis.HeroStat, supMax int, traps []analysis.HeroStat, trapMax int) map[string]bool {
	shown := make(map[string]bool)
	add := func(list []analysis.HeroStat, max int) {
		n := max
		if n > len(list) {
			n = len(list)
		}
		for i := 0; i < n; i++ {
			shown[list[i].Hero.DisplayName] = true
		}
	}
	add(cores, coreMax)
	add(supports, supMax)
	add(traps, trapMax)
	return shown
}

func filterMomentumExcluding(stats []analysis.HeroStat, exclude map[string]bool, tags ...analysis.MomentumTag) []analysis.HeroStat {
	tagSet := make(map[analysis.MomentumTag]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}
	var out []analysis.HeroStat
	for _, s := range stats {
		if tagSet[s.Momentum] && !exclude[s.Hero.DisplayName] {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].WinRate > out[j].WinRate })
	return out
}

func prepareIcon(src image.Image, size int) image.Image {
	scaled := scaleImage(src, size)
	mask := gg.NewContext(size, size)
	half := float64(size) / 2
	mask.DrawCircle(half, half, half)
	mask.Clip()
	mask.DrawImageAnchored(scaled, size/2, size/2, 0.5, 0.5)
	return mask.Image()
}

func scaleImage(src image.Image, size int) image.Image {
	bounds := src.Bounds()
	side := bounds.Dy()
	if bounds.Dx() < side {
		side = bounds.Dx()
	}
	ox := bounds.Min.X + (bounds.Dx()-side)/2
	oy := bounds.Min.Y + (bounds.Dy()-side)/2
	srcRect := image.Rect(ox, oy, ox+side, oy+side)

	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.BiLinear.Scale(dst, dst.Bounds(), src, srcRect, xdraw.Over, nil)
	return dst
}
