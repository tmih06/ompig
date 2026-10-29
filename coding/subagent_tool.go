package coding

// The `task`, `wait` and `agents` tools: the model-facing surface of the
// subagent runtime.
//
// Upstream: omp src/tools/task.ts, tools/wait.ts, and the `agent://` protocol.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/tmih06/ompig/agent"
	"github.com/tmih06/ompig/ai"
	"github.com/tmih06/ompig/coding/subagent"
)

// waitTimeoutCap bounds a `wait` call, matching omp's 30 minute cap.
const waitTimeoutCap = 30 * time.Minute

// taskToolParams is the batch-shaped `task` schema, plus the flat fields so a
// caller that omits `tasks` still works.
type taskToolParams struct {
	Context string         `json:"context,omitempty"`
	Tasks   []taskToolItem `json:"tasks,omitempty"`
	Task    string         `json:"task,omitempty"`
	Name    string         `json:"name,omitempty"`
	Agent   string         `json:"agent,omitempty"`
	// SolutionSpace describes how open-ended the child's problem is.
	SolutionSpace string `json:"solutionSpace,omitempty"`
}

type taskToolItem struct {
	Name          string          `json:"name,omitempty"`
	Agent         string          `json:"agent,omitempty"`
	Task          string          `json:"task"`
	SolutionSpace string          `json:"solutionSpace,omitempty"`
	OutputSchema  json.RawMessage `json:"outputSchema,omitempty"`
}

// TaskTool returns the `task` tool bound to this session's subagent runtime.
//
// The tool is sequential: two parallel spawns of the same name would race on
// the artifact store, and omp's concurrency bound is the runner's semaphore,
// not the agent loop's.
func (s *Session) TaskTool() agent.AgentTool {
	binding := &subagentToolBinding{}
	binding.set(s)
	return &subagentTaskTool{binding: binding}
}

type subagentTaskTool struct{ binding *subagentToolBinding }

// session returns the session this tool drives.
func (t *subagentTaskTool) session() *Session { return t.binding.get() }

// Name is the tool id the model calls.
func (t *subagentTaskTool) Name() string { return "task" }

// Label is the human-facing title.
func (t *subagentTaskTool) Label() string { return "Task" }

// ExecutionMode keeps spawn registration ordered.
func (t *subagentTaskTool) ExecutionMode() agent.ToolExecutionMode {
	return agent.ToolModeSequential
}

// Schema is the model-facing contract.
func (t *subagentTaskTool) Schema() ai.ToolSchema {
	return ai.ToolSchema{
		Name:        "task",
		Description: taskToolDescription,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"context": map[string]any{
					"type":        "string",
					"description": "Shared background rendered into every spawned subagent's prompt.",
				},
				"tasks": map[string]any{
					"type":        "array",
					"description": "One subagent per item. Preferred shape.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name":          map[string]any{"type": "string", "description": "Stable agent name, CamelCase, at most 32 characters."},
							"agent":         map[string]any{"type": "string", "description": "Agent type to run, e.g. scout or reviewer."},
							"task":          map[string]any{"type": "string", "description": "Complete, self-contained instructions."},
							"solutionSpace": map[string]any{"type": "string", "description": "How open-ended the assignment is: what is given and what remains open."},
							"outputSchema":  map[string]any{"type": "object", "description": "JSON Schema the child's structured result must satisfy."},
						},
						"required": []string{"task", "solutionSpace"},
					},
				},
				"task":          map[string]any{"type": "string", "description": "Single-spawn form: the work to do."},
				"name":          map[string]any{"type": "string", "description": "Single-spawn form: the agent name."},
				"agent":         map[string]any{"type": "string", "description": "Single-spawn form: the agent type."},
				"solutionSpace": map[string]any{"type": "string", "description": "Single-spawn form: how open-ended the assignment is."},
			},
		},
		PromptGuidelines: []string{
			"Prefer one `task` call with several `tasks` over sequential calls.",
			"Give each subagent a self-contained task; it does not see your conversation.",
			"Message an existing agent with `agents` instead of spawning a replacement.",
		},
	}
}

const taskToolDescription = "Spawn subagents to work in parallel. Each \"tasks\" item runs in its own\n" +
	"context: give complete instructions, and put shared background in \"context\".\n" +
	"\"solutionSpace\" says how open-ended the work is, not how large; it tunes reasoning.\n\n" +
	"Ask for the specific facts, files or edits you need. Read a result with\n" +
	"\"agent://<id>\"; message the agent with \"agents\" instead of spawning a replacement."

