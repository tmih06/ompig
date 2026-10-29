package piglogin

import (
	"image/color"
	"math"
	"strings"

	"github.com/tmih06/ompig/tui"
)

// TextMarkWidth is the cell width of the text mark: the five letters of "Ompig" plus its period. The mark replaced Pi's
// two-letter wordmark, which occupied 4 cells beside its logo, so the header pads hints by this width.
const TextMarkWidth = 6

const (
	markReset = "\x1b[0m"
	markBold  = "\x1b[1m"
)

func markColor(value color.RGBA) tui.Color {
	// The channels are bytes, inside the 0..255 range NewRgbColor accepts.
	c, err := tui.NewRgbColor(float64(value.R), float64(value.G), float64(value.B))
	if err != nil {
		panic(err)
	}
	return c
}

// markLetterRows are the Ramp rows that color the text mark's letters: the mid-to-dark stops, where the classic ramp is
// legible on both light and dark backgrounds (contrast of at least 3.7 against white and black); its top cyan is too light
// for white. The mark is "Ompig." (five letters), so the ramp is sampled across these stops rather than one stop per letter.
var markLetterRows = [3]int{7, 8, 9}

// minMarkContrast is the least WCAG contrast a letter color must have against both white and black.
const minMarkContrast = 3.0 // ompig divergence (D2): Ompig's own text-mark colors; Pi's wordmark uses fixed colors.

func luminance(c color.RGBA) float64 {
	lin := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	//portlint:allow fusedmuladd D2: Ompig's own text-mark contrast test, no Pi counterpart; a last-bit change cannot cross minMarkContrast for the ramp's colors
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// legibleOnLightAndDark reports whether c reads on both a white and a black background.
func legibleOnLightAndDark(c color.RGBA) bool {
	l := luminance(c)
	return 1.05/(l+0.05) >= minMarkContrast && (l+0.05)/0.05 >= minMarkContrast
}

// markLetterColor is the color of letter n of the "Ompig." mark: the Ramp stop its position maps to, or the wordmark's
// period color when the ramp is too pale or too dark there (the default sprite's flat near-white ramp is made for a dark
// background). Letters are distributed over the ramp so the mark keeps the wordmark's gradient at any letter count.
func markLetterColor(logo Logo, n, letters int) color.RGBA {
	if len(markLetterRows) == 0 {
		return logo.Period
	}
	index := 0
	if letters > 1 {
		index = n * (len(markLetterRows) - 1) / (letters - 1)
	}
	if index < 0 {
		index = 0
	}
	if index >= len(markLetterRows) {
		index = len(markLetterRows) - 1
	}
	row := markLetterRows[index]
	if row < 0 || row >= len(logo.Ramp) {
		return logo.Period
	}
	if c := logo.Ramp[row]; legibleOnLightAndDark(c) {
		return c
	}
	return logo.Period
}

// TextMark is the one-line "Ompig." mark for a terminal that cannot draw the pixel art. "Ompig" is bold and each letter takes a
// mid or dark stop of the variant's wordmark ramp, the way Pi's text wordmark colors "Pi"; a stop that would not read on both
// light and dark backgrounds gives way to the period color. The period takes the wordmark's period color.
func TextMark(variant Variant, mode tui.TerminalColorMode) string {
	logo := LogoFor(variant)
	const word = "Ompig"
	var b strings.Builder
	b.WriteString(markBold)
	for n, letter := range word {
		b.WriteString(tui.ForegroundAnsi(markColor(markLetterColor(logo, n, len(word))), mode))
		b.WriteRune(letter)
	}
	b.WriteString(tui.ForegroundAnsi(markColor(logo.Period), mode) + "." + markReset)
	return b.String()
}
