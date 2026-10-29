package coding

// The session-backed subagent spawner. It turns a subagent.Spawner request into
// a child coding.Session: a fresh session file under the parent's artifact
// directory, the agent definition's tools and prompt, and the parent's model
// runtime and settings.
//
// Upstream: omp src/task/executor.ts runSubprocess. Divergences are recorded in
// docs/ompig/DIVERGENCES.md (D-SUBAGENT-1..4).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tmih06/ompig/agent"
	"github.com/tmih06/ompig/ai"
	"github.com/tmih06/ompig/coding/subagent"
	icodingagent "github.com/tmih06/ompig/internal/codingagent"
)

// SubagentRuntime owns the subagent registry, jobs, artifacts and spawner for
// one session tree.
//
// A Session creates one lazily with SubagentRuntime; the task tool and the
// Agent Hub both read it, so every surface sees the same agents.
type SubagentRuntime struct {
	runner  *subagent.Runner
	spawner *sessionSpawner
	once    sync.Once
	err     error

	parent *Session
	// restoreOnce guards the one-time roster scan from the artifact directory.
	restoreOnce sync.Once
}

// sessionSpawner builds child sessions from a parent Session.
type sessionSpawner struct {
	parent *Session
	runner *subagent.Runner

	mu sync.Mutex
	// live holds child sessions by agent id, so a parked agent can be revived
	// with its own transcript instead of a new one.
	live map[string]*Session
	// parkTimers holds each idle agent's pending auto-park, cancelled when the
	// agent becomes busy.
	parkTimers map[string]*time.Timer
	// idleTTL is how long an idle agent keeps a live session.
	idleTTL time.Duration
}

// SubagentRuntime returns the session's subagent runtime, creating it on first
// use. A nil runtime means the session cannot spawn subagents (no model
// runtime).
func (s *Session) SubagentRuntime() *SubagentRuntime {
	// A child session shares its spawner's runtime, so a nested spawn registers
	// in the same roster the user's hub shows, and shutdown happens once at the
	// root. Upstream: omp keeps one AgentRegistry per session tree.
	if s.subagentParent != nil {
		return s.subagentParent
	}
	s.subagentOnce.Do(func() {
		if s.services == nil {
			s.subagentErr = errors.New("coding: session has no services; subagents are unavailable")
			return
		}
		dir := s.subagentArtifactsDir()
		registry := subagent.NewRegistry()
		jobs := subagent.NewJobs(0)
		artifacts := subagent.NewArtifacts(dir)
		runner := subagent.NewRunner(registry, jobs, artifacts)
		// omp runs spawns in the background so a batch returns immediately and
		// the results arrive as jobs; a definition may still force one inline
		// with `blocking: true`.
		runner.Async = true
		// Two levels match omp's task.maxRecursionDepth default: a session, its
		// subagents, and their subagents.
		runner.MaxDepth = 2
		spawner := &sessionSpawner{parent: s, runner: runner, live: map[string]*Session{},
			parkTimers: map[string]*time.Timer{}, idleTTL: DefaultAgentIdleTTL}
		runner.Spawner = spawner
		// Bundled first, then project and user definitions override them by name.
		RegisterBuiltinDefinitions(runner)
		agentDir := ""
		if s.services != nil {
			agentDir = s.services.AgentDir()
		}
		defs, _ := subagent.Load(subagent.Discovery{
			ProjectDir: filepath.Join(s.CWD(), ".ompig", "agents"),
			UserDir:    filepath.Join(agentDir, "agents"),
		})
		for _, def := range defs {
			runner.AddDefinition(def)
		}
		// Progress events feed the hub's live tool and usage columns.
		jobs.SetObserver(func(job subagent.Job) {
			if job.State == subagent.JobRunning || job.State == subagent.JobDone || job.State == subagent.JobFailed || job.State == subagent.JobKilled {
				s.emitSubagentEvent(job)
			}
		})
		s.subagentRuntime = &SubagentRuntime{runner: runner, spawner: spawner, parent: s}
	})
	if s.subagentErr != nil {
		return nil
	}
	return s.subagentRuntime
}

