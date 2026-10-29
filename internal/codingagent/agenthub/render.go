package agenthub

// The roster renderer, ported from omp's packages/tui/src/overlays/agent-hub-renderer.ts
// and the #renderEntry / #statusSummary / #renderDetailPanel methods of
// packages/tui/src/overlays/agent-hub.ts.
//
// omp's visual contract, kept exactly:
//
//   ❯ ⟳ Agent  ↳ Parent  read-only  ⧉ 3      role · model ◒ high
//       task summary, one line, muted
//       $0.01 · 2m3s active · 4 req · 12 tools · 3.4k tok · 5m
//
// A row is one to three lines: identity, the task, then the metric columns.
// Every metric column has a fixed width so the values line up under each other
// across rows, and the age closes the line.

import (
	"fmt"
	"strings"

	"github.com/tmih06/ompig/coding/subagent"
	"github.com/tmih06/ompig/tui"
	"github.com/tmih06/ompig/tui/widthx"
)

// Symbol presets, from omp's packages/tui/src/theme/symbols.ts. The default
// (nerd/unicode) set is what a modern terminal shows; ascii is the fallback the
// caller can select.
const (
	glyphRunning = "⟳"
	glyphIdle    = "●"
	glyphParked  = "○"
	glyphAborted = "⏹"
	glyphCursor  = "❯"
	glyphDot     = "·"
	glyphUnread  = "⧉"
	glyphParent  = "↳"
)

// Metric column widths, from formatMetricColumns.
const (
	colCost     = 8
	colDuration = 13
	colRequests = 8
	colTools    = 9
	colTokens   = 8
	colAge      = 8
)

// SPLIT_MIN_WIDTH is where the roster gains an inspector beside it (omp's
// SPLIT_MIN_WIDTH).
const SPLIT_MIN_WIDTH = 96

// DETAIL_MIN_WIDTH is the narrowest usable inspector pane (omp's constant).
const DETAIL_MIN_WIDTH = 34

// metricRowWidth is the width of the metric line before its indent: the six
// fixed cells and the five single-space separators.
const metricRowWidth = colCost + colDuration + colRequests + colTools + colTokens + colAge + 5

// detailIndent is where a row's task and metric lines start.
const detailIndent = 4

// statusGlyph is omp's status symbol for a status.
func statusGlyph(status subagent.Status) string {
	switch status {
	case subagent.StatusRunning:
		return glyphRunning
	case subagent.StatusIdle:
		return glyphIdle
	case subagent.StatusParked:
		return glyphParked
	default:
		return glyphAborted
	}
}

// statusToken is the theme token omp colors a status with.
func statusToken(status subagent.Status) string {
	switch status {
	case subagent.StatusRunning:
		return "accent"
	case subagent.StatusIdle:
		return "success"
	case subagent.StatusParked:
		return "muted"
	default:
		return "error"
	}
}

// statusGlyphStyled is the glyph in its status color.
func statusGlyphStyled(theme *tui.Theme, status subagent.Status) string {
	return theme.Fg(statusToken(status), statusGlyph(status))
}

// statusText is a status word in its status color.
func statusText(theme *tui.Theme, status subagent.Status, text string) string {
	return theme.Fg(statusToken(status), text)
}

// statusSummary is omp's "#statusSummary": every non-zero count as
// "<glyph> <n> <status>", joined by the separator.
func statusSummary(theme *tui.Theme, entries []subagent.Entry) string {
	var parts []string
	for _, status := range []subagent.Status{subagent.StatusRunning, subagent.StatusIdle, subagent.StatusParked, subagent.StatusAborted} {
		count := 0
		for _, entry := range entries {
			if entry.Status == status {
				count++
			}
		}
		if count == 0 {
			continue
		}
		// Four counts at most, once per repaint.
		parts = append(parts, statusGlyphStyled(theme, status)+" "+statusText(theme, status, fmt.Sprintf("%d %s", count, status))) //perf:ok a display path, not the hot path
	}
	return strings.Join(parts, themeSep(theme))
}

// themeSep is omp's theme.sep.dot, which renders as " · ".
func themeSep(theme *tui.Theme) string {
	return theme.Fg("dim", " "+glyphDot+" ")
}

// formatCost matches omp: four decimals under a cent, three under a dollar,
// two above.
func formatCost(cost float64) string {
	switch {
	case cost < 0.01:
		return fmt.Sprintf("$%.4f", cost)
	case cost < 1:
		return fmt.Sprintf("$%.3f", cost)
	default:
		return fmt.Sprintf("$%.2f", cost)
	}
}

// formatMetricDuration is omp's "2m3s active" / "12s span" / "—".
func formatMetricDuration(entry subagent.Entry, now int64) string {
	if entry.DurationMs <= 0 && entry.Running() {
		return "—"
	}
	kind := "duration"
	if entry.Running() {
		kind = "active"
	} else if entry.DurationMs > 0 {
		kind = "span"
	}
	if entry.DurationMs <= 0 {
		return "—"
	}
	return formatDurationMs(entry.DurationMs) + " " + kind
}

