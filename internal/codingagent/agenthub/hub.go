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
	"strings"
	"time"

	"github.com/tmih06/ompig/coding/subagent"
	"github.com/tmih06/ompig/tui"
	"github.com/tmih06/ompig/tui/widthx"
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

// theme is the active terminal theme.
func (m *Model) theme() *tui.Theme { return tui.ActiveTheme() }

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

// Render draws the hub in omp's layout: a roster header with the counts, one
// entry per agent (identity, task, metric columns), and the inspector beside it
// on a wide terminal.
func (m *Model) Render(width int) []string {
	if width <= 0 {
		width = m.opts.Width
	}
	m.reload()
	if m.sections == SectionActivity {
		return m.renderActivitySection(width)
	}
	if m.wide && width >= SPLIT_MIN_WIDTH {
		return m.renderSplit(width)
	}
	return m.renderRosterSection(width)
}

// renderRosterSection is the single-pane roster: header, entries, footer.
func (m *Model) renderRosterSection(width int) []string {
	lines := append(m.summaryHeader(width), "")
	budget := max(m.opts.Height-len(lines)-1, 1)
	lines = append(lines, m.renderEntries(width, budget)...)
	lines = append(lines, m.footerLine(width))
	return lines
}

// renderSplit is the wide layout: roster on the left, inspector on the right.
func (m *Model) renderSplit(width int) []string {
	// The roster needs the metric row plus its indent, so the split keeps that
	// much for it and gives the rest to the inspector.
	leftWidth := min(max(detailIndent+metricRowWidth, (width-1)*2/5), width-DETAIL_MIN_WIDTH-1)
	rightWidth := width - leftWidth - 1
	left := m.renderRosterSection(leftWidth)
	right := m.renderDetailPanel(rightWidth, m.opts.Height-1)
	rows := max(len(left), len(right))
	out := make([]string, 0, rows+1)
	for i := range rows {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		out = append(out, pad(l, leftWidth)+"│"+pad(r, rightWidth))
	}
	return out
}

// summaryHeader is omp's summary line: "Roster · Flat/By parent · counts",
// then the aggregate usage line.
func (m *Model) summaryHeader(width int) []string {
	theme := m.theme()
	mode := m.projectionLabel()
	header := theme.Bold("Roster") + theme.Fg("dim", glyphDot) + " " + mode
	if counts := statusSummary(theme, m.entries); counts != "" {
		header += theme.Fg("dim", " "+glyphDot+" ") + counts
	}
	return []string{
		truncate(header, width),
		theme.Fg("dim", truncate(m.aggregateUsage(), width)),
	}
}

// projectionLabel is omp's "Flat / By parent" toggle.
func (m *Model) projectionLabel() string {
	active := func(label string) string { return m.theme().Bold(m.theme().Fg("accent", " "+label+" ")) }
	inactive := func(label string) string { return m.theme().Fg("muted", " "+label+" ") }
	sep := m.theme().Fg("dim", "/")
	if m.tree {
		return inactive("Flat") + sep + active("By parent")
	}
	return active("Flat") + sep + inactive("By parent")
}

// aggregateUsage is omp's aggregate metric line.
func (m *Model) aggregateUsage() string {
	var cost float64
	var tokens, requests, tools int64
	measured := 0
	for _, entry := range m.entries {
		if entry.Usage.Total == 0 && entry.Usage.Requests == 0 {
			continue
		}
		measured++
		cost += entry.Usage.Cost
		tokens += entry.Usage.Total
		requests += entry.Usage.Requests
		tools += entry.Usage.ToolCalls
	}
	if measured == 0 {
		return fmt.Sprintf("Usage —%s0/%d measured", glyphDot, len(m.entries))
	}
	return strings.Join([]string{
		formatCost(cost),
		formatNumber(tokens) + " tok",
		formatNumber(requests) + " req",
		formatNumber(tools) + " tools",
		fmt.Sprintf("%d/%d measured", measured, len(m.entries)),
	}, " "+glyphDot+" ")
}