// subagentArtifactsDir is where subagent outputs live: beside the session file
// when the session persists, else under a temporary directory.
func (s *Session) subagentArtifactsDir() string {
	if path := s.Path(); path != "" {
		base := strings.TrimSuffix(path, filepath.Ext(path))
		return base + "-agents"
	}
	dir, err := os.MkdirTemp("", "ompig-agents-")
	if err != nil {
		return ""
	}
	return dir
}

// Runner returns the underlying runner.
func (r *SubagentRuntime) Runner() *subagent.Runner {
	if r == nil {
		return nil
	}
	return r.runner
}

// Registry returns the agent registry.
func (r *SubagentRuntime) Registry() *subagent.Registry {
	if r == nil || r.runner == nil {
		return nil
	}
	return r.runner.Registry()
}

// Jobs returns the job manager.
func (r *SubagentRuntime) Jobs() *subagent.Jobs {
	if r == nil || r.runner == nil {
		return nil
	}
	return r.runner.Jobs()
}

// Steer delivers a message to a running or idle agent. A parked or aborted
// agent is revived first, which reopens its transcript.
//
// Upstream: omp `write agent://<id>` and the Agent Hub's steer action.
func (r *SubagentRuntime) Steer(ctx context.Context, id, message string) error {
	if r == nil || r.spawner == nil {
		return errors.New("subagents are unavailable")
	}
	entry, ok := r.Registry().Get(id)
	if !ok {
		return fmt.Errorf("unknown agent %q", id)
	}
	if entry.Kind == subagent.KindAdvisor {
		return fmt.Errorf("agent %q is an advisor transcript and cannot be messaged", id)
	}
	if entry.Status == subagent.StatusAborted {
		return fmt.Errorf("agent %q was killed; revive it before messaging", id)
	}
	session, err := r.spawner.ensureLive(ctx, entry)
	if err != nil {
		return err
	}
	r.stopParkTimer(id)
	r.Registry().NoteMessage(id, message)
	r.Registry().SetStatus(id, subagent.StatusRunning)
	// A follow-up is a run like any other: it settles the entry and re-arms the
	// idle timer, so the hub never shows a finished agent as running forever.
	err = session.SendQueued(ctx, message)
	r.settleAfterRun(id, err)
	return err
}

// settleAfterRun records the outcome of a follow-up run and re-arms the idle
// timer for a successful one.
func (r *SubagentRuntime) settleAfterRun(id string, runErr error) {
	if r == nil || r.spawner == nil {
		return
	}
	if runErr != nil {
		r.Registry().Update(id, func(e *subagent.Entry) {
			e.Error = runErr.Error()
			e.EndedAt = time.Now()
		})
		if errors.Is(runErr, context.Canceled) {
			r.Registry().SetStatus(id, subagent.StatusAborted)
			return
		}
	}
	r.Registry().SetStatus(id, subagent.StatusIdle)
	r.parkIdleOnSettle(id)
}

// Revive restores a parked or aborted agent so it can be messaged again.
//
// Upstream: omp Agent Hub `r` and the idle-TTL revival path.
func (r *SubagentRuntime) Revive(ctx context.Context, id string) error {
	if r == nil || r.spawner == nil {
		return errors.New("subagents are unavailable")
	}
	entry, ok := r.Registry().Get(id)
	if !ok {
		return fmt.Errorf("unknown agent %q", id)
	}
	if entry.Kind == subagent.KindAdvisor {
		return fmt.Errorf("agent %q is an advisor transcript and cannot be revived", id)
	}
	if entry.Status == subagent.StatusRunning || entry.Status == subagent.StatusIdle {
		return nil
	}
	if _, err := r.spawner.ensureLive(ctx, entry); err != nil {
		return err
	}
	r.Registry().SetStatus(id, subagent.StatusIdle)
	return nil
}

// Kill stops an agent: a running turn is aborted and the child session is
// disposed. The transcript survives, so a later message revives it.
//
// Upstream: omp Agent Hub `x`.
func (r *SubagentRuntime) Kill(ctx context.Context, id string) error {
	if r == nil || r.spawner == nil {
		return errors.New("subagents are unavailable")
	}
	entry, ok := r.Registry().Get(id)
	if !ok {
		return fmt.Errorf("unknown agent %q", id)
	}
	if entry.Kind == subagent.KindAdvisor {
		return fmt.Errorf("agent %q is an advisor transcript and cannot be killed", id)
	}
	r.Jobs().Kill(id)
	if session := r.spawner.take(id); session != nil {
		_ = session.Abort(ctx)
		session.Dispose()
	}
	r.Registry().SetStatus(id, subagent.StatusAborted)
	return nil
}

