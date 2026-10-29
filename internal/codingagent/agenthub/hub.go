// Package agenthub renders the Agent Hub: the live roster of subagents a
// session spawned, with per-agent status, activity, usage and controls.
//
// Upstream: omp src/tui Agent Hub (docs/agent-hub.md). The component is
// headless-testable: it renders rows from registry snapshots and processes key
// input, so the interactive mode only wires it to a terminal and to the
// subagent runtime.
package agenthub

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tmih06/ompig/coding/subagent"
	"github.com/tmih06/ompig/tui"
)

// Section selects which pane the hub shows.
type Section int

const (
	// SectionAgents is the roster.
	SectionAgents Section = iota
	// SectionActivity is the combined activity log.
	SectionActivity
)

// Action is what the hub asks the caller to do after a key press.
type Action string

const (
	ActionNone    Action = ""
	ActionClose   Action = "close"
	ActionFocus   Action = "focus"
	ActionRevive  Action = "revive"
	ActionKill    Action = "kill"
	ActionMessage Action = "message"
)

// Request is the caller's instruction after a key press.
type Request struct {
	Action Action
	// AgentID is the row the action applies to.
	AgentID string
	// Message carries text for ActionMessage.
	Message string
}

// Options configures a hub instance.
type Options struct {
	// Registry is the live agent registry. Required.
	Registry *subagent.Registry
	// Jobs supplies the activity log. Optional.
	Jobs *subagent.Jobs
	// Width is the render width in cells; 0 uses 80.
	Width int
	// Height is the visible row budget; 0 uses 20.
	Height int
	// Now is the clock, for tests.
	Now func() time.Time
	// OnChange is called after every registry change, so the host can repaint
	// the open overlay. Render alone would leave a running agent's tool, tokens
	// and elapsed time stale until the next keypress.
	OnChange func()
}

// Model is the hub's presentation state.
type Model struct {
	opts     Options
	entries  []subagent.Entry
	sections Section
	selected int
	// filter narrows rows by id or agent type.
	filter string
	// tree shows parent/child nesting instead of the flat operational order.
	tree bool
	// wide shows the inspector beside the roster.
	wide bool
	// editing collects a steer message for the selected agent.
	editing bool
	draft   string
	// activity is the recent event log, newest first.
	activity []activityEntry
}

// activityEntry is one line of the activity pane.
type activityEntry struct {
	At      time.Time
	AgentID string
	Kind    subagent.EventKind
	Message string
}

// New builds a hub over a registry.
func New(opts Options) *Model {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Width <= 0 {
		opts.Width = 80
	}
	if opts.Height <= 0 {
		opts.Height = 20
	}
	model := &Model{opts: opts}
	if opts.Registry != nil {
		opts.Registry.Subscribe(model.observe)
	}
	model.reload()
	return model
}

// observe records a registry event for the activity pane and refreshes rows.
func (m *Model) observe(event subagent.Event) {
	if event.Kind == subagent.EventRemoved {
		m.reload()
		m.changed()
		return
	}
	message := ""
	switch event.Kind {
	case subagent.EventRegistered:
		message = "spawned " + event.Entry.AgentType
	case subagent.EventSettled:
		message = "settled " + string(event.Entry.Status)
	case subagent.EventMessaged:
		message = "message: " + event.Message
	case subagent.EventStatus:
		message = "status " + string(event.Entry.Status)
	}
	if message == "" {
		// A progress event carries no log line, but the roster columns moved.
		m.reload()
		m.changed()
		return
	}
	m.activity = append([]activityEntry{{
		At:      m.opts.Now(),
		AgentID: event.Entry.ID,
		Kind:    event.Kind,
		Message: message,
	}}, m.activity...)
	if len(m.activity) > 200 {
		m.activity = m.activity[:200]
	}
	m.reload()
	m.changed()
}

// changed tells the host to repaint.
func (m *Model) changed() {
	if m.opts.OnChange != nil {
		m.opts.OnChange()
	}
}