// renderEntries draws every visible entry with the selection window around it.
func (m *Model) renderEntries(width, budget int) []string {
	if budget < 1 {
		budget = 1
	}
	if len(m.entries) == 0 {
		return []string{
			themeFgDim(m.theme(), glyphParked+" No agents in this session"),
			themeFgDim(m.theme(), "Finished, parked, and killed subagents remain with the session that created them."),
			themeFgDim(m.theme(), "Resume that session, or spawn a task here."),
		}
	}
	// Keep the selection visible: grow a window outward from it.
	start, end := m.selected, m.selected+1
	used := len(m.entryLines(m.entries[m.selected], true, width))
	for used < budget {
		grew := false
		if end < len(m.entries) {
			if n := len(m.entryLines(m.entries[end], false, width)); used+n <= budget {
				used += n
				end++
				grew = true
			}
		}
		if start > 0 {
			if n := len(m.entryLines(m.entries[start-1], false, width)); used+n <= budget {
				used += n
				start--
				grew = true
			}
		}
		if !grew {
			break
		}
	}
	var lines []string
	if start > 0 {
		lines = append(lines, themeFgDim(m.theme(), fmt.Sprintf("… %d more", start)))
	}
	for i := start; i < end; i++ {
		lines = append(lines, m.entryLines(m.entries[i], i == m.selected, width)...)
	}
	if end < len(m.entries) {
		lines = append(lines, themeFgDim(m.theme(), fmt.Sprintf("… %d more", len(m.entries)-end)))
	}
	return lines
}

// entryLines is omp's #renderEntry: identity plus badges on one line, then the
// task, then the metric columns.
func (m *Model) entryLines(entry subagent.Entry, selected bool, width int) []string {
	theme := m.theme()
	cursor := " "
	if selected {
		cursor = theme.Fg("accent", glyphCursor)
	}
	id := entry.ID
	if selected {
		id = theme.Bold(theme.Fg("accent", id))
	} else {
		id = theme.Bold(id)
	}
	branch := ""
	if m.tree && entry.Depth > 1 {
		prefix := strings.Repeat("│   ", entry.Depth-2)
		connector := "├── "
		if selected {
			connector = "└── "
		}
		branch = theme.Fg("dim", prefix+connector)
	}
	fields := []string{cursor + " " + branch + statusGlyphStyled(theme, entry.Status) + " " + id}
	if !m.tree && entry.ParentID != "" {
		fields = append(fields, theme.Fg("dim", glyphParent+" "+entry.ParentID))
	}
	if entry.Kind == subagent.KindAdvisor {
		fields = append(fields, theme.Fg("warning", "read-only"))
	}
	if entry.Unread > 0 {
		fields = append(fields, theme.Fg("warning", fmt.Sprintf("%s %d", glyphUnread, entry.Unread)))
	}
	left := strings.Join(fields, "  ")

	// The right side carries only variable-width badges, so the metric line
	// below can never be split by task text.
	var meta []string
	if badge := modelBadge(theme, entry); badge != "" {
		meta = append(meta, badge)
	}
	right := strings.Join(meta, themeSep(theme))

	lines := make([]string, 0, 3)
	leftWidth, rightWidth := widthx.VisibleWidth(left), widthx.VisibleWidth(right)
	if rightWidth > 0 && leftWidth+2+rightWidth <= width {
		lines = append(lines, left+strings.Repeat(" ", width-leftWidth-rightWidth)+right)
	} else {
		lines = append(lines, truncate(left, width))
	}
	indent := strings.Repeat(" ", detailIndent+widthx.VisibleWidth(branch))
	if task := taskSummary(entry); task != "" {
		lines = append(lines, indent+theme.Fg("muted", truncate(task, width-detailIndent)))
	}
	now := m.opts.Now().UnixMilli()
	lines = append(lines, indent+theme.Fg("dim", metricColumns(entry, now)))
	return lines
}

// taskSummary is omp's taskSummary: the assignment preamble is not the task.
func taskSummary(entry subagent.Entry) string {
	task := entry.Task
	for _, prefix := range []string{"Complete assignment thoroughly:", "## Task"} {
		if at := strings.Index(task, prefix); at == 0 {
			task = strings.TrimSpace(task[len(prefix):])
		}
	}
	if at := strings.IndexByte(task, '\n'); at >= 0 {
		task = task[:at]
	}
	return strings.TrimSpace(task)
}

