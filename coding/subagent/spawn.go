package subagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tmih06/ompig/ai"
)

// SpawnRequest is one unit of subagent work.
//
// Upstream: omp task item `{ name?, agent?, task, solutionSpace, ... }`. The
// batch `context` field is carried on BatchRequest instead, because it is
// shared by every item.
type SpawnRequest struct {
	// Name is the stable agent id. Empty generates an AdjectiveNoun-style id.
	Name string
	// Agent selects an agent definition; empty uses the default.
	Agent string
	// Task is the self-contained assignment.
	Task string
	// SolutionSpace describes how open-ended the assignment is. It rides the
	// child's first prompt so an auto thinking classifier sees it.
	SolutionSpace string
	// OutputSchema, when set, requires a structured result.
	OutputSchema json.RawMessage
	// AgentID is the registry id the runner allocated for this spawn. It is set
	// by the runner before calling the Spawner and is the only key a spawner may
	// use for per-agent state: caller-supplied names collide.
	AgentID string
}

// BatchRequest is one `task` tool call.
type BatchRequest struct {
	// Context is shared background rendered into every child's prompt.
	Context string
	Tasks   []SpawnRequest
}

// Result is the outcome of one spawn.
type Result struct {
	ID     string
	Name   string
	Agent  string
	Task   string
	Status Status

	Output string
	// Structured is the validated structured payload, when a schema was asked
	// for and validation passed.
	Structured json.RawMessage
	// StructuredError reports why a requested schema was not satisfied: invalid
	// JSON, or a value the schema rejects.
	StructuredError string

	ExitCode int
	Error    string
	Aborted  bool

	Usage      Usage
	DurationMs int64
	Requests   int64

	Model         string
	ThinkingLevel string

	OutputPath     string
	TranscriptPath string
}

// Spawner runs one spawn. A session implements it by creating a child session;
// a test implements it with a stub. It returns the final text and the
// structured payload.
type Spawner interface {
	Spawn(ctx context.Context, req SpawnRequest) (SpawnResult, error)
}

// SpawnResult is what a Spawner returns.
type SpawnResult struct {
	Output     string
	Structured json.RawMessage
	Usage      Usage
	// TranscriptPath is the child session file, when the child persisted one.
	TranscriptPath string
	Model          string
	ThinkingLevel  string
	// Err is a run failure that still produced output. A non-nil error return
	// from Spawn means the spawn never ran.
	Err error
}

// Runner executes spawn requests, registers agents, and delivers results.
//
// Upstream: omp src/task/executor.ts and src/task/index.ts. The runner is the
// piece a session owns; the tool and the hub both talk to it, so they see the
// same agents.
type Runner struct {
	registry  *Registry
	jobs      *Jobs
	artifacts *Artifacts

	// Spawner builds the child session. Required.
	Spawner Spawner
	// Definitions are the discoverable agents, keyed by name.
	Definitions map[string]Definition
	// DefaultAgent is used when a request names none.
	DefaultAgent string
	// MaxConcurrency bounds concurrently running spawns. 0 is unbounded.
	MaxConcurrency int
	// Async runs spawns in the background. When false the call blocks, which is
	// what print mode and a small terminal want.
	Async bool
	// MaxDepth bounds nested spawning; 0 disables the cap.
	MaxDepth int
	// MaxOutputBytes caps the inline output a result carries; the full text is
	// always written to the artifact file.
	MaxOutputBytes int
	// BriefOutputCap is the size the model sees inline.
	BriefOutputCap int

	defsMu  sync.RWMutex
	semOnce sync.Once
	sem     chan struct{}
	// depth tracks the nesting level of the calling agent, by agent id.
	depthMu sync.Mutex
	depth   map[string]int
	now     func() time.Time
}

// defaultOutputCap mirrors omp's MAX_OUTPUT_BYTES.
const defaultOutputCap = 500_000

// defaultBriefCap mirrors omp's FULL_OUTPUT_THRESHOLD.
const defaultBriefCap = 5_000

// NewRunner builds a runner over the given registry, jobs and artifacts.
func NewRunner(registry *Registry, jobs *Jobs, artifacts *Artifacts) *Runner {
	if registry == nil {
		registry = NewRegistry()
	}
	if jobs == nil {
		jobs = NewJobs(0)
	}
	return &Runner{
		registry:       registry,
		jobs:           jobs,
		artifacts:      artifacts,
		Definitions:    map[string]Definition{},
		DefaultAgent:   "task",
		MaxConcurrency: 32,
		MaxOutputBytes: defaultOutputCap,
		BriefOutputCap: defaultBriefCap,
		depth:          map[string]int{},
		now:            time.Now,
	}
}

