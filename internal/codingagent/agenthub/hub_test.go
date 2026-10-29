package agenthub

import (
	"strings"
	"testing"
	"time"

	"github.com/tmih06/ompig/coding/subagent"
)

// newFixture builds a registry with a running, an idle and an aborted agent.
func newFixture(t *testing.T) (*subagent.Registry, *subagent.Jobs) {
	t.Helper()
	registry := subagent.NewRegistry()
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	registry.SetClock(func() time.Time { return now })

	running := registry.Register(subagent.Entry{Name: "Scout", AgentType: "scout", Task: "map the repo"})
	registry.Update(running.ID, func(e *subagent.Entry) {
		e.CurrentTool = "grep"
		e.CurrentIntent = "Searching callers"
		e.Model = "openai/gpt-5"
		e.Usage = subagent.Usage{Input: 1200, Output: 300, Total: 1500, Cost: 0.02, Requests: 3, ToolCalls: 7}
	})
	idle := registry.Register(subagent.Entry{Name: "Writer", AgentType: "task", Task: "write the doc"})
	registry.SetStatus(idle.ID, subagent.StatusIdle)
	registry.Update(idle.ID, func(e *subagent.Entry) {
		e.Usage = subagent.Usage{Total: 4200, Cost: 0.11}
	})
	killed := registry.Register(subagent.Entry{Name: "Broken", AgentType: "task", Task: "fail"})
	registry.Update(killed.ID, func(e *subagent.Entry) { e.Error = "provider 500" })
	registry.SetStatus(killed.ID, subagent.StatusAborted)
	return registry, subagent.NewJobs(0)
}

func newTestModel(t *testing.T, registry *subagent.Registry, jobs *subagent.Jobs) *Model {
	t.Helper()
	model := New(Options{Registry: registry, Jobs: jobs, Width: 100, Height: 30,
		Now: func() time.Time { return time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC) }})
	model.SetWide(false)
	return model
}

// TestHubRendersRosterInOperationalOrder covers omp's roster: the header with
// the status counts, and one entry per agent in status order, each with its
// identity, task and metric lines.
func TestHubRendersRosterInOperationalOrder(t *testing.T) {
	registry, jobs := newFixture(t)
	model := newTestModel(t, registry, jobs)
	text := stripAnsi(strings.Join(model.Render(120), "\n"))

	// The roster carries identity, task and metrics; the current tool is the
	// inspector's, not the roster's.
	for _, want := range []string{"Roster", "Flat", "running", "idle", "aborted", "Scout", "Writer", "Broken", "$0.020"} {
		if !strings.Contains(text, want) {
			t.Errorf("rendered roster missing %q:\n%s", want, text)
		}
	}
	scout, writer, broken := strings.Index(text, "Scout"), strings.Index(text, "Writer"), strings.Index(text, "Broken")
	if scout >= writer || writer >= broken {
		t.Errorf("roster order Scout(%d) Writer(%d) Broken(%d) is not running→idle→aborted", scout, writer, broken)
	}
}

// TestHubInspectorShowsUsageAndTool covers omp's inspector sections: status and
// lifecycle, the model badge, then Task, Current, Usage and Lineage.
func TestHubInspectorShowsUsageAndTool(t *testing.T) {
	registry, jobs := newFixture(t)
	model := newTestModel(t, registry, jobs)
	model.SetWide(true)
	model.Select("Scout")
	text := stripAnsi(strings.Join(model.Render(140), "\n"))
	for _, want := range []string{"Scout", "running", "openai/gpt-5", "Task", "Current", "Usage", "Lineage", "Spawned by", "grep", "Searching callers", "req", "tools", "tok"} {
		if !strings.Contains(text, want) {
			t.Errorf("inspector missing %q:\n%s", want, text)
		}
	}
}

// TestHubActionsCoverEveryControl checks the key map the docs promise.
func TestHubActionsCoverEveryControl(t *testing.T) {
	registry, jobs := newFixture(t)
	model := newTestModel(t, registry, jobs)

	if entry, ok := model.Selected(); !ok || entry.ID != "Scout" {
		t.Fatalf("initial selection is %v/%v, want Scout (the running row)", entry.ID, ok)
	}
	if got := model.HandleInput("j"); got.Action != ActionNone {
		t.Errorf("j produced %q, want navigation", got.Action)
	}
	if entry, ok := model.Selected(); !ok || entry.ID != "Writer" {
		t.Errorf("after j the selection is %v/%v, want Writer", entry.ID, ok)
	}
	model.HandleInput("k")
	if entry, ok := model.Selected(); !ok || entry.ID != "Scout" {
		t.Errorf("after k the selection is %v/%v, want Scout", entry.ID, ok)
	}
	if got := model.HandleInput("enter"); got.Action != ActionFocus || got.AgentID != "Scout" {
		t.Errorf("enter = %+v, want focus Scout", got)
	}
	if got := model.HandleInput("r"); got.Action != ActionRevive || got.AgentID != "Scout" {
		t.Errorf("r = %+v, want revive Scout", got)
	}
	if got := model.HandleInput("x"); got.Action != ActionKill || got.AgentID != "Scout" {
		t.Errorf("x = %+v, want kill Scout", got)
	}

	// Message: first press opens the editor, typed keys collect, enter sends.
	if got := model.HandleInput("m"); got.Action != ActionMessage || got.Message != "" {
		t.Errorf("m = %+v, want an open editor", got)
	}
	if !model.Editing() {
		t.Fatal("m did not open the message editor")
	}
	for _, key := range []string{"h", "i"} {
		model.HandleInput(key)
	}
	if model.Draft() != "hi" {
		t.Errorf("draft = %q, want hi", model.Draft())
	}
	if got := model.HandleInput("enter"); got.Action != ActionMessage || got.Message != "hi" || got.AgentID != "Scout" {
		t.Errorf("enter in editor = %+v, want a message to Scout", got)
	}

	if got := model.HandleInput("2"); got.Action != ActionNone {
		t.Errorf("2 = %+v, want a pane switch", got)
	}
	if got := model.HandleInput("esc"); got.Action != ActionClose {
		t.Errorf("esc = %+v, want close", got)
	}
}