// Park disposes an idle agent's live session, keeping its transcript, so it can
// be revived later.
func (r *SubagentRuntime) Park(id string) {
	if r == nil || r.spawner == nil {
		return
	}
	r.stopParkTimer(id)
	// Mark parked before releasing the session: an observer that sees the
	// status must never find a live session for a parked agent, and one that
	// finds no session must see the parked status.
	r.Registry().SetStatus(id, subagent.StatusParked)
	if session := r.spawner.take(id); session != nil {
		session.Dispose()
	}
}

// ActiveSession returns the live child session for an agent, when one exists.
func (r *SubagentRuntime) ActiveSession(id string) (*Session, bool) {
	if r == nil || r.spawner == nil {
		return nil, false
	}
	r.spawner.mu.Lock()
	defer r.spawner.mu.Unlock()
	session, ok := r.spawner.live[id]
	return session, ok
}

// Shutdown aborts and disposes every live child session and cancels jobs.
func (r *SubagentRuntime) Shutdown() {
	if r == nil || r.spawner == nil {
		return
	}
	r.Jobs().KillAll()
	r.spawner.closeAll()
}

// take removes and returns a live session.
func (s *sessionSpawner) take(id string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.live[id]
	if !ok {
		return nil
	}
	delete(s.live, id)
	return session
}

// closeAll disposes every live session.
func (s *sessionSpawner) closeAll() {
	s.mu.Lock()
	sessions := make([]*Session, 0, len(s.live))
	for id, session := range s.live {
		sessions = append(sessions, session)
		delete(s.live, id)
	}
	for id, timer := range s.parkTimers {
		timer.Stop()
		delete(s.parkTimers, id)
	}
	s.mu.Unlock()
	for _, session := range sessions {
		session.Dispose()
	}
}

// ensureLive returns the agent's live session, reviving a parked or aborted one
// from its transcript when needed.
func (s *sessionSpawner) ensureLive(ctx context.Context, entry subagent.Entry) (*Session, error) {
	s.mu.Lock()
	if session, ok := s.live[entry.ID]; ok {
		s.mu.Unlock()
		return session, nil
	}
	s.mu.Unlock()

	transcript := entry.TranscriptPath
	if transcript == "" && s.runner != nil && s.runner.Artifacts() != nil {
		transcript = s.runner.Artifacts().TranscriptPath(entry.ID)
	}
	var (
		session *Session
		err     error
	)
	if transcript != "" {
		if _, statErr := os.Stat(transcript); statErr == nil {
			session, err = s.parent.resumeChildSession(transcript)
		}
	}
	if session == nil || err != nil {
		session, err = s.parent.newChildSession(entry)
	}
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.live[entry.ID] = session
	s.mu.Unlock()
	return session, nil
}

// Spawn creates a child session, runs the task, and returns the final text.
//
// Divergence D-SUBAGENT-1: the child is built from the parent's model runtime
// and settings directly rather than through a full option-resolution pipeline;
// per-agent model roles are honoured when the definition names one.
func (s *sessionSpawner) Spawn(ctx context.Context, req subagent.SpawnRequest) (subagent.SpawnResult, error) {
	entry := subagent.Entry{ID: req.AgentID, AgentType: req.Agent, Task: req.Task}
	def, _ := s.runnerDefinition(req.Agent)
	session, err := s.parent.newChildSessionWithDefinition(entry, def)
	if err != nil {
		return subagent.SpawnResult{}, err
	}
	key := req.AgentID
	if key == "" {
		key = req.Name
	}
	s.mu.Lock()
	if previous, replaced := s.live[key]; replaced && previous != session {
		// A second run under the same id replaces the live session; the old one
		// still owns a pump goroutine, so it is disposed rather than dropped.
		go previous.Dispose()
	}
	s.live[key] = session
	s.mu.Unlock()
	// Live columns: the hub shows the tool in flight and the tokens spent so
	// far, so the child's own event stream drives the registry entry.
	detach := session.Subscribe(func(event agent.AgentEvent) {
		s.observeChildEvent(key, event)
	})
	defer detach()

	prompt := req.Task
	if strings.TrimSpace(req.SolutionSpace) != "" {
		prompt += "\n\n## Solution space\n\n" + strings.TrimSpace(req.SolutionSpace)
	}
	if _, err := session.Send(ctx, prompt); err != nil {
		return subagent.SpawnResult{
			TranscriptPath: session.Path(),
			Model:          modelID(session.Model()),
		}, err
	}
	// The run is done: arm the idle timer so a forgotten agent releases its
	// live session while keeping its transcript.
	if runtime := s.parent.subagentRuntime; runtime != nil {
		runtime.parkIdleOnSettle(key)
	}
	stats := session.GetSessionStats()
	return subagent.SpawnResult{
		Output:         finalAssistantText(session),
		Usage:          usageOf(stats),
		TranscriptPath: session.Path(),
		Model:          modelID(session.Model()),
		ThinkingLevel:  string(session.ThinkingLevel()),
	}, nil
}

