package agenthub

import (
	"strings"
	"testing"
	"time"

	"github.com/tmih06/ompig/coding/subagent"
	"github.com/tmih06/ompig/tui"
	"github.com/tmih06/ompig/tui/widthx"
)

// newOMPRegistry builds the roster omp's docs describe: a running agent with a
// tool in flight, an idle one, and a killed one.
func newOMPRegistry(t *testing.T) (*subagent.Registry, time.Time) {
	t.Helper()
	registry := subagent.NewRegistry()
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	registry.SetClock(func() time.Time { return now })

	running := registry.Register(subagent.Entry{
		Name: "Scout", AgentType: "scout", Task: "Complete assignment thoroughly: map the repo",
		Model: "openai/gpt-5", ThinkingLevel: "high", Status: subagent.StatusRunning,
	})
	registry.Update(running.ID, func(e *subagent.Entry) {
		e.CurrentTool = "grep"
		e.CurrentIntent = "Searching callers"
		e.Usage = subagent.Usage{Total: 3400, Cost: 0.012, Requests: 4, ToolCalls: 12}
	})
	idle := registry.Register(subagent.Entry{Name: "Writer", AgentType: "task", Task: "Write the doc", Status: subagent.StatusIdle})
	registry.Update(idle.ID, func(e *subagent.Entry) { e.DurationMs = 123000; e.Usage = subagent.Usage{Total: 900, Cost: 0.004} })
	killed := registry.Register(subagent.Entry{Name: "Broken", AgentType: "task", Task: "fail", Status: subagent.StatusAborted})
	registry.Update(killed.ID, func(e *subagent.Entry) { e.Error = "provider 500" })
	return registry, now
}

func renderOMP(t *testing.T, width int, wide bool) string {
	t.Helper()
	registry, now := newOMPRegistry(t)
	model := New(Options{Registry: registry, Width: width, Height: 24,
		Now: func() time.Time { return now }})
	model.SetWide(wide)
	return strings.Join(model.Render(width), "\n")
}

// TestHubRosterHeaderMatchesOMP covers omp's summary line: "Roster · Flat /
// By parent · <counts>" and the aggregate usage line beneath it.
func TestHubRosterHeaderMatchesOMP(t *testing.T) {
	text := renderOMP(t, 120, false)
	plain := stripAnsi(text)
	for _, want := range []string{"Roster", "Flat", "By parent", "1 running", "1 idle", "1 aborted"} {
		if !strings.Contains(plain, want) {
			t.Errorf("header missing %q:\n%s", want, plain)
		}
	}
	if !strings.Contains(plain, "measured") {
		t.Errorf("aggregate usage line missing:\n%s", plain)
	}
}

