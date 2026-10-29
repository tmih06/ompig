// Package parity provides the test harness for TUI byte-level and visible-state
// parity against upstream pi-tui.
//
// Two test styles:
//
//  1. Headless byte parity (`AssertByteParity`): runs a renderer function
//     against an in-process Grid, captures the output stream, and asserts the
//     resulting grid state matches the upstream render of the same component.
//     Used for fast unit-level parity checks of individual components.
//
//  2. Live tmux parity (`AssertLiveParity`): drives both `~/.local/bin/ompig`
//     and the installed upstream pi binary through a tmux scenario, captures
//     panes, and asserts the visible state matches.
//
// TUI parity is enforced by test/parity/scenarios/*.tmux scenarios.
package parity

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tmih06/ompig/internal/tui/termsim"
)

// Capture runs render into an in-process Grid of the given dimensions and
// returns the final grid plus the raw byte stream.
func Capture(rows, cols int, render func(w *bytes.Buffer)) (*termsim.Grid, []byte) {
	var buf bytes.Buffer
	render(&buf)
	g := termsim.New(rows, cols)
	g.Write(buf.Bytes())
	return g, buf.Bytes()
}

// AssertByteParity runs `ompigRender` and `upstreamRender` into independent
// grids and asserts the resulting visible state is identical line-for-line.
// Both functions receive a fresh buffer they should write their full render
// output into.
//
// On failure, prints a side-by-side diff. Style differences are NOT compared
// here (rendering colour is a separate axis verified by AssertStyleParity).
func AssertByteParity(t *testing.T, rows, cols int, ompigRender, upstreamRender func(*bytes.Buffer)) {
	t.Helper()
	gGrid, gBytes := Capture(rows, cols, ompigRender)
	uGrid, uBytes := Capture(rows, cols, upstreamRender)
	if gGrid.String() != uGrid.String() {
		t.Errorf("byte parity failed.\n--- ompig (visible) ---\n%s\n--- upstream (visible) ---\n%s",
			withRowMarkers(gGrid.String()), withRowMarkers(uGrid.String()))
		t.Logf("ompig raw bytes: %d, upstream raw bytes: %d", len(gBytes), len(uBytes))
	}
}

func withRowMarkers(s string) string {
	var b strings.Builder
	for i, line := range strings.Split(s, "\n") {
		b.WriteString(formatRow(i, line))
		b.WriteByte('\n')
	}
	return b.String()
}

func formatRow(i int, line string) string {
	return "  " + padInt(i, 3) + "│" + line + "│"
}

func padInt(n, width int) string {
	s := ""
	for n > 0 || len(s) < width {
		if n == 0 {
			s = " " + s
			continue
		}
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
