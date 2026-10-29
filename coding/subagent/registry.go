// Package subagent holds the runtime state of subagents a session spawns:
// their registry, output artifacts, background jobs, and the delivery of
// finished results back to the parent turn.
//
// The package is a leaf: it depends only on the standard library so the TUI,
// the coding session and the RPC surface can all observe the same agents
// without an import cycle.
package subagent

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Status is a registered agent's lifecycle state.
//
// Upstream: omp src/registry/agent-registry.ts AgentStatus
// (running | idle | parked | aborted).
type Status string

const (
	// StatusRunning means a turn or tool call is in flight.
	StatusRunning Status = "running"
	// StatusIdle means the agent finished its turn and its session is live, so
	// a follow-up message resumes it without rebuilding anything.
	StatusIdle Status = "idle"
	// StatusParked means the idle TTL expired: the session was disposed and
	// only its transcript and artifact paths remain. Messaging it revives it.
	StatusParked Status = "parked"
	// StatusAborted means the agent was killed. Its transcript survives, so a
	// later message revives it as a fresh generation.
	StatusAborted Status = "aborted"
)

// Terminal reports whether the status allows no further work without a revive.
func (s Status) Terminal() bool { return s == StatusParked || s == StatusAborted }

// Kind distinguishes what a row represents in the hub.
type Kind string

const (
	KindTask Kind = "task"
	// KindAdvisor rows are observability records: they cannot be messaged,
	// revived or killed.
	KindAdvisor Kind = "advisor"
)

// Usage is the token and cost accounting shown for one agent.
type Usage struct {
	Input      int64   `json:"input"`
	Output     int64   `json:"output"`
	CacheRead  int64   `json:"cacheRead"`
	CacheWrite int64   `json:"cacheWrite"`
	Total      int64   `json:"total"`
	Cost       float64 `json:"cost"`
	Requests   int64   `json:"requests"`
	ToolCalls  int64   `json:"toolCalls"`
}

// Add accumulates a turn's usage.
func (u *Usage) Add(other Usage) {
	u.Input += other.Input
	u.Output += other.Output
	u.CacheRead += other.CacheRead
	u.CacheWrite += other.CacheWrite
	u.Total += other.Total
	u.Cost += other.Cost
	u.Requests += other.Requests
	u.ToolCalls += other.ToolCalls
}

// Entry is one registered agent.
type Entry struct {
	ID   string
	Name string
	Kind Kind

	// ParentID is the spawning agent's id, empty for agents a user session
	// spawned directly. Nested children are dot-qualified: "Parent.Child".
	ParentID string
	// Depth is the recursion depth: 1 for a direct child of the session.
	Depth int

	// Task is the self-contained assignment; Description is a one-line label.
	Task        string
	Description string
	// AgentType is the agent definition the spawn selected ("scout", "task"...).
	AgentType string

	Model         string
	ThinkingLevel string

	Status Status
	// StartedAt is when the spawn was accepted; EndedAt when the last run
	// settled (zero while running).
	StartedAt time.Time
	EndedAt   time.Time
	// LastActivityAt updates on every event, so the hub can show idleness.
	LastActivityAt time.Time

	// CurrentTool is the tool call in flight, empty between calls. CurrentIntent
	// is the call's declared intent when the tool supplied one.
	CurrentTool   string
	CurrentIntent string
	CurrentArgs   string

	Usage Usage
	// ContextTokens and ContextWindow report window pressure when known.
	ContextTokens int
	ContextWindow int

	// OutputPath is the saved final output ("agent://<id>" resolves here);
	// TranscriptPath is the child session JSONL.
	OutputPath     string
	TranscriptPath string

	// ExitCode is meaningful once Status is idle or aborted.
	ExitCode int
	// Error is a spawn or run failure, empty on success.
	Error string
	// Aborted reports that the run was cancelled rather than finishing.
	Aborted bool

	// Unread is the count of messages from this agent the human has not seen.
	Unread int
	// Children lists direct child ids, in spawn order.
	Children []string
}

// Running reports whether the agent holds live work.
func (e *Entry) Running() bool { return e.Status == StatusRunning }

// Elapsed is the wall-clock span the hub displays: live for a running agent,
// settled otherwise.
func (e *Entry) Elapsed(now time.Time) time.Duration {
	if e.StartedAt.IsZero() {
		return 0
	}
	end := e.EndedAt
	if e.Status == StatusRunning || end.IsZero() {
		end = now
	}
	if end.Before(e.StartedAt) {
		return 0
	}
	return end.Sub(e.StartedAt)
}