// Registry returns the agent registry.
func (r *Runner) Registry() *Registry { return r.registry }

// Jobs returns the background job manager.
func (r *Runner) Jobs() *Jobs { return r.jobs }

// Artifacts returns the artifact store.
func (r *Runner) Artifacts() *Artifacts { return r.artifacts }

// AddDefinition registers one agent definition.
func (r *Runner) AddDefinition(def Definition) {
	r.defsMu.Lock()
	if r.Definitions == nil {
		r.Definitions = map[string]Definition{}
	}
	r.Definitions[def.Name] = def
	r.defsMu.Unlock()
}

// AddDefinitions registers definitions, first name wins.
func (r *Runner) AddDefinitions(defs []Definition) {
	for _, def := range defs {
		r.AddDefinition(def)
	}
}

// resolveAgent selects the agent definition for a request. An unknown name is
// an error that lists what is available, as upstream preflight does.
func (r *Runner) resolveAgent(requested string) (Definition, error) {
	r.defsMu.RLock()
	defer r.defsMu.RUnlock()
	name := requested
	if name == "" {
		name = r.DefaultAgent
	}
	if name == "" {
		return Definition{Name: "task"}, nil
	}
	def, ok := r.Definitions[name]
	if !ok {
		available := make([]string, 0, len(r.Definitions))
		for candidate := range r.Definitions {
			available = append(available, candidate)
		}
		sort.Strings(available)
		return Definition{}, fmt.Errorf("unknown agent %q. Available: %s", name, strings.Join(available, ", "))
	}
	return def, nil
}

// childDepth reports the nesting depth a spawn would run at, given the calling
// agent id (empty for the session itself).
func (r *Runner) childDepth(parentID string) int {
	if parentID == "" {
		return 1
	}
	r.depthMu.Lock()
	defer r.depthMu.Unlock()
	return r.depth[parentID] + 1
}

// setDepth records the depth an agent runs at, so its own spawns nest.
func (r *Runner) setDepth(id string, depth int) {
	if id == "" {
		return
	}
	r.depthMu.Lock()
	r.depth[id] = depth
	r.depthMu.Unlock()
}