// runnerDefinition looks a definition up in the runner.
func (s *sessionSpawner) runnerDefinition(name string) (subagent.Definition, bool) {
	if s.runner == nil {
		return subagent.Definition{}, false
	}
	return s.runner.LookupDefinition(name)
}

// newChildSession builds a child session with the default agent definition.
func (s *Session) newChildSession(entry subagent.Entry) (*Session, error) {
	return s.newChildSessionWithDefinition(entry, subagent.Definition{Name: entry.AgentType})
}

// newChildSessionWithDefinition builds a child session over the parent's
// services, with the definition's tools and prompt.
//
// Divergence D-SUBAGENT-2: the child persists its own session file under the
// parent's artifact directory instead of the project session bucket, so a
// subagent never appears in the resume picker.
func (s *Session) newChildSessionWithDefinition(entry subagent.Entry, def subagent.Definition) (*Session, error) {
	svcs := s.services
	if svcs == nil {
		return nil, errors.New("coding: parent session has no services")
	}
	dir := ""
	if path := s.subagentSessionPath(entry.ID); path != "" {
		dir = filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("coding: create child session dir: %w", err)
		}
	}
	manager := icodingagent.NewSessionManagerWithDir(s.CWD(), dir)
	sessionID, idErr := icodingagent.GenerateSessionID()
	if idErr != nil {
		return nil, fmt.Errorf("coding: generate child session id: %w", idErr)
	}
	childLog, createErr := manager.Create(sessionID, "")
	if createErr != nil {
		return nil, fmt.Errorf("coding: create child session log: %w", createErr)
	}
	opts := SessionOptions{
		SessionManager: childLog,
		Model:          s.resolveChildModel(def),
		SystemPrompt:   def.SystemPrompt,
		// A child inherits the parent's runtime and settings, so credentials,
		// model catalog and extension configuration are identical.
		Services: svcs,
	}
	// The definition's tools select the child's built-ins. An explicit list is
	// an allowlist, so a read-only agent cannot write, and the child tools are
	// added only when the definition asks for them (directly or via spawns).
	// A definition that names no tools keeps the default set, as upstream does.
	if names := childToolNames(def); names != nil {
		allowed := make(map[string]struct{}, len(names))
		for _, name := range names {
			allowed[name] = struct{}{}
		}
		opts.AllowedTools = allowed
		opts.InitialActiveToolNames = append([]string(nil), names...)
	}
	if level := thinkingLevelOf(def.ThinkingLevel); level != "" {
		opts.ThinkingLevel = level
	}
	child, err := NewSession(svcs, opts)
	if err != nil {
		return nil, fmt.Errorf("coding: create child session: %w", err)
	}
	// One roster for the whole tree: the child's spawns land beside its parent's,
	// and nest under the child's own id.
	child.subagentParent = s.SubagentRuntime()
	child.subagentID = entry.ID
	return child, nil
}

// resumeChildSession reopens a child transcript.
func (s *Session) resumeChildSession(path string) (*Session, error) {
	svcs := s.services
	if svcs == nil {
		return nil, errors.New("coding: parent session has no services")
	}
	manager := icodingagent.NewSessionManagerWithDir(s.CWD(), filepath.Dir(path))
	opened, err := manager.Load(path)
	if err != nil {
		return nil, err
	}
	return NewSession(svcs, SessionOptions{SessionManager: opened, Services: svcs})
}