// TestHubTreeGroupsChildren checks that the tree view nests a child under its
// parent.
func TestHubTreeGroupsChildren(t *testing.T) {
	registry := subagent.NewRegistry()
	parent := registry.Register(subagent.Entry{Name: "Lead"})
	child := registry.Register(subagent.Entry{Name: "Helper", ParentID: parent.ID})
	model := New(Options{Registry: registry, Width: 80, Height: 10})
	model.HandleInput("t")
	model.reload()
	text := strings.Join(model.Render(80), "\n")
	lead, helper := indexOf(text, "Lead"), indexOf(text, "Helper")
	if lead < 0 || helper < 0 || lead > helper {
		t.Errorf("tree order Lead(%d) Helper(%d):\n%s", lead, helper, text)
	}
	if got := child.ID; got != "Lead.Helper" {
		t.Errorf("child id = %q, want Lead.Helper", got)
	}
}

// TestHubActivityPaneRecordsEvents covers the combined log.
func TestHubActivityPaneRecordsEvents(t *testing.T) {
	registry, jobs := newFixture(t)
	model := newTestModel(t, registry, jobs)
	registry.Register(subagent.Entry{Name: "Late"})
	model.HandleInput("2")
	text := strings.Join(model.Render(100), "\n")
	if !strings.Contains(text, "spawned") {
		t.Errorf("activity log missing a spawn:\n%s", text)
	}
}

// TestHubFilterNarrowsRows covers the id/type filter.
func TestHubFilterNarrowsRows(t *testing.T) {
	registry, jobs := newFixture(t)
	model := newTestModel(t, registry, jobs)
	model.filter = "writ"
	model.reload()
	if len(model.Entries()) != 1 || model.Entries()[0].ID != "Writer" {
		t.Errorf("filtered rows = %+v, want just Writer", model.Entries())
	}
}

// TestTranscriptRendersMessages covers the Enter view.
func TestTranscriptRendersMessages(t *testing.T) {
	view := NewTranscript("Scout", nil, 60, 10)
	text := strings.Join(view.Render(60), "\n")
	if !strings.Contains(text, "no messages") {
		t.Errorf("empty transcript:\n%s", text)
	}
	view.HandleInput("j")
	view.HandleInput("G")
	if scope := view.Render(60); len(scope) == 0 {
		t.Error("transcript rendered nothing")
	}
}

func indexOf(haystack, needle string) int {
	return strings.Index(haystack, needle)
}

// TestHubRepaintsOnRegistryChange proves live updates: a registry change calls
// the host's repaint hook, so a running agent's columns are not stale until the
// next keypress.
func TestHubRepaintsOnRegistryChange(t *testing.T) {
	registry := subagent.NewRegistry()
	paints := 0
	model := New(Options{Registry: registry, Width: 80, Height: 8,
		OnChange: func() { paints++ }})
	entry := registry.Register(subagent.Entry{Name: "Live", AgentType: "task"})
	if paints == 0 {
		t.Fatal("a spawn did not repaint the hub")
	}
	before := paints
	registry.Update(entry.ID, func(e *subagent.Entry) {
		e.CurrentTool = "grep"
		e.Usage = subagent.Usage{Total: 900, Requests: 2}
	})
	if paints <= before {
		t.Error("a progress update did not repaint the hub")
	}
	text := stripAnsi(strings.Join(model.Render(120), "\n"))
	// The current tool shows in the inspector, and the metrics move in the roster.
	if !strings.Contains(text, "900") || !strings.Contains(text, "2 req") {
		t.Errorf("the repainted roster is stale:\n%s", text)
	}
	model.SetWide(true)
	model.Select("Live")
	wide := stripAnsi(strings.Join(model.Render(140), "\n"))
	if !strings.Contains(wide, "grep") {
		t.Errorf("the repainted inspector is stale:\n%s", wide)
	}
}
