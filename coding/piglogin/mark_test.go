package piglogin_test

import (
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/tmih06/ompig/coding/piglogin"
	"github.com/tmih06/ompig/tui"
	"github.com/tmih06/ompig/tui/widthx"
)

const reset = "\x1b[0m"

func rgb(t *testing.T, c color.RGBA) tui.Color {
	t.Helper()
	value, err := tui.NewRgbColor(float64(c.R), float64(c.G), float64(c.B))
	if err != nil {
		t.Fatal(err)
	}
	return value
}

// letter is the expected color of letter n: ramp rows 7 to 9, or the period color where those are too pale for a light background.
// letter is the color TextMark gives letter n of a word of length letters: the
// ramp stop its position maps to, else the sprite's period color.
func letter(logo piglogin.Logo, n, letters int) color.RGBA {
	rows := [3]int{7, 8, 9}
	index := 0
	if letters > 1 {
		index = n * (len(rows) - 1) / (letters - 1)
	}
	if index < 0 {
		index = 0
	}
	if index >= len(rows) {
		index = len(rows) - 1
	}
	c := logo.Ramp[rows[index]]
	l := lum(c)
	if 1.05/(l+0.05) >= 3 && (l+0.05)/0.05 >= 3 {
		return c
	}
	return logo.Period
}

func lum(c color.RGBA) float64 {
	lin := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// The text mark stands in for the art where it cannot be drawn. It spells the product's name, so it is as wide as
// "Ompig." (six cells), each letter takes a stop of the sprite's ramp and the period takes the sprite's period color.
func TestTextMarkIsALegibleOneLineOmpig(t *testing.T) {
	if piglogin.TextMarkWidth != 6 {
		t.Fatalf("TextMarkWidth = %d, want the width of \"Ompig.\" (6 cells)", piglogin.TextMarkWidth)
	}
	const word = "Ompig"
	for _, variant := range piglogin.Variants {
		for _, mode := range []tui.TerminalColorMode{tui.TerminalColorModeTrueColor, tui.TerminalColorMode256} {
			mark := piglogin.TextMark(variant, mode)
			logo := piglogin.LogoFor(variant)
			fg := func(c color.RGBA) string { return tui.ForegroundAnsi(rgb(t, c), mode) }
			want := "\x1b[1m"
			for n := range len(word) {
				want += fg(letter(logo, n, len(word))) + string(word[n])
			}
			want += fg(logo.Period) + "." + reset
			if mark != want {
				t.Errorf("%s/%s: text mark = %q, want %q", variant.ID, mode, mark, want)
			}
			if widthx.VisibleWidth(mark) != piglogin.TextMarkWidth {
				t.Errorf("%s/%s: text mark is %d cells wide", variant.ID, mode, widthx.VisibleWidth(mark))
			}
			if mode == tui.TerminalColorMode256 && strings.Contains(mark, "38;2;") {
				t.Errorf("%s: 256-color text mark contains a truecolor sequence", variant.ID)
			}
		}
	}
}

// The default text mark's period is the website wordmark's green accent, the pixel art's period color.
func TestDefaultTextMarkPeriodIsTheWordmarkAccent(t *testing.T) {
	mark := piglogin.TextMark(piglogin.Default(), tui.TerminalColorModeTrueColor)
	if !strings.Contains(mark, "\x1b[38;2;22;134;111m.") {
		t.Errorf("default text mark = %q, want the period in #16866F", mark)
	}
	if strings.Contains(mark, "38;2;72;163;129") {
		t.Errorf("default text mark = %q colors the letters with the pale-on-light-unsafe body color", mark)
	}
}

// Every letter and the period reads on both a white and a black background, whatever the sprite's ramp.
func TestTextMarkLettersAreLegibleOnLightAndDark(t *testing.T) {
	for _, variant := range piglogin.Variants {
		logo := piglogin.LogoFor(variant)
		for n := range 5 {
			l := lum(letter(logo, n, 5))
			if 1.05/(l+0.05) < 3 || (l+0.05)/0.05 < 3 {
				t.Errorf("%s letter %d is not legible on both backgrounds", variant.ID, n)
			}
		}
	}
}

// Every sprite draws a distinct login: no two catalogue entries share their mascot pixels and palette.
func TestSpritesDiffer(t *testing.T) {
	seen := map[string]string{}
	for _, variant := range piglogin.Variants {
		definition := piglogin.LoginDefinitionFor(variant)
		var key strings.Builder
		key.WriteString(strings.Join(definition.Mascot, "\n"))
		for _, row := range append(append([]string(nil), definition.Hero...), definition.Mascot...) {
			for i := range len(row) {
				key.WriteString(definition.Palette[string(row[i])])
			}
		}
		if other, dup := seen[key.String()]; dup {
			t.Errorf("%s and %s draw the same login", variant.ID, other)
		}
		seen[key.String()] = variant.ID
	}
}

func TestSheriffWearsTheHat(t *testing.T) {
	sheriff, ok := piglogin.ByID("sheriff")
	if !ok {
		t.Fatal("no sheriff sprite")
	}
	definition := piglogin.LoginDefinitionFor(sheriff)
	if definition.Palette["H"] != "#6B4C3A" || !strings.Contains(definition.Mascot[3], "HHHHHHHHHHHHHHHH") {
		t.Errorf("the sheriff's login has no hat brim: palette H = %q, row 3 = %q", definition.Palette["H"], definition.Mascot[3])
	}
	if _, hat := piglogin.LoginDefinitionFor(piglogin.Default()).Palette["H"]; hat {
		t.Error("the default ompig wears the sheriff's hat")
	}
}