// subagentSessionPath is where a child's transcript lives.
// ompig divergence (D207): a child transcript lives beside the parent's session
// file, not in the project session bucket.
func (s *Session) subagentSessionPath(id string) string {
	if path := s.Path(); path != "" {
		base := strings.TrimSuffix(path, filepath.Ext(path))
		return filepath.Join(base+"-agents", sanitizeAgentFile(id)+".jsonl")
	}
	return ""
}

// sanitizeAgentFile keeps an id usable as a file name.
func sanitizeAgentFile(id string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", "..", "-")
	out := replacer.Replace(strings.TrimSpace(id))
	if out == "" {
		return "agent"
	}
	return out
}

// modelID is a model's display identity.
func modelID(model *ai.Model) string {
	if model == nil {
		return ""
	}
	id := model.ID
	if model.ProviderMeta.ProviderID != "" {
		return model.ProviderMeta.ProviderID + "/" + id
	}
	if model.Provider != nil {
		return model.Provider.ID() + "/" + id
	}
	return id
}

// thinkingLevelOf parses a definition's thinking selector.
func thinkingLevelOf(value string) ai.ModelThinkingLevel {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "off":
		return ai.ThinkingOff
	case "minimal":
		return ai.ThinkingMinimal
	case "low":
		return ai.ThinkingLow
	case "medium":
		return ai.ThinkingMedium
	case "high":
		return ai.ThinkingHigh
	case "xhigh":
		return ai.ThinkingXHigh
	case "max":
		return ai.ThinkingMax
	case "auto":
		return ai.ModelThinkingLevel("auto")
	}
	return ""
}

// usageOf converts session statistics into registry usage.
func usageOf(stats SessionStats) subagent.Usage {
	return subagent.Usage{
		Input:      int64(stats.Tokens.Input),
		Output:     int64(stats.Tokens.Output),
		CacheRead:  int64(stats.Tokens.CacheRead),
		CacheWrite: int64(stats.Tokens.CacheWrite),
		Total:      int64(stats.Tokens.Total),
		Cost:       stats.Cost,
		ToolCalls:  int64(stats.ToolCalls),
	}
}

// finalAssistantText returns the last assistant text in a child session.
func finalAssistantText(session *Session) string {
	if text := session.LastAssistantText(); text != nil {
		return *text
	}
	messages := session.Messages()
	for _, message := range slices.Backward(messages) {
		if message.Assistant != nil {
			var b strings.Builder
			for _, block := range message.Assistant.Content {
				if text, ok := block.(ai.TextContent); ok {
					b.WriteString(text.Text)
				}
			}
			return b.String()
		}
	}
	return ""
}

// emitSubagentEvent forwards a job transition to the session's listeners so the
// TUI can update the pinned subagent list.
func (s *Session) emitSubagentEvent(job subagent.Job) {
	s.subagentListenersMu.RLock()
	listeners := s.subagentListeners
	s.subagentListenersMu.RUnlock()
	for _, fn := range listeners {
		fn(job)
	}
}

// SubscribeSubagentJobs observes subagent job transitions and returns its
// cancel.
func (s *Session) SubscribeSubagentJobs(fn func(subagent.Job)) func() {
	if fn == nil {
		return func() {}
	}
	s.subagentListenersMu.Lock()
	s.subagentListeners = append(s.subagentListeners, fn)
	index := len(s.subagentListeners) - 1
	s.subagentListenersMu.Unlock()
	return func() {
		s.subagentListenersMu.Lock()
		if index < len(s.subagentListeners) {
			s.subagentListeners[index] = nil
		}
		s.subagentListenersMu.Unlock()
	}
}

// SendQueued delivers a message to a child session, queueing it as a steering
// message when a turn is already running.
func (s *Session) SendQueued(ctx context.Context, message string) error {
	if s.IsStreaming() {
		s.QueueSteer(message, nil)
		return nil
	}
	_, err := s.Send(ctx, message)
	return err
}

// StructuredOutput parses a child's structured result against its schema. It is
// a thin helper the task tool uses for outputSchema validation.
func StructuredOutput(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return raw, nil
}

