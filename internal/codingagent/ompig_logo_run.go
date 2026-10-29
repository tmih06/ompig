package codingagent

import (
	"math"
	"strings"

	"github.com/tmih06/ompig/coding/piglogin"
)

// ompig divergence (D87): where Pi's logo plays its sliding puzzle, the spinning head gives way to a side-view ompig in the
// active sprite's colors, ray cast in braille like the head. It turns to face the camera, runs in place and turns back into
// the head's spin when Pi's shuffle ends.

// ompigRunBody is the running ompig without legs, facing right, in pixels: outline ('O'), inner ear ('e'), back highlight
// ('p'), body ('P'), belly shade ('D'), eye ('W' and 'K'), cheek ('b') and snout ('s') with nostrils ('K').
var ompigRunBody = [...]string{
	"......................OOOO.....OOOO...",
	"......................OeeeOOOOOeeeO...",
	"......................OeeePPPPPeeeO...",
	".OOO..................OpppppppPPPPPO..",
	"OPPPO..OOOOOOOOOOOOOOOPPPPPPPPPPPPPPO.",
	"OPO.O.OPPpppppppppppPPPPPPPPWWPPPPPPO.",
	".OPPOOPpppPPPPPPPPPPPPPPPPPPWKPPPPPPO.",
	"....OPPPPPPPPPPPPPPPPPPPPPPPPPPPssssO.",
	"....OPPPPPPPPPPPPPPPPPPPPPPPPPPssssssO",
	"....OPPPPPPPPPPPPPPPPPPPPPPPPPPssKsKsO",
	"....OPPPPPPPPPPPPPPPPPPPPPPPbbbPsssssO",
	"....OPPPPPPPPPPPPPPPPPPPPPPPPPPPssssO.",
	"....OPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPO..",
	"....OPPPPPPPPPPPPPPPPPPPPPPPPPPPPPO...",
	".....ODDDDDDDDDDDDDDDDDDDDOOOOOOOO....",
	"......ODDDDDDDDDDDDDDDDDDDO...........",
	".......OOOOOOOOOOOOOOOOOOO............",
}

const (
	ompigRunWidth = 38
	// ompigRunLegRows is the height of a leg below the body's last row; the leg's first row overlaps that row.
	ompigRunLegRows = 4
	// ompigRunHeight is the pixel height of the running ompig: the body, the legs and one pixel of bob.
	ompigRunHeight = len(ompigRunBody) + ompigRunLegRows + 1
	// ompigRunStep is the time of one run-cycle frame.
	ompigRunStep = logoPuzzleStep / 3
)

// ompigRunLegColumns are the left columns of the four legs: far hind, near hind, far front, near front.
var ompigRunLegColumns = [4]int{6, 10, 20, 24}

// ompigRunFrames is the run cycle: each leg's horizontal reach at the hoof, in pixels, and how far the body is lifted.
var ompigRunFrames = [4]struct {
	reach [4]int
	lift  int
}{
	{reach: [4]int{-2, -3, 3, 2}, lift: 0},
	{reach: [4]int{0, -1, 1, 0}, lift: 1},
	{reach: [4]int{2, 3, -3, -2}, lift: 0},
	{reach: [4]int{0, 1, -1, 0}, lift: 1},
}

// Accessory overlays keep a character's look on the running ompig: '.' keeps the base pixel. Each symbol takes the sprite's
// palette color.
var ompigRunAccessories = map[string][]string{
	// The sheriff's hat with its gold star, on the head.
	"sheriff": {
		"........................HHHHHHHH......",
		"........................HhhSShhH......",
		"....................HHHHHHHHHHHHHHHH..",
	},
	// Vader's red chest panel lights.
	"darth-vader": {
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		".....................bbsb.............",
	},
	// Kratos's red tattoo from the back over the head.
	"kratos": {
		"",
		"",
		"",
		"..........................RRR.........",
		"........................RRR...........",
		".........RRRRRRRRRRRRRRRR.............",
		".......RRR............................",
		".......R..............................",
		".......R..............................",
		".......RR.............................",
		"........RRR...........................",
	},
	// Ompiglet's striped shirt on the body.
	"ompiglet": {
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"......RRRRRRRRRRRRRRRRRRRR............",
		"......rrrrrrrrrrrrrrrrrrrr............",
		"......RRRRRRRRRRRRRRRRRRRR............",
		"......rrrrrrrrrrrrrrrrrrrr............",
		"......RRRRRRRRRRRRRRRRRRRR............",
		"......rrrrrrrrrrrrrrrrrrr.............",
	},
	// Spider-Ham's web lines.
	"spider-ham": {
		"",
		"",
		"",
		"",
		"",
		"..........r.......r.......r...........",
		"..........r.......r.......r...........",
		"......rrrrrrrrrrrrrrrrrrrrrr..........",
		"..........r.......r.......r...........",
		"..........r.......r.......r...........",
		"......rrrrrrrrrrrrrrrrrrrrrr..........",
		"..........r.......r.......r...........",
		"..........r.......r.......r...........",
	},
	// PiGrogu's robe over the body.
	"pigrogu": {
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"......BBBBBBBBBBBBBBBBBBB.............",
		"......BBBBBBBBBBBBBBBBBBBB............",
		"......BBBBBBBBBBBBBBBBBBBBB...........",
		"......BBBBBBBBBBBBBBBBBBBB............",
		"......bbbbbbbbbbbbbbbbbbb.............",
	},
}