// Registry is the process-wide directory of agents for one session tree.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]*Entry
	order   []string
	// bySession maps a child session id to its agent id, so a revived session
	// resolves back to the same entry.
	bySession map[string]string
	listeners map[uint64]func(Event)
	nextSub   uint64
	now       func() time.Time
}

// EventKind names a registry change.
type EventKind string

const (
	EventRegistered EventKind = "registered"
	EventStatus     EventKind = "status"
	EventProgress   EventKind = "progress"
	EventSettled    EventKind = "settled"
	EventRemoved    EventKind = "removed"
	EventMessaged   EventKind = "messaged"
)

// Event is one registry change. Entry is a snapshot, never the live record.
type Event struct {
	Kind  EventKind
	Entry Entry
	// Message carries steer text for EventMessaged.
	Message string
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		entries:   map[string]*Entry{},
		bySession: map[string]string{},
		listeners: map[uint64]func(Event){},
		now:       time.Now,
	}
}

// SetClock replaces the time source, for tests.
func (r *Registry) SetClock(now func() time.Time) {
	r.mu.Lock()
	r.now = now
	r.mu.Unlock()
}

// Subscribe registers a listener and returns its cancel.
func (r *Registry) Subscribe(fn func(Event)) func() {
	if fn == nil {
		return func() {}
	}
	r.mu.Lock()
	id := r.nextSub
	r.nextSub++
	r.listeners[id] = fn
	r.mu.Unlock()
	return func() {
		r.mu.Lock()
		delete(r.listeners, id)
		r.mu.Unlock()
	}
}

// emit calls listeners with the snapshot. Callers hold no lock.
func (r *Registry) emit(kind EventKind, snapshot Entry, message string) {
	r.mu.RLock()
	listeners := make([]func(Event), 0, len(r.listeners))
	for _, fn := range r.listeners {
		listeners = append(listeners, fn)
	}
	r.mu.RUnlock()
	event := Event{Kind: kind, Entry: snapshot, Message: message}
	for _, fn := range listeners {
		fn(event)
	}
}

// Register adds an entry. It allocates a unique id from name: the first use of
// a name is the name itself and repeats gain "-2", "-3", matching omp's
// AgentOutputManager ids.
func (r *Registry) Register(entry Entry) Entry {
	r.mu.Lock()
	name := entry.Name
	if name == "" {
		name = "Task"
	}
	id := r.uniqueIDLocked(name, entry.ParentID)
	entry.ID = id
	entry.Name = name
	if entry.Status == "" {
		entry.Status = StatusRunning
	}
	now := r.now()
	if entry.StartedAt.IsZero() {
		entry.StartedAt = now
	}
	entry.LastActivityAt = now
	if entry.Depth == 0 {
		entry.Depth = 1
	}
	stored := entry
	r.entries[id] = &stored
	r.order = append(r.order, id)
	if parent := r.entries[entry.ParentID]; parent != nil {
		parent.Children = append(parent.Children, id)
	}
	snapshot := stored
	r.mu.Unlock()
	r.emit(EventRegistered, snapshot, "")
	return snapshot
}

// uniqueIDLocked builds a dot-qualified id unique within the registry.
func (r *Registry) uniqueIDLocked(name, parentID string) string {
	base := name
	if parentID != "" {
		base = parentID + "." + name
	}
	if _, taken := r.entries[base]; !taken {
		return base
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", base, n)
		if _, taken := r.entries[candidate]; !taken {
			return candidate
		}
	}
}

// Get returns a snapshot of one entry.
func (r *Registry) Get(id string) (Entry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[id]
	if !ok {
		return Entry{}, false
	}
	return *entry, true
}

// List returns snapshots in registration order.
func (r *Registry) List() []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Entry, 0, len(r.order))
	for _, id := range r.order {
		if entry, ok := r.entries[id]; ok {
			out = append(out, *entry)
		}
	}
	return out
}

// Sorted returns snapshots ordered for display: running first, then idle,
// then terminal, and by start time within a status.
func (r *Registry) Sorted() []Entry {
	entries := r.List()
	rank := func(s Status) int {
		switch s {
		case StatusRunning:
			return 0
		case StatusIdle:
			return 1
		case StatusParked:
			return 2
		default:
			return 3
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if ri, rj := rank(entries[i].Status), rank(entries[j].Status); ri != rj {
			return ri < rj
		}
		return entries[i].StartedAt.Before(entries[j].StartedAt)
	})
	return entries
}

