package pdf

import (
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

type Design struct {
	sizeCompany float64
	sizeSubject float64
	sizeBody    float64
	sizeSmall   float64
	sizeMicro   float64
	sizeTotal   float64

	ink      *props.Color
	muted    *props.Color
	hairline *props.Color

	defaultAccent *props.Color

	infoOpacity      float64
	tableHeadOpacity float64
	zebraOpacity     float64
	totalOpacity     float64

	letterheadHeight float64
	logoOnRight      bool
	logoScale        float64
	letterheadRule   float64

	tableHeadRule   bool
	tableRowRule    bool
	tableHeadUpper  bool
	tableRowSpacing float64

	totalBorder border.Type
	totalRule   float64

	footerRuleIsAccent bool
	totalInkIsAccent   bool
	monochrome         bool

	// Computed by resolve().
	accent       *props.Color
	infoPanelBg  *props.Color
	tableHeadBg  *props.Color
	tableHeadInk *props.Color
	zebraBg      *props.Color
	totalBg      *props.Color
	totalInk     *props.Color
}

var (
	inkBlack  = &props.Color{Red: 17, Green: 24, Blue: 39}
	inkGrey   = &props.Color{Red: 107, Green: 114, Blue: 128}
	ruleGrey  = &props.Color{Red: 203, Green: 208, Blue: 214}
	ruleLight = &props.Color{Red: 228, Green: 231, Blue: 235}

	defaultBrandColor = &props.Color{Red: 32, Green: 84, Blue: 132}
)

func (t Design) resolve(brand *props.Color) Design {
	accent := brand
	if t.monochrome {
		// Reference classic: no colour anywhere, even with a brand colour set.
		accent = nil
	}
	if accent == nil {
		accent = t.defaultAccent
	}
	if accent == nil {
		accent = inkBlack
	}
	t.accent = accent

	surface := func(opacity float64) *props.Color {
		if opacity <= 0 {
			return nil
		}
		return tint(accent, opacity)
	}
	t.infoPanelBg = surface(t.infoOpacity)
	t.tableHeadBg = surface(t.tableHeadOpacity)
	t.zebraBg = surface(t.zebraOpacity)
	t.totalBg = surface(t.totalOpacity)

	t.tableHeadInk = readableInk(t.tableHeadBg, t.ink)
	t.totalInk = readableInk(t.totalBg, t.ink)
	return t
}

func tint(c *props.Color, opacity float64) *props.Color {
	if opacity > 1 {
		opacity = 1
	}
	mix := func(channel int) int {
		return int(float64(255) + (float64(channel)-255)*opacity + 0.5)
	}
	return &props.Color{Red: mix(c.Red), Green: mix(c.Green), Blue: mix(c.Blue)}
}

func readableInk(background, dark *props.Color) *props.Color {
	if background == nil {
		return dark
	}
	luminance := (299*float64(background.Red) + 587*float64(background.Green) + 114*float64(background.Blue)) / 1000
	if luminance < 128 {
		return &props.Color{Red: 255, Green: 255, Blue: 255}
	}
	return dark
}

// Invalid values return nil; resolve falls back to the default.
func parseHexColor(value string) *props.Color {
	if len(value) > 0 && value[0] == '#' {
		value = value[1:]
	}
	if len(value) != 6 {
		return nil
	}
	channel := func(hi, lo byte) (int, bool) {
		digit := func(b byte) (int, bool) {
			switch {
			case b >= '0' && b <= '9':
				return int(b - '0'), true
			case b >= 'a' && b <= 'f':
				return int(b-'a') + 10, true
			case b >= 'A' && b <= 'F':
				return int(b-'A') + 10, true
			}
			return 0, false
		}
		high, okHigh := digit(hi)
		low, okLow := digit(lo)
		return high*16 + low, okHigh && okLow
	}
	red, okRed := channel(value[0], value[1])
	green, okGreen := channel(value[2], value[3])
	blue, okBlue := channel(value[4], value[5])
	if !okRed || !okGreen || !okBlue {
		return nil
	}
	return &props.Color{Red: red, Green: green, Blue: blue}
}

var Classic = Design{
	sizeCompany: 13,
	sizeSubject: 12,
	sizeBody:    10,
	sizeSmall:   8.5,
	sizeMicro:   7,
	sizeTotal:   11,

	ink:           inkBlack,
	muted:         inkGrey,
	hairline:      ruleGrey,
	defaultAccent: inkBlack,

	monochrome: true,

	letterheadHeight: 20,
	logoOnRight:      false,
	logoScale:        85,
	letterheadRule:   0.3,

	tableHeadRule:   true,
	tableRowRule:    true,
	tableHeadUpper:  false,
	tableRowSpacing: 7,

	totalBorder: border.None,
	totalRule:   0.3,
}

var Modern = Design{
	sizeCompany: 16,
	sizeSubject: 15,
	sizeBody:    9.5,
	sizeSmall:   8.5,
	sizeMicro:   7,
	sizeTotal:   12,

	ink:           inkBlack,
	muted:         inkGrey,
	hairline:      ruleLight,
	defaultAccent: defaultBrandColor,

	infoOpacity:      0,
	tableHeadOpacity: 0.16,
	zebraOpacity:     0,
	totalOpacity:     0,

	letterheadHeight: 20,
	logoOnRight:      true,
	logoScale:        85,
	letterheadRule:   0.6,

	tableHeadRule:   false,
	tableRowRule:    true,
	tableHeadUpper:  false,
	tableRowSpacing: 8,

	totalRule:          0.4,
	footerRuleIsAccent: true,
	totalInkIsAccent:   true,
}

var Minimal = Design{
	sizeCompany: 18,
	sizeSubject: 12,
	sizeBody:    9.5,
	sizeSmall:   8,
	sizeMicro:   6.5,
	sizeTotal:   16,

	ink:           inkBlack,
	muted:         inkGrey,
	hairline:      ruleLight,
	defaultAccent: inkBlack,

	letterheadHeight: 24,
	logoScale:        62,

	tableHeadRule:   true,
	tableHeadUpper:  true,
	tableRowSpacing: 8.5,

	totalRule: 0.6,
}