// ChildAgentEnvironment is the environment a child session runs with, exposed so
// extensions can report which agent they execute for.
func ChildAgentEnvironment(agentID string) map[string]string {
	if agentID == "" {
		return nil
	}
	return map[string]string{"OMPIG_SUBAGENT_ID": agentID}
}

var _ = agent.RoleAssistant

// ReadAgentOutput returns an agent's saved final output, as `agent://<id>`
// resolves it. A live agent with no artifact yet falls back to its last
// assistant message.
func (r *SubagentRuntime) ReadAgentOutput(id string) (string, error) {
	if r == nil || r.runner == nil {
		return "", errors.New("subagents are unavailable")
	}
	if _, ok := r.Registry().Get(id); !ok {
		return "", fmt.Errorf("unknown agent %q", id)
	}
	if artifacts := r.runner.Artifacts(); artifacts != nil {
		if text, err := artifacts.ReadOutput(id); err == nil {
			return text, nil
		}
	}
	if session, live := r.ActiveSession(id); live {
		if text := finalAssistantText(session); text != "" {
			return text, nil
		}
	}
	return "", fmt.Errorf("agent %q has no saved output yet", id)
}

// AgentOutput reads an output by id, reporting whether the agent exists.
func (r *SubagentRuntime) AgentOutput(id string) (string, bool) {
	text, err := r.ReadAgentOutput(id)
	if err != nil {
		return "", false
	}
	return text, true
}

// SubagentRegistry implements the Agent Hub's roster source. On first use it
// restores the roster from disk, so a resumed session's hub shows the agents
// that session already spawned.
func (s *Session) SubagentRegistry() *subagent.Registry {
	runtime := s.SubagentRuntime()
	if runtime == nil {
		return nil
	}
	runtime.restoreOnce.Do(func() { runtime.RestoreRoster() })
	return runtime.Registry()
}

// SubagentJobs implements the Agent Hub's activity source.
func (s *Session) SubagentJobs() *subagent.Jobs {
	runtime := s.SubagentRuntime()
	if runtime == nil {
		return nil
	}
	return runtime.Jobs()
}

// SubagentSteer implements the Agent Hub's message action.
func (s *Session) SubagentSteer(ctx context.Context, id, message string) error {
	runtime := s.SubagentRuntime()
	if runtime == nil {
		return errors.New("subagents are unavailable")
	}
	return runtime.Steer(ctx, id, message)
}

// SubagentRevive implements the Agent Hub's revive action.
func (s *Session) SubagentRevive(ctx context.Context, id string) error {
	runtime := s.SubagentRuntime()
	if runtime == nil {
		return errors.New("subagents are unavailable")
	}
	return runtime.Revive(ctx, id)
}

// SubagentKill implements the Agent Hub's kill action.
func (s *Session) SubagentKill(ctx context.Context, id string) error {
	runtime := s.SubagentRuntime()
	if runtime == nil {
		return errors.New("subagents are unavailable")
	}
	return runtime.Kill(ctx, id)
}

// SubagentTranscript returns an agent's messages for the transcript view. A
// parked or killed agent still has its transcript on disk, so the view opens
// from the saved session rather than failing.
func (s *Session) SubagentTranscript(id string) ([]agent.AgentMessage, bool) {
	runtime := s.SubagentRuntime()
	if runtime == nil {
		return nil, false
	}
	if child, ok := runtime.ActiveSession(id); ok {
		return child.Messages(), true
	}
	if entry, ok := runtime.Registry().Get(id); ok && entry.TranscriptPath != "" {
		session, err := s.resumeChildSession(entry.TranscriptPath)
		if err != nil {
			return nil, false
		}
		defer session.Dispose()
		return session.Messages(), true
	}
	return nil, false
}