// Execute runs the spawns.
func (t *subagentTaskTool) Execute(ctx context.Context, toolCallID string, params json.RawMessage, onUpdate agent.ToolUpdateCallback) (agent.AgentToolResult, error) {
	runtime := subagentRuntimeOf(t)
	if runtime == nil {
		return toolError("subagents are unavailable in this session"), nil
	}
	var parsed taskToolParams
	if err := json.Unmarshal(params, &parsed); err != nil {
		return toolError("invalid task arguments: " + err.Error()), nil
	}
	items := parsed.Tasks
	if len(items) == 0 {
		if strings.TrimSpace(parsed.Task) == "" {
			return toolError("task requires `tasks` or `task`"), nil
		}
		items = []taskToolItem{{
			Name:          parsed.Name,
			Agent:         parsed.Agent,
			Task:          parsed.Task,
			SolutionSpace: parsed.SolutionSpace,
		}}
	}
	requests := make([]subagent.SpawnRequest, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Task) == "" {
			return toolError("every task item needs a non-empty `task`"), nil
		}
		requests = append(requests, subagent.SpawnRequest{
			Name:          item.Name,
			Agent:         item.Agent,
			Task:          item.Task,
			SolutionSpace: item.SolutionSpace,
			OutputSchema:  item.OutputSchema,
		})
	}

	runner := runtime.Runner()
	results, err := runner.SpawnBatch(ctx, t.session().subagentID, subagent.BatchRequest{Context: parsed.Context, Tasks: requests})
	if err != nil {
		return toolError("spawn failed: " + err.Error()), nil
	}
	if runtime.Runner().Async {
		var b strings.Builder
		fmt.Fprintf(&b, "Spawned %d background agent(s). Results are delivered when each settles.\n", len(results))
		for _, result := range results {
			fmt.Fprintf(&b, "- `%s` (%s)\n", result.ID, result.Agent)
		}
		b.WriteString("\nLeave eval and call `wait` when you are blocked; do not poll.")
		return agent.AgentToolResult{
			Content: []ai.ToolResultMessageContent{ai.TextContent{Text: b.String()}},
			Details: results,
		}, nil
	}
	var b strings.Builder
	for i, result := range results {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(runner.Summarize(result))
	}
	return agent.AgentToolResult{
		Content: []ai.ToolResultMessageContent{ai.TextContent{Text: b.String()}},
		Details: results,
	}, nil
}

// subagentRuntimeOrNil resolves the runtime the tool acts on.
func subagentRuntimeOf(tool interface{ session() *Session }) *SubagentRuntime {
	session := tool.session()
	if session == nil {
		return nil
	}
	return session.SubagentRuntime()
}

// toolError is a tool-level failure the model sees and can correct.
func toolError(message string) agent.AgentToolResult {
	return agent.AgentToolResult{
		Content: []ai.ToolResultMessageContent{ai.TextContent{Text: message}},
		IsError: true,
	}
}

// WaitTool returns the `wait` tool, which blocks until one subagent settles.
func (s *Session) WaitTool() agent.AgentTool {
	binding := &subagentToolBinding{}
	binding.set(s)
	return &subagentWaitTool{binding: binding}
}

type subagentWaitTool struct{ binding *subagentToolBinding }

// session returns the session this tool drives.
func (t *subagentWaitTool) session() *Session { return t.binding.get() }

// Name is the tool id.
func (t *subagentWaitTool) Name() string { return "wait" }

// Label is the display title.
func (t *subagentWaitTool) Label() string { return "Wait" }

// ExecutionMode is sequential: waiting is a scheduling point, not a batch.
func (t *subagentWaitTool) ExecutionMode() agent.ToolExecutionMode {
	return agent.ToolModeSequential
}

// Schema is the model-facing contract.
func (t *subagentWaitTool) Schema() ai.ToolSchema {
	return ai.ToolSchema{
		Name:        "wait",
		Description: "Block until one background subagent settles, and return its result. Use it when you are completely blocked; do not poll.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"timeout": map[string]any{
					"type":        "number",
					"description": "Seconds to wait before giving up. Default 300, maximum 1800.",
				},
			},
		},
	}
}