// ompigRunHats are the accessory symbols drawn over the outline and outside the body; other accessory pixels color only the
// body's inside.
var ompigRunHats = map[byte]bool{'H': true, 'h': true, 'S': true}

// ompigRunPalette is the running ompig's colors for variant. A symbol the head draws takes the head's color; otherwise the
// body is the head's most common color other than its outline (Kratos's skin, Spider-Ham's suit) and the highlight, shade,
// snout, cheek and inner ear are mixed from it. A body too dark for a dark outline (Vader) is outlined in its highlight.
func ompigRunPalette(variant piglogin.Variant) map[byte]logoRgb {
	palette := piglogin.MascotPalette(variant)
	drawn := map[byte]bool{}
	counts := map[byte]int{}
	var common byte
	for _, pixel := range piglogin.HeadPixels(variant) {
		drawn[pixel.Symbol] = true
		counts[pixel.Symbol]++
		if pixel.Symbol != 'O' && (common == 0 || counts[pixel.Symbol] > counts[common]) {
			common = pixel.Symbol
		}
	}
	toRgb := func(symbol byte) logoRgb {
		value := palette[symbol]
		return logoRgb{float64(value.R), float64(value.G), float64(value.B)}
	}
	colors := map[byte]logoRgb{}
	for symbol := range palette {
		if symbol != '.' {
			colors[symbol] = toRgb(symbol)
		}
	}
	body := toRgb(common)
	if drawn['P'] {
		body = toRgb('P')
	}
	colors['P'] = body
	derive := func(symbol byte, from logoRgb, amount float64) {
		if !drawn[symbol] {
			colors[symbol] = logoMix(body, from, amount)
		}
	}
	derive('p', logoRgb{255, 255, 255}, 0.3)
	derive('s', logoRgb{0, 0, 0}, 0.2)
	derive('b', logoRgb{232, 120, 140}, 0.5)
	derive('e', logoRgb{0, 0, 0}, 0.12)
	colors['D'] = logoMix(body, logoRgb{0, 0, 0}, 0.22)
	//portlint:allow fusedmuladd D87: the running ompig is Ompig's own, with no Pi counterpart to match bit for bit
	if 0.299*body[0]+0.587*body[1]+0.114*body[2] < 48 {
		colors['O'] = toRgb('p')
		colors['D'] = toRgb('s')
	}
	if _, ok := colors['W']; !ok {
		colors['W'] = logoRgb{255, 255, 255}
	}
	if _, ok := colors['K']; !ok {
		colors['K'] = colors['O']
	}
	return colors
}

// ompigRunPixels is frame of the running ompig as ompigRunWidth by ompigRunHeight pixels, each a palette symbol or '.'.
func ompigRunPixels(variantID string, frame int) [][]byte {
	f := ompigRunFrames[frame%len(ompigRunFrames)]
	grid := make([][]byte, ompigRunHeight)
	for y := range grid {
		grid[y] = []byte(strings.Repeat(".", ompigRunWidth))
	}
	top := 1 - f.lift
	legTop := top + len(ompigRunBody) - 1
	drawLeg := func(leg int, inner byte) {
		for row := range ompigRunLegRows + 1 {
			shift := int(math.Round(float64(f.reach[leg]*row) / float64(ompigRunLegRows)))
			x := ompigRunLegColumns[leg] + shift
			y := legTop + row
			if y >= ompigRunHeight {
				break
			}
			for dx, symbol := range []byte{'O', inner, inner, inner, 'O'} {
				if row == ompigRunLegRows {
					symbol = 'O'
				}
				if column := x + dx; column >= 0 && column < ompigRunWidth {
					grid[y][column] = symbol
				}
			}
		}
	}
	// The far legs are behind the body and shaded; the near legs are drawn over the body's lower edge.
	drawLeg(0, 'D')
	drawLeg(2, 'D')
	accessory := ompigRunAccessories[variantID]
	for y, row := range ompigRunBody {
		for x := range len(row) {
			symbol := row[x]
			if y < len(accessory) && x < len(accessory[y]) && accessory[y][x] != '.' && (ompigRunHats[accessory[y][x]] || (symbol != '.' && symbol != 'O' && symbol != 'W' && symbol != 'K')) {
				symbol = accessory[y][x]
			}
			if symbol != '.' {
				grid[top+y][x] = symbol
			}
		}
	}
	drawLeg(1, 'P')
	drawLeg(3, 'P')
	return grid
}

func ompigRunPixel(grid [][]byte, colors map[byte]logoRgb, x, y int) *logoRgb {
	if y < 0 || y >= len(grid) || x < 0 || x >= len(grid[y]) || grid[y][x] == '.' {
		return nil
	}
	color, ok := colors[grid[y][x]]
	if !ok {
		color = colors['P']
	}
	return &color
}