// reload refreshes the visible rows from the registry.
func (m *Model) reload() {
	if m.opts.Registry == nil {
		m.entries = nil
		return
	}
	entries := m.opts.Registry.Sorted()
	if m.filter != "" {
		needle := strings.ToLower(m.filter)
		kept := entries[:0]
		for _, entry := range entries {
			if strings.Contains(strings.ToLower(entry.ID), needle) ||
				strings.Contains(strings.ToLower(entry.AgentType), needle) {
				kept = append(kept, entry)
			}
		}
		entries = kept
	}
	if m.tree {
		entries = treeOrder(entries)
	}
	m.entries = entries
	if m.selected >= len(m.entries) {
		m.selected = max(0, len(m.entries)-1)
	}
}

// treeOrder returns rows in parent-before-child order, keeping the operational
// sort among roots.
func treeOrder(entries []subagent.Entry) []subagent.Entry {
	byID := make(map[string]subagent.Entry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	var out []subagent.Entry
	var visit func(parent string, depth int)
	seen := map[string]bool{}
	visit = func(parent string, depth int) {
		for _, entry := range entries {
			if entry.ParentID != parent || seen[entry.ID] {
				continue
			}
			seen[entry.ID] = true
			entry.Depth = depth + 1
			out = append(out, entry)
			visit(entry.ID, depth+1)
		}
	}
	visit("", 0)
	// Rows whose parent is filtered out or missing still need to appear.
	for _, entry := range entries {
		if !seen[entry.ID] {
			out = append(out, entry)
		}
	}
	return out
}

// Selected returns the highlighted agent, if any.
func (m *Model) Selected() (subagent.Entry, bool) {
	if m.selected < 0 || m.selected >= len(m.entries) {
		return subagent.Entry{}, false
	}
	return m.entries[m.selected], true
}

// HandleInput processes one key and returns the request it produced.
func (m *Model) HandleInput(key string) Request {
	if m.editing {
		return m.handleEditing(key)
	}
	switch key {
	case "esc":
		if m.filter != "" {
			m.filter = ""
			m.reload()
			return Request{}
		}
		return Request{Action: ActionClose}
	case "ctrl+c":
		return Request{Action: ActionClose}
	case "j", "down":
		if m.selected < len(m.entries)-1 {
			m.selected++
		}
	case "k", "up":
		if m.selected > 0 {
			m.selected--
		}
	case "t":
		m.tree = !m.tree
		m.reload()
	case "1":
		m.sections = SectionAgents
	case "2":
		m.sections = SectionActivity
	case "r":
		if entry, ok := m.Selected(); ok {
			return Request{Action: ActionRevive, AgentID: entry.ID}
		}
	case "x":
		if entry, ok := m.Selected(); ok {
			return Request{Action: ActionKill, AgentID: entry.ID}
		}
	case "enter":
		if entry, ok := m.Selected(); ok {
			return Request{Action: ActionFocus, AgentID: entry.ID}
		}
	case "m":
		if entry, ok := m.Selected(); ok && !entry.Status.Terminal() {
			m.editing = true
			m.draft = ""
			return Request{Action: ActionMessage, AgentID: entry.ID}
		}
	case "/":
		m.filter = ""
	}
	return Request{}
}

// handleEditing accumulates a steer message. Enter submits, Esc cancels.
func (m *Model) handleEditing(key string) Request {
	switch key {
	case "esc":
		m.editing = false
		m.draft = ""
		return Request{}
	case "enter":
		entry, ok := m.Selected()
		message := m.draft
		m.editing = false
		m.draft = ""
		if !ok || strings.TrimSpace(message) == "" {
			return Request{}
		}
		return Request{Action: ActionMessage, AgentID: entry.ID, Message: message}
	case "backspace":
		if m.draft != "" {
			runes := []rune(m.draft)
			m.draft = string(runes[:len(runes)-1])
		}
	default:
		if len(key) == 1 && key >= " " {
			m.draft += key
		}
	}
	return Request{}
}

// Editing reports whether the hub is collecting a message.
func (m *Model) Editing() bool { return m.editing }

// Draft is the in-progress message.
func (m *Model) Draft() string { return m.draft }

// Render draws the hub.
func (m *Model) Render(width int) []string {
	if width <= 0 {
		width = m.opts.Width
	}
	m.reload()
	lines := make([]string, 0, m.opts.Height+2)
	lines = append(lines, m.header(width))
	if m.sections == SectionActivity {
		lines = append(lines, m.renderActivity(width)...)
	} else if m.wide {
		lines = append(lines, m.renderWide(width)...)
	} else {
		lines = append(lines, m.renderRoster(width)...)
	}
	lines = append(lines, m.footer(width))
	return lines
}

// header is the aggregate status line.
func (m *Model) header(width int) string {
	running, total := 0, len(m.entries)
	var usage subagent.Usage
	for _, entry := range m.entries {
		if entry.Running() {
			running++
		}
		usage.Add(entry.Usage)
	}
	mode := "agents"
	if m.sections == SectionActivity {
		mode = "activity"
	}
	text := fmt.Sprintf(" Agent Hub · %s · %d running · %d total · %s tokens · $%.2f",
		mode, running, total, formatTokens(usage.Total), usage.Cost)
	if m.filter != "" {
		text += " · filter: " + m.filter
	}
	if m.tree {
		text += " · tree"
	}
	return pad(text, width)
}

// footer is the key hint line.
func (m *Model) footer(width int) string {
	if m.editing {
		return pad(" message "+fmt.Sprintf("%q", m.draft)+" · enter send · esc cancel", width)
	}
	return pad(" j/k select · enter focus · m message · r revive · x kill · t tree · 1/2 pane · / filter · esc close", width)
}

// renderRoster draws the flat roster.
func (m *Model) renderRoster(width int) []string {
	if len(m.entries) == 0 {
		return []string{pad(" No subagents yet. Spawn one with the task tool.", width)}
	}
	now := m.opts.Now()
	lines := make([]string, 0, len(m.entries))
	for index, entry := range m.entries {
		lines = append(lines, m.rosterRow(index, entry, now, width))
	}
	return lines
}

// rosterRow formats one roster line.
func (m *Model) rosterRow(index int, entry subagent.Entry, now time.Time, width int) string {
	marker := " "
	if index == m.selected {
		marker = "❯"
	}
	indent := ""
	if m.tree && entry.Depth > 1 {
		indent = strings.Repeat("  ", entry.Depth-1)
	}
	activity := entry.CurrentTool
	if activity == "" {
		if entry.Status == subagent.StatusIdle {
			activity = "(done)"
		} else if entry.Task != "" {
			activity = firstLine(entry.Task)
		}
	}
	row := fmt.Sprintf("%s %-11s %s%s", marker, statusLabel(entry), indent+entry.ID, "")
	if entry.AgentType != "" && entry.AgentType != entry.ID {
		row += " [" + entry.AgentType + "]"
	}
	row += " · " + activity
	if tokens := entry.Usage.Total; tokens > 0 {
		row += " · " + formatTokens(tokens)
	}
	if elapsed := entry.Elapsed(now); elapsed > 0 {
		row += " · " + formatDuration(elapsed)
	}
	if entry.Unread > 0 {
		row += fmt.Sprintf(" · %d unread", entry.Unread)
	}
	return pad(row, width)
}

// renderWide draws the roster beside the inspector.
func (m *Model) renderWide(width int) []string {
	leftWidth := width / 2
	rightWidth := width - leftWidth - 1
	roster := m.renderRoster(leftWidth)
	inspect := m.renderInspector(rightWidth)
	rows := max(len(roster), len(inspect))
	out := make([]string, 0, rows)
	for i := range rows {
		left := ""
		if i < len(roster) {
			left = roster[i]
		}
		right := ""
		if i < len(inspect) {
			right = inspect[i]
		}
		out = append(out, pad(left, leftWidth)+"│"+pad(right, rightWidth))
	}
	return out
}

// renderInspector is the selected agent's detail pane.
func (m *Model) renderInspector(width int) []string {
	entry, ok := m.Selected()
	if !ok {
		return []string{pad(" no agent selected", width)}
	}
	now := m.opts.Now()
	lines := []string{
		pad(" "+entry.ID, width),
		pad(" status    "+string(entry.Status), width),
		pad(" agent     "+orDash(entry.AgentType), width),
		pad(" model     "+orDash(entry.Model), width),
		pad(" thinking  "+orDash(entry.ThinkingLevel), width),
		pad(" elapsed   "+formatDuration(entry.Elapsed(now)), width),
		pad(" tokens    "+fmt.Sprintf("%s (in %s, out %s)", formatTokens(entry.Usage.Total), formatTokens(entry.Usage.Input), formatTokens(entry.Usage.Output)), width),
		pad(" requests  "+fmt.Sprintf("%d", entry.Usage.Requests), width),
		pad(" tools     "+fmt.Sprintf("%d", entry.Usage.ToolCalls), width),
		pad(" cost      "+fmt.Sprintf("$%.4f", entry.Usage.Cost), width),
	}
	if entry.ContextWindow > 0 {
		lines = append(lines, pad(" context   "+fmt.Sprintf("%d/%d", entry.ContextTokens, entry.ContextWindow), width))
	}
	lines = append(lines,
		pad(" tool      "+orDash(entry.CurrentTool), width),
		pad(" intent    "+orDash(entry.CurrentIntent), width),
	)
	if entry.ParentID != "" {
		lines = append(lines, pad(" parent    "+entry.ParentID, width))
	}
	if len(entry.Children) > 0 {
		lines = append(lines, pad(" children  "+strings.Join(entry.Children, ", "), width))
	}
	if entry.Error != "" {
		lines = append(lines, pad(" error     "+entry.Error, width))
	}
	if entry.OutputPath != "" {
		lines = append(lines, pad(" output    "+entry.OutputPath, width))
	}
	return lines
}

// renderActivity draws the combined log.
func (m *Model) renderActivity(width int) []string {
	if len(m.activity) == 0 {
		return []string{pad(" no activity yet", width)}
	}
	limit := min(len(m.activity), m.opts.Height)
	lines := make([]string, 0, limit)
	for _, event := range m.activity[:limit] {
		// The activity pane renders at most the visible rows, once per repaint.
		line := fmt.Sprintf(" %s %-20s %s", event.At.Format("15:04:05"), event.AgentID, event.Message) //perf:ok a display path, not the hot path
		lines = append(lines, pad(line, width))
	}
	return lines
}

// statusLabel is the fixed-width status column.
func statusLabel(entry subagent.Entry) string {
	switch entry.Status {
	case subagent.StatusRunning:
		return "running"
	case subagent.StatusIdle:
		return "idle"
	case subagent.StatusParked:
		return "parked"
	case subagent.StatusAborted:
		return "aborted"
	default:
		return string(entry.Status)
	}
}

// formatTokens renders a token count compactly.
func formatTokens(count int64) string {
	switch {
	case count >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(count)/1_000_000)
	case count >= 1_000:
		return fmt.Sprintf("%.1fk", float64(count)/1_000)
	default:
		return fmt.Sprintf("%d", count)
	}
}