// Execute blocks for one settlement.
func (t *subagentWaitTool) Execute(ctx context.Context, toolCallID string, params json.RawMessage, onUpdate agent.ToolUpdateCallback) (agent.AgentToolResult, error) {
	runtime := subagentRuntimeOf(t)
	if runtime == nil {
		return toolError("subagents are unavailable in this session"), nil
	}
	var parsed struct {
		Timeout float64 `json:"timeout"`
	}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &parsed); err != nil {
			return toolError("invalid wait arguments: " + err.Error()), nil
		}
	}
	timeout := 300 * time.Second
	if parsed.Timeout > 0 {
		timeout = time.Duration(parsed.Timeout * float64(time.Second))
	}
	if timeout > waitTimeoutCap {
		timeout = waitTimeoutCap
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	settled, err := runtime.Jobs().Wait(waitCtx)
	if err != nil {
		if errors.Is(err, subagent.ErrNoJobs) {
			return agent.AgentToolResult{
				Content: []ai.ToolResultMessageContent{ai.TextContent{Text: "Nothing to wait for."}},
			}, nil
		}
		return toolError("wait: " + err.Error()), nil
	}
	entry, ok := runtime.Registry().Get(settled.Job.ID)
	status := "finished"
	failure := settled.Err
	if ok {
		switch entry.Status {
		case subagent.StatusAborted:
			status = "was aborted"
		case subagent.StatusIdle:
			status = "finished"
		}
		if entry.Error != "" {
			status = "failed: " + entry.Error
			if failure == nil {
				failure = errors.New(entry.Error)
			}
		}
	}
	text := "Agent `" + settled.Job.ID + "` " + status + ".\n"
	if settled.Result != "" {
		text += settled.Result
	}
	return agent.AgentToolResult{
		Content: []ai.ToolResultMessageContent{ai.TextContent{Text: text}},
		Details: settled.Job,
		IsError: failure != nil,
	}, nil
}

// AgentsTool returns the `agents` tool: list, read, message, revive or kill a
// subagent. It is the model-facing form of the Agent Hub's actions.
func (s *Session) AgentsTool() agent.AgentTool {
	binding := &subagentToolBinding{}
	binding.set(s)
	return &subagentAgentsTool{binding: binding}
}

type subagentAgentsTool struct{ binding *subagentToolBinding }

// session returns the session this tool drives.
func (t *subagentAgentsTool) session() *Session { return t.binding.get() }

// Name is the tool id.
func (t *subagentAgentsTool) Name() string { return "agents" }

// Label is the display title.
func (t *subagentAgentsTool) Label() string { return "Agents" }

// ExecutionMode is sequential: a revive or kill mutates registry state.
func (t *subagentAgentsTool) ExecutionMode() agent.ToolExecutionMode {
	return agent.ToolModeSequential
}

// Schema is the model-facing contract.
func (t *subagentAgentsTool) Schema() ai.ToolSchema {
	return ai.ToolSchema{
		Name: "agents",
		Description: "Inspect and control subagents: list them, read one's output, message it, " +
			"revive a parked or killed one, or kill a running one.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"enum":        []string{"list", "read", "message", "revive", "kill", "jobs"},
					"description": "What to do. Default: list.",
				},
				"id": map[string]any{
					"type":        "string",
					"description": "Agent id for read, message, revive and kill.",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "Text to deliver for the message action.",
				},
			},
			"required": []string{"action"},
		},
	}
}