// ompigRunner is the running ompig of one render: its frame's pixels and colors, how far it has turned from the head's yaw to
// face the camera (facing, 0 to 1), and how far it has replaced the head (mix, 0 to 1).
type ompigRunner struct {
	grid   [][]byte
	colors map[byte]logoRgb
	facing float64
	mix    float64
	// local is the time since the run started.
	local float64
}

// ompigRunFade is the time the head's dots take to dissolve into the running ompig's, and back: at the start and end of the
// run, and when the exit starts during the run.
const ompigRunFade = 0.6

// runner is the running ompig at time, or nil while only the head shows: from the start of each puzzle cycle until Pi's
// shuffle ends, and during the exit while a running ompig dissolves back into the head. It runs in place where the head
// spins.
func (a *ompigLogoAnimation) runner(time float64) *ompigRunner {
	// Only the logo's header ompig runs; a model with a puzzle plays Pi's puzzle.
	if a.model.puzzleMoves != nil {
		return nil
	}
	if a.exit != nil {
		run := a.exit.run
		if run == nil {
			return nil
		}
		since := a.now().Sub(a.exit.start).Seconds()
		fade := 1 - logoSmooth(since/ompigRunFade)
		if fade <= 0 {
			return nil
		}
		local := run.local + since
		return &ompigRunner{grid: ompigRunPixels(a.variant.ID, int(math.Floor(local/ompigRunStep))), colors: run.colors, facing: run.facing * fade, mix: run.mix * fade, local: local}
	}
	local, ok := a.runTime(time)
	if !ok {
		return nil
	}
	if a.runColors == nil {
		a.runColors = ompigRunPalette(a.variant)
	}
	weight := logoSmooth(local/ompigRunFade) * (1 - logoSmooth((local-(logoShuffleEnd-ompigRunFade))/ompigRunFade))
	if weight <= 0 {
		return nil
	}
	return &ompigRunner{
		grid:   ompigRunPixels(a.variant.ID, int(math.Floor(local/ompigRunStep))),
		colors: a.runColors,
		facing: weight,
		mix:    weight,
		local:  local,
	}
}

// pose is the running ompig's pose: the head's, turned facing of the way to the nearest view facing the camera, and scaled
// for its blocks (appendBoxes).
func (r *ompigRunner) pose(head logoPose) logoPose {
	turn := math.Pi * 2
	//portlint:allow fusedmuladd D87: the running ompig is Ompig's own, with no Pi counterpart to match bit for bit
	head.yaw += (math.Round(head.yaw/turn)*turn - head.yaw) * r.facing
	head.pitch *= 1 - r.facing
	head.roll *= 1 - r.facing
	// appendBoxes builds the ompig at 2*ompigLogoCenterX/ompigRunWidth per block; this draws a block the size of a head block.
	head.scale *= ompigLogoPixel * ompigRunWidth / (2 * ompigLogoCenterX)
	return head
}

// ompigRunDotThreshold is a dot's place in the dissolve between the head and the running ompig, from 0 to 1.
func ompigRunDotThreshold(dot int) float64 {
	return logoHash(float64(dot)*0.7548776662 + 0.5698402910) //portlint:allow fusedmuladd D87: Ompig's running ompig, no Pi counterpart
}

// appendBoxes appends a block per pixel of the running ompig, centered on the origin, with the head's depth.
func (r *ompigRunner) appendBoxes(boxes []logoBox) []logoBox {
	// size is a block's side in object space: the ompig is as long there as the head is wide, so its ends come no closer to
	// the camera than the head's when it turns. With head-sized blocks its ends would reach within half a unit of the
	// camera at logoCameraDistance and blow up across the screen. pose scales the view back up, so a block still
	// projects to the size of a head block.
	const size = 2 * ompigLogoCenterX / ompigRunWidth
	// Both edges come from pixel indices, so neighbors share a plane exactly and hide each other's inner faces.
	edgeX := func(x int) float64 { return (float64(x) - ompigRunWidth/2.0) * size }
	edgeY := func(y int) float64 { return (float64(y) - float64(ompigRunHeight)/2) * size }
	for y := range r.grid {
		for x := range r.grid[y] {
			color := ompigRunPixel(r.grid, r.colors, x, y)
			if color == nil {
				continue
			}
			boxes = append(boxes, logoBox{
				min:   [3]float64{edgeX(x), edgeY(y), -logoDepth / 2},
				max:   [3]float64{edgeX(x + 1), edgeY(y + 1), logoDepth / 2},
				color: *color,
			})
		}
	}
	return boxes
}

// runTime is the time since the run started in the current puzzle cycle, and whether the ompig is running.
func (a *ompigLogoAnimation) runTime(time float64) (float64, bool) {
	if a.exit != nil || time < logoPuzzleStart {
		return 0, false
	}
	cycle := math.Floor((time - logoPuzzleStart) / logoPuzzleCycle)
	local := time - logoPuzzleStart - cycle*logoPuzzleCycle //portlint:allow fusedmuladd D87: Ompig's running ompig, no Pi counterpart
	return local, local < logoShuffleEnd
}