// formatDuration renders an elapsed span.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}

// firstLine ends at the first line break and truncates.
func firstLine(text string) string {
	if at := strings.IndexByte(text, '\n'); at >= 0 {
		text = text[:at]
	}
	text = strings.TrimSpace(text)
	if len(text) > 60 {
		text = text[:60] + "…"
	}
	return text
}

// orDash renders an empty field as a dash.
func orDash(value string) string {
	if value == "" {
		return "—"
	}
	return value
}

// pad truncates or pads a line to exactly width cells, ignoring styling.
func pad(line string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(line)
	if len(runes) > width {
		if width <= 1 {
			return string(runes[:width])
		}
		return string(runes[:width-1]) + "…"
	}
	return line + strings.Repeat(" ", width-len(runes))
}

// Entries returns the current visible rows, for tests and callers that render
// their own view.
func (m *Model) Entries() []subagent.Entry { return m.entries }

// Select moves the highlight to an id, reporting whether it exists.
func (m *Model) Select(id string) bool {
	for index, entry := range m.entries {
		if entry.ID == id {
			m.selected = index
			return true
		}
	}
	return false
}

// SetWide toggles the inspector pane.
func (m *Model) SetWide(wide bool) { m.wide = wide }

// SetSection switches panes.
func (m *Model) SetSection(section Section) { m.sections = section }

// sortEntriesByStart is a helper kept for callers formatting their own lists.
func sortEntriesByStart(entries []subagent.Entry) []subagent.Entry {
	out := append([]subagent.Entry(nil), entries...)
	sort.SliceStable(out, func(a, b int) bool { return out[a].StartedAt.Before(out[b].StartedAt) })
	return out
}

var _ tui.Component = (*Model)(nil)

// Invalidate marks the hub dirty. The roster is rebuilt on each Render, so this
// only needs to satisfy the Component contract.
func (m *Model) Invalidate() {}