// TestHubRowMatchesOMPLayout proves every roster row carries omp's three parts:
// "<cursor> <glyph> <id>", the task on its own line, and the metric columns.
func TestHubRowMatchesOMPLayout(t *testing.T) {
	plain := stripAnsi(renderOMP(t, 120, false))
	lines := strings.Split(plain, "\n")

	var scoutIdentity, scoutTask, scoutMetrics int
	for i, line := range lines {
		if strings.Contains(line, "Scout") && strings.Contains(line, glyphRunning) {
			scoutIdentity = i
		}
	}
	if scoutIdentity == 0 {
		t.Fatalf("no Scout identity row:\n%s", plain)
	}
	// Identity carries the model badge on the right.
	if !strings.Contains(lines[scoutIdentity], "openai/gpt-5") || !strings.Contains(lines[scoutIdentity], "high") {
		t.Errorf("identity row lacks the model badge: %q", lines[scoutIdentity])
	}
	// The task line drops the assignment preamble, as omp's taskSummary does.
	scoutTask = scoutIdentity + 1
	if strings.Contains(lines[scoutTask], "Complete assignment thoroughly") {
		t.Errorf("task line kept the assignment preamble: %q", lines[scoutTask])
	}
	if !strings.Contains(lines[scoutTask], "map the repo") {
		t.Errorf("task line = %q, want the task", lines[scoutTask])
	}
	// The metric line carries all six columns in omp's order.
	scoutMetrics = scoutIdentity + 2
	for _, want := range []string{"$0.012", "req", "tools", "tok"} {
		if !strings.Contains(lines[scoutMetrics], want) {
			t.Errorf("metric line %q missing %q", lines[scoutMetrics], want)
		}
	}
	// Columns are fixed width, so the cost cell starts every metric line at the
	// same offset.
	// Only the metric lines start with the cost cell, so measure those: each is
	// "<indent>$" with the same indent, which is what alignment means here.
	var offsets []int
	for _, line := range lines {
		// Only the roster's metric line has this shape: indent, then a cost
		// cell followed by the duration cell. The inspector's usage line is not
		// a column grid.
		trimmed := strings.TrimLeft(line, " ")
		if !strings.HasPrefix(trimmed, "$") {
			continue
		}
		if widthx.VisibleWidth(line) < detailIndent+metricRowWidth-1 {
			continue
		}
		offsets = append(offsets, strings.Index(line, "$"))
	}
	if len(offsets) < 2 {
		t.Fatalf("expected at least two metric lines, got %d:\n%s", len(offsets), plain)
	}
	for _, offset := range offsets {
		if offset != offsets[0] {
			t.Errorf("metric columns are not aligned: offsets %v", offsets)
			break
		}
	}
}

// TestHubMetricColumnsFixedWidth proves the six cells have omp's widths and
// alignment, so values line up across rows.
func TestHubMetricColumnsFixedWidth(t *testing.T) {
	registry, now := newOMPRegistry(t)
	entries := registry.List()
	line := stripAnsi(metricColumns(entries[0], now.UnixMilli()))
	want := colCost + 1 + colDuration + 1 + colRequests + 1 + colTools + 1 + colTokens + 1 + colAge
	if got := len([]rune(line)); got != want {
		t.Errorf("metric line width = %d, want %d: %q", got, want, line)
	}
}

// TestHubStatusGlyphsMatchOMP proves the status symbols and colors are omp's.
func TestHubStatusGlyphsMatchOMP(t *testing.T) {
	for status, want := range map[subagent.Status]string{
		subagent.StatusRunning: glyphRunning,
		subagent.StatusIdle:    glyphIdle,
		subagent.StatusParked:  glyphParked,
		subagent.StatusAborted: glyphAborted,
	} {
		if got := statusGlyph(status); got != want {
			t.Errorf("statusGlyph(%s) = %q, want %q", status, got, want)
		}
	}
}

// TestHubDetailPanelMatchesOMP proves the inspector carries omp's sections.
func TestHubDetailPanelMatchesOMP(t *testing.T) {
	registry, now := newOMPRegistry(t)
	model := New(Options{Registry: registry, Width: 140, Height: 40,
		Now: func() time.Time { return now }})
	model.SetWide(true)
	model.Select("Scout")
	text := stripAnsi(strings.Join(model.Render(140), "\n"))
	for _, want := range []string{"Task", "Current", "Usage", "Lineage", "Spawned by", "openai/gpt-5", "grep"} {
		if !strings.Contains(text, want) {
			t.Errorf("inspector missing %q:\n%s", want, text)
		}
	}
}

// TestHubFooterMatchesOMP proves the key hints are omp's.
func TestHubFooterMatchesOMP(t *testing.T) {
	plain := stripAnsi(renderOMP(t, 160, false))
	for _, want := range []string{"1:agents", "2:activity", "r:revive", "x:kill", "escape:close"} {
		if !strings.Contains(plain, want) {
			t.Errorf("footer missing %q:\n%s", want, plain)
		}
	}
}

// stripAnsi removes SGR sequences for plain-text assertions.
func stripAnsi(text string) string {
	var b strings.Builder
	for i := 0; i < len(text); i++ {
		if text[i] == 0x1b {
			for i < len(text) && text[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(text[i])
	}
	return b.String()
}

var _ = tui.ActiveTheme