// formatDurationMs matches omp's formatDuration: 950ms, 3s, 2m3s, 1h2m.
func formatDurationMs(ms int64) string {
	switch {
	case ms < 1000:
		return fmt.Sprintf("%dms", ms)
	case ms < 60_000:
		return fmt.Sprintf("%ds", ms/1000)
	case ms < 3_600_000:
		return fmt.Sprintf("%dm%ds", ms/60_000, (ms/1000)%60)
	default:
		return fmt.Sprintf("%dh%dm", ms/3_600_000, (ms/60_000)%60)
	}
}

// formatNumber abbreviates a count as omp does: below 1000 plain, then k, then M.
func formatNumber(value int64) string {
	switch {
	case value < 1000:
		return fmt.Sprintf("%d", value)
	case value < 1_000_000:
		return trimZero(float64(value)/1000) + "k"
	default:
		return trimZero(float64(value)/1_000_000) + "M"
	}
}

// trimZero renders one decimal, dropping a trailing ".0".
func trimZero(value float64) string {
	text := fmt.Sprintf("%.1f", value)
	return strings.TrimSuffix(text, ".0")
}

// formatAge matches omp's formatAge for a staleness span in seconds.
func formatAge(seconds int64) string {
	switch {
	case seconds < 1:
		return "now"
	case seconds < 60:
		return fmt.Sprintf("%ds", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%dm", seconds/60)
	case seconds < 86_400:
		return fmt.Sprintf("%dh", seconds/3600)
	default:
		return fmt.Sprintf("%dd", seconds/86_400)
	}
}

// cell right- or left-aligns text in a fixed column, truncating overflow as
// renderTableRow does. Padding is measured in visible cells, so a multi-byte
// character such as the em dash occupies one cell, not its byte count.
func cell(text string, width int, right bool) string {
	if visible := widthx.VisibleWidth(text); visible > width {
		text = truncate(text, width)
	}
	pad := strings.Repeat(" ", max(0, width-widthx.VisibleWidth(text)))
	if right {
		return pad + text
	}
	return text + pad
}

// metricColumns is omp's formatMetricColumns: six fixed-width cells, the cost
// flush left and the numbers right-aligned, separated by one space.
func metricColumns(entry subagent.Entry, now int64) string {
	age := formatAge((now - entry.LastActivityAt.UnixMilli()) / 1000)
	return strings.Join([]string{
		cell(formatCost(entry.Usage.Cost), colCost, false),
		cell(formatMetricDuration(entry, now), colDuration, true),
		cell(formatNumber(entry.Usage.Requests)+" req", colRequests, true),
		cell(formatNumber(entry.Usage.ToolCalls)+" tools", colTools, true),
		cell(formatNumber(entry.Usage.Total)+" tok", colTokens, true),
		cell(age, colAge, true),
	}, " ")
}

// metricsText is omp's formatMetrics, the flowing variant the inspector uses.
func metricsText(theme *tui.Theme, entry subagent.Entry, now int64) string {
	sep := themeSep(theme)
	return strings.Join([]string{
		formatCost(entry.Usage.Cost),
		formatMetricDuration(entry, now),
		formatNumber(entry.Usage.Requests) + " req",
		formatNumber(entry.Usage.ToolCalls) + " tools",
		formatNumber(entry.Usage.Total) + " tok",
	}, sep)
}

// modelBadge is omp's modelBadge: the model id in muted, then the thinking
// level in its own tone when the model reasons.
func modelBadge(theme *tui.Theme, entry subagent.Entry) string {
	if entry.Model == "" {
		return ""
	}
	badge := theme.Fg("muted", entry.Model)
	if level := entry.ThinkingLevel; level != "" && level != "off" && level != "inherit" {
		badge += " " + theme.Fg("warning", level)
	}
	return badge
}

// contextGauge is omp's contextGauge: a ten-cell bar, the counts and the
// percentage.
func contextGauge(theme *tui.Theme, tokens, window int) string {
	if window <= 0 {
		return ""
	}
	ratio := float64(tokens) / float64(window)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * 10)
	bar := theme.Fg("accent", strings.Repeat("━", filled)) + theme.Fg("dim", strings.Repeat("─", 10-filled))
	return fmt.Sprintf("%s %d/%d %d%%", bar, tokens, window, int(ratio*100))
}

// truncate cuts a line to width visible cells, appending an ellipsis when it
// had to. Styling is preserved: an SGR sequence is copied through and does not
// count against the budget, and a cut inside styled text closes the styling it
// inherited so the ellipsis does not bleed the row's color.
func truncate(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if widthx.VisibleWidth(text) <= width {
		return text
	}
	var b strings.Builder
	visible := 0
	// One cell is held back for the ellipsis.
	limit := width - 1
	var active []string
	for i := 0; i < len(text) && visible < limit; {
		if text[i] == 0x1b && i+1 < len(text) && text[i+1] == '[' {
			end := i + 2
			for end < len(text) && text[end] != 'm' {
				end++
			}
			if end < len(text) {
				seq := text[i : end+1]
				b.WriteString(seq)
				if seq == "\x1b[0m" || strings.HasSuffix(seq, "0m") {
					active = active[:0]
				} else {
					active = append(active, seq)
				}
				i = end + 1
				continue
			}
		}
		b.WriteByte(text[i])
		visible++
		i++
	}
	b.WriteString("…")
	if len(active) > 0 {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}