// renderDetailPanel is omp's #renderDetailPanel: status, lifecycle, model, then
// the Task, Current, Usage and Lineage sections.
func (m *Model) renderDetailPanel(width, rows int) []string {
	theme := m.theme()
	entry, ok := m.Selected()
	if !ok {
		return []string{theme.Fg("dim", "Select an agent to inspect")}
	}
	now := m.opts.Now().UnixMilli()
	var lines []string
	add := func(line string) { lines = append(lines, truncate(line, width)) }
	section := func(label string) {
		if len(lines) > 0 {
			add("")
		}
		add(theme.Bold(theme.Fg("accent", label)))
	}

	add(statusGlyphStyled(theme, entry.Status) + " " + theme.Bold(entry.ID))
	add(statusText(theme, entry.Status, string(entry.Status)) + theme.Fg("dim",
		themeSep(theme)+formatMetricDuration(entry, now)+themeSep(theme)+"active "+formatAge((now-entry.LastActivityAt.UnixMilli())/1000)))
	if badge := modelBadge(theme, entry); badge != "" {
		add(badge)
	}
	if task := taskSummary(entry); task != "" {
		section("Task")
		add(theme.Fg("muted", task))
	}
	if current := entry.CurrentTool; current != "" {
		section("Current")
		detail := current
		if entry.CurrentIntent != "" {
			detail += " " + glyphDot + " " + entry.CurrentIntent
		}
		add(detail)
	}
	section("Usage")
	if entry.Usage.Total > 0 || entry.Usage.Requests > 0 {
		add(metricsText(theme, entry, now))
		if entry.ContextWindow > 0 {
			add(contextGauge(theme, entry.ContextTokens, entry.ContextWindow))
		}
	} else {
		add(theme.Fg("dim", "usage —"))
	}
	section("Lineage")
	parent := entry.ParentID
	if parent == "" {
		parent = "Main"
	}
	lineage := "Spawned by " + parent
	if len(entry.Children) > 0 {
		lineage += fmt.Sprintf(" %s %d children", glyphDot, len(entry.Children))
	}
	add(lineage)
	if len(entry.Children) > 0 {
		add(theme.Fg("dim", strings.Join(entry.Children, ", ")))
	}
	section("Changes")
	if entry.Kind == subagent.KindAdvisor {
		add(theme.Fg("dim", "Read-only "+glyphDot+" 0 LoC"))
	} else {
		add(theme.Fg("dim", "Shared workspace "+glyphDot+" per-agent LoC not attributable"))
	}
	if entry.OutputPath != "" {
		add(theme.Fg("dim", "Output "+entry.OutputPath))
	}
	if entry.Error != "" {
		section("Error")
		add(theme.Fg("error", entry.Error))
	}
	for len(lines) < rows {
		lines = append(lines, "")
	}
	if len(lines) > rows {
		lines = lines[:rows]
	}
	return lines
}

// renderActivitySection draws the combined log with omp's footer hints.
func (m *Model) renderActivitySection(width int) []string {
	theme := m.theme()
	lines := []string{theme.Bold("Activity") + themeSep(theme) + statusSummary(theme, m.entries), ""}
	if len(m.activity) == 0 {
		lines = append(lines, theme.Fg("muted", "No response or tool activity yet"))
	}
	limit := min(len(m.activity), m.opts.Height-len(lines)-1)
	for _, event := range m.activity[:limit] {
		// The activity pane renders at most the visible rows, once per repaint.
		line := fmt.Sprintf("%s %s %s", theme.Fg("dim", event.At.Format("15:04:05")), event.AgentID, event.Message) //perf:ok a display path, not the hot path
		lines = append(lines, truncate(line, width))
	}
	lines = append(lines, m.footerLine(width))
	return lines
}

// footerLine is omp's footer hint line for the current pane.
func (m *Model) footerLine(width int) string {
	if m.editing {
		return pad(m.theme().Fg("dim", "message "+fmt.Sprintf("%q", m.draft)+"  enter:send  esc:cancel"), width)
	}
	if m.sections == SectionActivity {
		return pad(m.theme().Fg("dim", "1:agents  2:activity  tab:roster  pageUp/pageDown:scroll  enter:open  t:tree  escape:roster"), width)
	}
	if m.wide {
		return pad(m.theme().Fg("dim", "1:agents  2:activity  tab:roster  pageUp/pageDown:scroll  enter:open  t:tree  escape:close"), width)
	}
	return pad(m.theme().Fg("dim", "1:agents  2:activity  j/k/wheel:select  pageUp/pageDown:details  enter/click:open  t:tree  r:revive  x:kill  escape:close"), width)
}

// themeFgDim is a dim string when the theme has no styling hook.
func themeFgDim(theme *tui.Theme, text string) string {
	return theme.Fg("dim", text)
}

// pad truncates or pads a line to exactly width cells, ignoring styling.
func pad(line string, width int) string {
	if width <= 0 {
		return ""
	}
	if visible := widthx.VisibleWidth(line); visible > width {
		return truncate(line, width)
	} else if visible < width {
		return line + strings.Repeat(" ", width-visible)
	}
	return line
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

var _ tui.Component = (*Model)(nil)

// Invalidate marks the hub dirty. The roster is rebuilt on each Render, so this
// only needs to satisfy the Component contract.
func (m *Model) Invalidate() {}