// Execute performs one control action.
func (t *subagentAgentsTool) Execute(ctx context.Context, toolCallID string, params json.RawMessage, onUpdate agent.ToolUpdateCallback) (agent.AgentToolResult, error) {
	runtime := subagentRuntimeOf(t)
	if runtime == nil {
		return toolError("subagents are unavailable in this session"), nil
	}
	var parsed struct {
		Action  string `json:"action"`
		ID      string `json:"id"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(params, &parsed); err != nil {
		return toolError("invalid agents arguments: " + err.Error()), nil
	}
	if parsed.Action == "" {
		parsed.Action = "list"
	}
	switch parsed.Action {
	case "list":
		return agent.AgentToolResult{
			Content: []ai.ToolResultMessageContent{ai.TextContent{Text: renderAgentList(runtime.Registry().Sorted())}},
			Details: runtime.Registry().List(),
		}, nil
	case "jobs":
		return agent.AgentToolResult{
			Content: []ai.ToolResultMessageContent{ai.TextContent{Text: renderJobs(runtime.Jobs().List())}},
			Details: runtime.Jobs().List(),
		}, nil
	case "read":
		text, err := runtime.ReadAgentOutput(parsed.ID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return agent.AgentToolResult{Content: []ai.ToolResultMessageContent{ai.TextContent{Text: text}}}, nil
	case "message":
		if strings.TrimSpace(parsed.Message) == "" {
			return toolError("message action needs `message`"), nil
		}
		if err := runtime.Steer(ctx, parsed.ID, parsed.Message); err != nil {
			return toolError(err.Error()), nil
		}
		return agent.AgentToolResult{
			Content: []ai.ToolResultMessageContent{ai.TextContent{Text: "Message delivered to `" + parsed.ID + "`."}},
		}, nil
	case "revive":
		if err := runtime.Revive(ctx, parsed.ID); err != nil {
			return toolError(err.Error()), nil
		}
		return agent.AgentToolResult{
			Content: []ai.ToolResultMessageContent{ai.TextContent{Text: "Revived `" + parsed.ID + "`."}},
		}, nil
	case "kill":
		if err := runtime.Kill(ctx, parsed.ID); err != nil {
			return toolError(err.Error()), nil
		}
		return agent.AgentToolResult{
			Content: []ai.ToolResultMessageContent{ai.TextContent{Text: "Killed `" + parsed.ID + "`."}},
		}, nil
	default:
		return toolError("unknown action " + parsed.Action), nil
	}
}

// renderAgentList formats the roster for the model.
func renderAgentList(entries []subagent.Entry) string {
	if len(entries) == 0 {
		return "No subagents."
	}
	var b strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&b, "- `%s` (%s) %s", entry.ID, entry.AgentType, entry.Status)
		if entry.Running() && entry.CurrentTool != "" {
			fmt.Fprintf(&b, ", tool %s", entry.CurrentTool)
		}
		if entry.Usage.Total > 0 {
			fmt.Fprintf(&b, ", %d tokens", entry.Usage.Total)
		}
		if entry.Error != "" {
			fmt.Fprintf(&b, ", error: %s", entry.Error)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// renderJobs formats the job table for the model.
func renderJobs(jobs []subagent.Job) string {
	if len(jobs) == 0 {
		return "No background jobs."
	}
	sorted := append([]subagent.Job(nil), jobs...)
	sort.SliceStable(sorted, func(a, b int) bool { return sorted[a].StartedAt.After(sorted[b].StartedAt) })
	var b strings.Builder
	now := time.Now()
	for _, job := range sorted {
		fmt.Fprintf(&b, "- `%s` (%s) %s, %s\n", job.ID, job.Type, job.State, job.Elapsed(now).Round(time.Second))
	}
	return b.String()
}

// subagentToolsFor returns the subagent tools a new session registers. The
// registry is built before the Session exists, so the tools resolve their
// session through the binding the constructor fills in afterwards.
func subagentToolsFor(opts SessionOptions) []agent.AgentTool {
	if opts.NoTools == "all" {
		return nil
	}
	binding := opts.subagentBinding
	if binding == nil {
		return nil
	}
	return []agent.AgentTool{
		&subagentTaskTool{binding: binding},
		&subagentWaitTool{binding: binding},
		&subagentAgentsTool{binding: binding},
	}
}

// subagentToolBinding holds the Session the subagent tools act on. The tools
// are registered while the Session is still being constructed, so the binding
// is filled in before the first turn.
type subagentToolBinding struct {
	session atomic.Pointer[Session]
}

// set records the session the tools drive.
func (b *subagentToolBinding) set(session *Session) {
	if b != nil {
		b.session.Store(session)
	}
}

// get returns the bound session, or nil before construction completes.
func (b *subagentToolBinding) get() *Session {
	if b == nil {
		return nil
	}
	return b.session.Load()
}

// session returns the bound session for a tool, or nil.
func (b *subagentToolBinding) sessionOrNil() *Session { return b.get() }

// subagentToolNames are the tools the subagent runtime contributes to the
// default built-in selection.
var subagentToolNames = []string{"task", "wait", "agents"}