// observeChildEvent mirrors one child event into its registry entry, which is
// what the Agent Hub renders for tool, token and elapsed columns.
func (s *sessionSpawner) observeChildEvent(agentID string, event agent.AgentEvent) {
	if s.runner == nil {
		return
	}
	registry := s.runner.Registry()
	switch typed := event.(type) {
	case agent.ToolExecutionStartEvent:
		registry.Update(agentID, func(e *subagent.Entry) {
			e.CurrentTool = typed.ToolName
			e.CurrentArgs = truncateArgs(typed.Args)
		})
	case agent.ToolExecutionEndEvent:
		registry.Update(agentID, func(e *subagent.Entry) {
			e.CurrentTool = ""
			e.CurrentArgs = ""
			e.Usage.ToolCalls++
		})
	case agent.MessageEndEvent:
		if typed.Message.Assistant == nil {
			return
		}
		assistant := typed.Message.Assistant
		registry.Update(agentID, func(e *subagent.Entry) {
			e.Usage.Requests++
			if usage := assistant.Usage; usage != nil {
				e.Usage.Input += int64(usage.Input)
				e.Usage.Output += int64(usage.Output)
				e.Usage.CacheRead += int64(usage.CacheRead)
				e.Usage.CacheWrite += int64(usage.CacheWrite)
				e.Usage.Total += int64(usage.TotalTokens)
				e.Usage.Cost += usage.Cost.Total
				total := usage.Input + usage.CacheRead + usage.CacheWrite + usage.Output
				if total > e.ContextTokens {
					e.ContextTokens = total
				}
			}
			if e.Model == "" {
				e.Model = modelIDOf(assistant)
			}
		})
	}
}

// modelIDOf renders an assistant message's model identity.
func modelIDOf(message *agent.AssistantMessage) string {
	if message == nil || message.ModelID == "" {
		return ""
	}
	if message.Provider != "" {
		return message.Provider + "/" + message.ModelID
	}
	return message.ModelID
}

// truncateArgs keeps a tool argument summary short enough for one row.
func truncateArgs(args json.RawMessage) string {
	if len(args) == 0 {
		return ""
	}
	text := string(args)
	if len(text) > 200 {
		text = text[:200] + "…"
	}
	return text
}

// DefaultAgentIdleTTL matches omp's task.agentIdleTtlMs: an idle agent's session
// is released after this long, keeping its transcript for a revive.
const DefaultAgentIdleTTL = 7 * time.Minute

// parkIdleOnSettle arms the idle timer for an agent that just finished, so a
// forgotten subagent stops holding a live session. A non-positive TTL disables
// parking, which keeps every idle session live until the parent closes.
func (r *SubagentRuntime) parkIdleOnSettle(id string) {
	if r == nil || r.spawner == nil {
		return
	}
	ttl := r.spawner.idleTTL
	if ttl <= 0 {
		return
	}
	r.spawner.mu.Lock()
	if previous := r.spawner.parkTimers[id]; previous != nil {
		previous.Stop()
	}
	r.spawner.parkTimers[id] = time.AfterFunc(ttl, func() {
		entry, ok := r.Registry().Get(id)
		if !ok || entry.Status != subagent.StatusIdle {
			return
		}
		r.Park(id)
	})
	r.spawner.mu.Unlock()
}

// SetIdleTTL changes how long an idle agent keeps a live session. Zero or a
// negative duration disables auto-parking.
func (r *SubagentRuntime) SetIdleTTL(ttl time.Duration) {
	if r == nil || r.spawner == nil {
		return
	}
	r.spawner.idleTTL = ttl
}

// stopParkTimer cancels a pending idle timer when an agent becomes busy again.
func (r *SubagentRuntime) stopParkTimer(id string) {
	if r == nil || r.spawner == nil {
		return
	}
	r.spawner.mu.Lock()
	if timer := r.spawner.parkTimers[id]; timer != nil {
		timer.Stop()
		delete(r.spawner.parkTimers, id)
	}
	r.spawner.mu.Unlock()
}

// resolveChildModel applies the definition's model selectors to the parent's
// model. A selector that resolves to a known model wins; an unresolvable one
// keeps the parent's model, so a typo degrades to the default instead of
// failing the spawn.
func (s *Session) resolveChildModel(def subagent.Definition) *ai.Model {
	if len(def.Model) == 0 {
		return s.Model()
	}
	registry := s.ModelRegistry()
	if registry == nil {
		return s.Model()
	}
	for _, selector := range def.Model {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			continue
		}
		if model := resolveModelSelector(registry, selector); model != nil {
			return model
		}
	}
	return s.Model()
}