// Update mutates one entry under the lock and emits the resulting snapshot.
// A nil fn leaves the entry unchanged. It reports whether the entry exists.
func (r *Registry) Update(id string, fn func(*Entry)) (Entry, bool) {
	if fn == nil {
		fn = func(*Entry) {}
	}
	r.mu.Lock()
	entry, ok := r.entries[id]
	if !ok {
		r.mu.Unlock()
		return Entry{}, false
	}
	fn(entry)
	entry.LastActivityAt = r.now()
	snapshot := *entry
	r.mu.Unlock()
	r.emit(EventProgress, snapshot, "")
	return snapshot, true
}

// SetStatus transitions an entry and emits a status event.
func (r *Registry) SetStatus(id string, status Status) (Entry, bool) {
	var kind EventKind = EventStatus
	r.mu.Lock()
	entry, ok := r.entries[id]
	if !ok {
		r.mu.Unlock()
		return Entry{}, false
	}
	entry.Status = status
	now := r.now()
	entry.LastActivityAt = now
	if status == StatusIdle || status == StatusAborted {
		entry.EndedAt = now
		kind = EventSettled
	}
	if status == StatusRunning {
		entry.EndedAt = time.Time{}
		kind = EventStatus
	}
	snapshot := *entry
	r.mu.Unlock()
	r.emit(kind, snapshot, "")
	return snapshot, true
}

// BindSession records the child session id an entry runs on.
func (r *Registry) BindSession(agentID, sessionID string) {
	if agentID == "" || sessionID == "" {
		return
	}
	r.mu.Lock()
	r.bySession[sessionID] = agentID
	r.mu.Unlock()
}

// BySession resolves a child session id to its agent entry.
func (r *Registry) BySession(sessionID string) (Entry, bool) {
	r.mu.RLock()
	id, ok := r.bySession[sessionID]
	r.mu.RUnlock()
	if !ok {
		return Entry{}, false
	}
	return r.Get(id)
}

// Remove deletes an entry, detaching it from its parent's children.
func (r *Registry) Remove(id string) {
	r.mu.Lock()
	entry, ok := r.entries[id]
	if !ok {
		r.mu.Unlock()
		return
	}
	snapshot := *entry
	delete(r.entries, id)
	r.order = deleteString(r.order, id)
	if parent := r.entries[entry.ParentID]; parent != nil {
		parent.Children = deleteString(parent.Children, id)
	}
	for session, agent := range r.bySession {
		if agent == id {
			delete(r.bySession, session)
		}
	}
	r.mu.Unlock()
	r.emit(EventRemoved, snapshot, "")
}

// NoteMessage records a steer or follow-up delivered to an agent.
func (r *Registry) NoteMessage(id, text string) {
	r.Update(id, func(e *Entry) { e.Unread++ })
	r.mu.RLock()
	entry, ok := r.entries[id]
	var snapshot Entry
	if ok {
		snapshot = *entry
	}
	r.mu.RUnlock()
	if ok {
		r.emit(EventMessaged, snapshot, text)
	}
}

// MarkRead clears an entry's unread counter.
func (r *Registry) MarkRead(id string) {
	r.Update(id, func(e *Entry) { e.Unread = 0 })
}

// Counts reports totals across measured agents for a hub header.
func (r *Registry) Counts() (running, total int, usage Usage) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, entry := range r.entries {
		total++
		if entry.Status == StatusRunning {
			running++
		}
		usage.Add(entry.Usage)
	}
	return running, total, usage
}

func deleteString(list []string, target string) []string {
	for i, value := range list {
		if value == target {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

// RandomSuffix returns a short random hex string for generated agent names.
func RandomSuffix() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "0000"
	}
	return hex.EncodeToString(b[:])
}

// SanitizeName makes a caller-supplied agent name safe for an id: no dots
// (they qualify nesting), no whitespace, and at most 32 characters.
func SanitizeName(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	for _, r := range name {
		switch {
		case r == '.' || r == '/' || r == '\\':
			b.WriteByte('-')
		case r <= ' ':
			continue
		default:
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return ""
	}
	if len(out) > 32 {
		out = out[:32]
	}
	return out
}