// acquire takes a concurrency slot, honouring ctx. The slot pool is sized once,
// on first use, so concurrent callers cannot race the initialization.
func (r *Runner) acquire(ctx context.Context) (func(), error) {
	if r.MaxConcurrency <= 0 {
		return func() {}, nil
	}
	sem := r.semaphore()
	select {
	case sem <- struct{}{}:
		return func() { <-sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// semaphore returns the slot pool, creating it once.
func (r *Runner) semaphore() chan struct{} {
	r.semOnce.Do(func() {
		size := r.MaxConcurrency
		if size <= 0 {
			size = 1
		}
		r.sem = make(chan struct{}, size)
	})
	return r.sem
}

// noteProgress records tool activity for the hub.
func (r *Runner) noteTool(agentID, toolName, intent string, args json.RawMessage) {
	r.registry.Update(agentID, func(e *Entry) {
		e.CurrentTool = toolName
		e.CurrentIntent = intent
		if len(args) > 0 {
			text := string(args)
			if len(text) > 240 {
				text = text[:240] + "…"
			}
			e.CurrentArgs = text
		}
	})
}

// clearTool clears the in-flight tool once a call ends.
func (r *Runner) clearTool(agentID string) {
	r.registry.Update(agentID, func(e *Entry) {
		e.CurrentTool = ""
		e.CurrentIntent = ""
		e.CurrentArgs = ""
	})
}

// Run executes one spawn request, registering the agent first so the hub sees
// it while it runs.
func (r *Runner) Run(ctx context.Context, parentID string, req SpawnRequest) (Result, error) {
	entry, def, err := r.reserve(parentID, req)
	if err != nil {
		return Result{}, err
	}
	result := r.execute(ctx, entry, def, req)
	return result, nil
}

// reserve validates a spawn and registers its agent, so an asynchronous caller
// can report the id before the work starts. The returned entry is the live
// registry record the spawn updates.
func (r *Runner) reserve(parentID string, req SpawnRequest) (Entry, Definition, error) {
	if strings.TrimSpace(req.Task) == "" {
		return Entry{}, Definition{}, errors.New("task must not be empty")
	}
	def, err := r.resolveAgent(req.Agent)
	if err != nil {
		return Entry{}, Definition{}, err
	}
	depth := r.childDepth(parentID)
	if r.MaxDepth > 0 && depth > r.MaxDepth {
		return Entry{}, Definition{}, fmt.Errorf("task recursion depth %d exceeds the maximum of %d", depth, r.MaxDepth)
	}
	// The spawning agent's own definition governs what it may delegate: a
	// `spawns` list that omits this agent denies the spawn before any child
	// session exists.
	if parent := r.parentDefinition(parentID); parent != nil && !parent.CanSpawnAgent(def.Name) {
		return Entry{}, Definition{}, fmt.Errorf("agent %q may not spawn %q", parent.Name, def.Name)
	}
	name := SanitizeName(req.Name)
	if name == "" {
		name = def.Name
		if name == "" {
			name = "Task"
		}
	}
	entry := r.registry.Register(Entry{
		Name:      name,
		Kind:      KindTask,
		ParentID:  parentID,
		Depth:     depth,
		Task:      req.Task,
		AgentType: def.Name,
		Status:    StatusRunning,
	})
	r.setDepth(entry.ID, depth)
	return entry, def, nil
}

// execute runs one registered spawn to completion and records its outcome.
func (r *Runner) execute(ctx context.Context, entry Entry, def Definition, req SpawnRequest) Result {
	started := r.now()
	release, err := r.acquire(ctx)
	if err != nil {
		return r.finish(entry, Result{Status: StatusAborted, Aborted: true, Error: err.Error()}, started)
	}
	defer release()

	if r.Spawner == nil {
		out := Result{Status: StatusAborted, Error: "no spawner configured"}
		return r.finish(entry, out, started)
	}

	spawnReq := req
	if spawnReq.Agent == "" {
		spawnReq.Agent = def.Name
	}
	spawnReq.AgentID = entry.ID
	spawned, spawnErr := r.Spawner.Spawn(ctx, spawnReq)

	out := Result{
		ID:             entry.ID,
		Name:           entry.ID,
		Agent:          def.Name,
		Task:           req.Task,
		Output:         spawned.Output,
		Structured:     spawned.Structured,
		Usage:          spawned.Usage,
		Model:          spawned.Model,
		ThinkingLevel:  spawned.ThinkingLevel,
		TranscriptPath: spawned.TranscriptPath,
		Status:         StatusIdle,
	}
	if spawnErr != nil {
		out.Status = StatusAborted
		out.Aborted = errors.Is(spawnErr, context.Canceled)
		out.ExitCode = 1
		out.Error = spawnErr.Error()
	} else if spawned.Err != nil {
		out.ExitCode = 1
		out.Error = spawned.Err.Error()
	}
	if ctx.Err() != nil {
		out.Status = StatusAborted
		out.Aborted = true
		if out.Error == "" {
			out.Error = ctx.Err().Error()
		}
	}

	// A requested output schema must be satisfied by the child's final message,
	// parsed as JSON. A schema-invalid payload is kept but reported, so the
	// caller can tell a contract failure from a run failure.
	if len(req.OutputSchema) > 0 && out.Output != "" {
		if structured, structErr := validateStructuredOutput(req.OutputSchema, out.Output); structErr != nil {
			out.StructuredError = structErr.Error()
		} else {
			out.Structured = structured
		}
	}

	// Persist the output so `agent://<id>` resolves even after the agent parks.
	if r.artifacts != nil {
		if path, writeErr := r.artifacts.WriteOutput(entry.ID, out.Output, out.Structured); writeErr == nil {
			out.OutputPath = path
		}
	}
	if out.TranscriptPath != "" && r.artifacts != nil {
		// The child session manager names the transcript itself, so its real
		// path is recorded for a later roster restore.
		_ = r.artifacts.WriteTranscriptNote(entry.ID, out.TranscriptPath)
	}
	if out.TranscriptPath == "" && r.artifacts != nil {
		out.TranscriptPath = r.artifacts.TranscriptPath(entry.ID)
	}
	return r.finish(entry, out, started)
}

// finish records the settled entry and publishes the result.
func (r *Runner) finish(entry Entry, out Result, started time.Time) Result {
	out.DurationMs = r.now().Sub(started).Milliseconds()
	if out.Requests == 0 {
		out.Requests = out.Usage.Requests
	}
	status := out.Status
	if status == "" {
		status = StatusIdle
	}
	r.registry.Update(entry.ID, func(e *Entry) {
		e.Status = status
		e.Usage.Add(out.Usage)
		e.OutputPath = out.OutputPath
		e.TranscriptPath = out.TranscriptPath
		e.ExitCode = out.ExitCode
		e.Error = out.Error
		e.Aborted = out.Aborted
		e.Model = out.Model
		e.ThinkingLevel = out.ThinkingLevel
		e.CurrentTool = ""
		e.CurrentIntent = ""
		e.CurrentArgs = ""
		if status != StatusRunning {
			e.EndedAt = r.now()
		}
	})
	return out
}

// SpawnBatch runs a batch request. With Async the calls return immediately as
// background jobs; otherwise it blocks and returns every result. Blocking items
// run inline even when Async is on, and independent items run concurrently: the
// runner's semaphore, not the call, is the concurrency bound.
//
// Upstream: omp task tool (task.batch), where an item's agent may declare
// `blocking: true` and every other item becomes a background job.
func (r *Runner) SpawnBatch(ctx context.Context, parentID string, batch BatchRequest) ([]Result, error) {
	if len(batch.Tasks) == 0 {
		return nil, errors.New("tasks must not be empty")
	}
	plans := make([]spawnPlan, 0, len(batch.Tasks))
	for index, item := range batch.Tasks {
		req := item
		req.Task = withSharedContext(batch.Context, req.Task)
		def, err := r.resolveAgent(req.Agent)
		if err != nil {
			return nil, err
		}
		plans = append(plans, spawnPlan{req: req, def: def, blocking: def.Blocking, index: index})
	}

	results := make([]Result, len(plans))
	if r.Async {
		var inline []spawnPlan
		for _, plan := range plans {
			if plan.blocking {
				inline = append(inline, plan)
				continue
			}
			// Reserve the agent now so the call can name it and the hub can show
			// it while the job runs; the job only executes the work.
			entry, def, err := r.reserve(parentID, plan.req)
			if err != nil {
				return nil, err
			}
			request := plan.req
			if _, err := r.jobs.Register(ctx, entry.ID, "task", firstLine(request.Task), parentID,
				func(jobCtx context.Context) (string, error) {
					result := r.execute(jobCtx, entry, def, request)
					if result.Error != "" && result.Status != StatusIdle {
						return result.Output, errors.New(result.Error)
					}
					return result.Output, nil
				}); err != nil {
				r.registry.Remove(entry.ID)
				return nil, err
			}
			results[plan.index] = Result{
				ID:             entry.ID,
				Name:           entry.ID,
				Agent:          def.Name,
				Task:           request.Task,
				Status:         StatusRunning,
				TranscriptPath: r.transcriptPath(entry.ID),
			}
		}
		// Blocking items still run inline and return their result.
		r.runConcurrently(ctx, parentID, inline, results)
		return results, nil
	}
	r.runConcurrently(ctx, parentID, plans, results)

	// A spawn error that prevented a child from existing fails the call.
	for _, result := range results {
		if result.Error == "" && result.Status == "" {
			return results, errors.New("a task item produced no result")
		}
	}
	return results, nil
}

// runConcurrently runs the planned spawns in parallel and fills results by
// index. The first pre-spawn failure is returned through the slot's Error.
func (r *Runner) runConcurrently(ctx context.Context, parentID string, plans []spawnPlan, results []Result) {
	var wg sync.WaitGroup
	for _, plan := range plans {
		wg.Add(1)
		go func(plan spawnPlan) {
			defer wg.Done()
			result, err := r.Run(ctx, parentID, plan.req)
			if err != nil {
				name := SanitizeName(plan.req.Name)
				if name == "" {
					name = plan.def.Name
				}
				results[plan.index] = Result{
					ID: name, Name: name, Agent: plan.def.Name,
					Task: plan.req.Task, Status: StatusAborted, Error: err.Error(),
				}
				return
			}
			results[plan.index] = result
		}(plan)
	}
	wg.Wait()
}

// parentDefinition returns the definition the parent agent runs, when it has
// one. A session-spawned agent has none, so it inherits the default policy.
func (r *Runner) parentDefinition(parentID string) *Definition {
	if parentID == "" {
		return nil
	}
	r.defsMu.RLock()
	name := ""
	if entry, ok := r.registry.Get(parentID); ok {
		name = entry.AgentType
	}
	r.defsMu.RUnlock()
	if name == "" {
		return nil
	}
	def, ok := r.LookupDefinition(name)
	if !ok {
		return nil
	}
	return &def
}

// spawnPlan is one resolved batch item: its request, the agent it runs, and
// whether it must block.
type spawnPlan struct {
	req      SpawnRequest
	def      Definition
	blocking bool
	index    int
}

// withSharedContext prefixes the batch's shared background to one task.
func withSharedContext(shared, task string) string {
	shared = strings.TrimSpace(shared)
	if shared == "" {
		return task
	}
	return "## Context\n\n" + shared + "\n\n## Task\n\n" + task
}

// firstLine ends at the first line break, for a one-line job label.
func firstLine(text string) string {
	if at := strings.IndexByte(text, '\n'); at >= 0 {
		text = text[:at]
	}
	if len(text) > 80 {
		text = text[:80] + "…"
	}
	return strings.TrimSpace(text)
}

// Summarize renders a compact result summary for the model, with the full text
// left in `agent://<id>` when it is too long.
func (r *Runner) Summarize(result Result) string {
	cap := r.BriefOutputCap
	if cap <= 0 {
		cap = defaultBriefCap
	}
	output := result.Output
	truncated := false
	if len(output) > cap {
		output = output[:cap]
		truncated = true
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Agent `%s` (%s) %s in %dms", result.ID, result.Agent, statusWord(result.Status), result.DurationMs)
	if result.Error != "" {
		fmt.Fprintf(&b, ": %s", result.Error)
	}
	b.WriteString("\n")
	if output != "" {
		b.WriteString(output)
		if truncated {
			fmt.Fprintf(&b, "\n[output truncated; read agent://%s for the full text]", result.ID)
		}
	} else {
		b.WriteString("(no output)")
	}
	return b.String()
}

// statusWord is the human form of a settlement status.
func statusWord(status Status) string {
	switch status {
	case StatusRunning:
		return "is running"
	case StatusIdle:
		return "finished"
	case StatusAborted:
		return "was aborted"
	case StatusParked:
		return "is parked"
	default:
		return string(status)
	}
}

// LookupDefinition returns a registered agent definition.
func (r *Runner) LookupDefinition(name string) (Definition, bool) {
	if r == nil {
		return Definition{}, false
	}
	r.defsMu.RLock()
	defer r.defsMu.RUnlock()
	def, ok := r.Definitions[name]
	return def, ok
}

// transcriptPath is where an agent's transcript will live, when the artifact
// store knows.
func (r *Runner) transcriptPath(id string) string {
	if r.artifacts == nil {
		return ""
	}
	return r.artifacts.TranscriptPath(id)
}

// SetArtifacts replaces the artifact store.
func (r *Runner) SetArtifacts(artifacts *Artifacts) {
	if r == nil {
		return
	}
	r.artifacts = artifacts
}

// validateStructuredOutput extracts the JSON value from a child's final message
// and validates it against the requested schema. A fenced json block is
// accepted, because models routinely wrap a JSON answer in Markdown.
func validateStructuredOutput(schema json.RawMessage, output string) (json.RawMessage, error) {
	payload := extractJSONOutput(output)
	if len(payload) == 0 {
		return nil, errors.New("the result contains no JSON value to validate against outputSchema")
	}
	validated, err := ai.ValidateToolArgumentsJSON("outputSchema", schema, payload)
	if err != nil {
		return nil, fmt.Errorf("outputSchema rejected the result: %w", err)
	}
	return validated, nil
}

// extractJSONOutput returns the JSON value a reply carries: the whole text when
// it parses, else the contents of the last fenced block that parses.
func extractJSONOutput(output string) json.RawMessage {
	trimmed := strings.TrimSpace(output)
	if json.Valid([]byte(trimmed)) {
		return json.RawMessage(trimmed)
	}
	// Walk fenced blocks from the end: a model's final answer is the last one.
	rest := trimmed
	for {
		open := strings.Index(rest, "```")
		if open < 0 {
			return nil
		}
		after := rest[open+3:]
		// Skip a language tag on the opening fence line.
		if newline := strings.IndexByte(after, '\n'); newline >= 0 && !strings.Contains(after[:newline], "}") {
			after = after[newline+1:]
		}
		close := strings.Index(after, "```")
		if close < 0 {
			return nil
		}
		block := strings.TrimSpace(after[:close])
		if json.Valid([]byte(block)) {
			candidate := json.RawMessage(block)
			// Keep scanning earlier blocks only if this one is not the last.
			if next := after[close+3:]; strings.Contains(next, "```") {
				rest = next
				continue
			}
			return candidate
		}
		rest = after[close+3:]
	}
}