// resolveModelSelector matches a definition selector against the registry: an
// exact "provider/model-id", then an exact model id, then a fuzzy match. A
// leading "@role" selector is not resolved here because model roles are not
// part of this fork's settings surface.
func resolveModelSelector(registry *ModelRegistry, selector string) *ai.Model {
	if strings.HasPrefix(selector, "@") {
		return nil
	}
	wantProvider, wantID := "", selector
	if slash := strings.Index(selector, "/"); slash > 0 {
		wantProvider, wantID = selector[:slash], selector[slash+1:]
	}
	// Strip a thinking suffix: ":high" selects an effort level, not a model.
	if colon := strings.LastIndex(wantID, ":"); colon > 0 && isThinkingSuffix(wantID[colon+1:]) {
		wantID = wantID[:colon]
	}
	var fuzzy *ai.Model
	needle := strings.ToLower(wantID)
	for _, model := range registry.GetAll() {
		if model == nil {
			continue
		}
		provider := model.ProviderID()
		id := model.ID
		if wantProvider != "" && !strings.EqualFold(provider, wantProvider) {
			continue
		}
		if id == wantID {
			return model
		}
		if fuzzy == nil && strings.Contains(strings.ToLower(id), needle) {
			fuzzy = model
		}
	}
	return fuzzy
}

// isThinkingSuffix reports whether a selector suffix names a thinking level.
func isThinkingSuffix(value string) bool {
	switch strings.ToLower(value) {
	case "off", "minimal", "low", "medium", "high", "xhigh", "max", "auto":
		return true
	}
	return false
}

// childToolNames is the child's active tool set. A definition that names no
// tools keeps the default set (nil). A definition that names tools gets exactly
// those, minus the subagent tools it is not allowed to use: `task` needs a
// `spawns` grant, and `wait`/`agents` only make sense beside it.
func childToolNames(def subagent.Definition) []string {
	if def.Tools == nil {
		return nil
	}
	allowed := map[string]struct{}{}
	for _, name := range def.Tools {
		// omp names the file-finder `glob`, Pi names it `find`; a definition
		// written for omp keeps working.
		if name == "glob" {
			name = "find"
		}
		allowed[name] = struct{}{}
	}
	// An explicit tool list is the child's whole allowlist, so `task` appears
	// only when the definition names it AND may spawn; `wait` and `agents` ride
	// with it because they only control subagents.
	if !def.CanSpawn() {
		delete(allowed, "task")
		delete(allowed, "wait")
		delete(allowed, "agents")
	}
	names := make([]string, 0, len(allowed))
	for name := range allowed {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RestoreRoster repopulates the registry from the artifact directory, so the
// Agent Hub shows the subagents a resumed session already spawned. Each
// restored agent is parked: its live session is gone, but its transcript,
// output and revive path are intact.
//
// Upstream: omp Agent Hub "Opening the Hub for a persisted session scans that
// session's artifact tree. Historical subagent JSONL files become parked rows."
func (r *SubagentRuntime) RestoreRoster() int {
	if r == nil || r.runner == nil {
		return 0
	}
	artifacts := r.runner.Artifacts()
	if artifacts == nil {
		return 0
	}
	restored := 0
	for _, record := range artifacts.Scan() {
		if _, exists := r.Registry().Get(record.ID); exists {
			continue
		}
		// A restored row has no definition to name, so its type is unknown.
		parent, name := splitNestedID(record.ID)
		// Restored agents are terminal rows: no live session, so nothing can
		// claim they are running.
		r.Registry().Register(subagent.Entry{
			Name:           name,
			Kind:           subagent.KindTask,
			ParentID:       parent,
			Status:         subagent.StatusParked,
			AgentType:      "",
			Task:           "",
			OutputPath:     record.OutputPath,
			TranscriptPath: record.TranscriptPath,
			Usage:          subagent.Usage{Total: record.Size / 4},
		})
		restored++
	}
	return restored
}

// splitNestedID splits "Parent.Child" into its parent id and leaf name. A
// top-level id has no parent.
func splitNestedID(id string) (parent, name string) {
	if at := strings.LastIndex(id, "."); at > 0 {
		return id[:at], id[at+1:]
	}
	return "", id
}

// SetArtifactsForTest replaces the artifact store, so a test can point a fresh
// runtime at another session's directory. It is not for production callers.
func (r *SubagentRuntime) SetArtifactsForTest(artifacts *subagent.Artifacts) {
	if r == nil || r.runner == nil {
		return
	}
	r.runner.SetArtifacts(artifacts)
}
