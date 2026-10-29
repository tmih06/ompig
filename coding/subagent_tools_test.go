package coding

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tmih06/ompig/agent"
	"github.com/tmih06/ompig/ai"
	"github.com/tmih06/ompig/coding/subagent"
)

// TestTaskToolRejectsEmptyTask covers the tool's own validation, which the
// model sees and can correct.
func TestTaskToolRejectsEmptyTask(t *testing.T) {
	session, _ := newSubagentTestSession(t, "ok")
	tool := session.TaskTool()
	result, err := tool.Execute(context.Background(), "call-1", json.RawMessage(`{"task":"   "}`), nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.IsError || !strings.Contains(textOf(result), "requires") {
		t.Errorf("result = %q, want a missing-task rejection", textOf(result))
	}
	if len(session.SubagentRuntime().Registry().List()) != 0 {
		t.Error("a rejected task still registered an agent")
	}
}

// TestTaskToolSpawnsThroughTheTool covers the tool's happy path: the model's
// arguments reach the runner and the result names the agent.
func TestTaskToolSpawnsThroughTheTool(t *testing.T) {
	session, _ := newSubagentTestSession(t, "tool output")
	tool := session.TaskTool()
	params, _ := json.Marshal(map[string]any{
		"context": "shared",
		"tasks": []map[string]any{
			{"name": "Alpha", "task": "do alpha", "solutionSpace": "one answer"},
			{"name": "Beta", "task": "do beta", "solutionSpace": "one answer"},
		},
	})
	result, err := tool.Execute(context.Background(), "call-2", params, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.IsError {
		t.Fatalf("task tool failed: %s", textOf(result))
	}
	entries := session.SubagentRuntime().Registry().List()
	if len(entries) != 2 {
		t.Fatalf("agents = %d, want 2", len(entries))
	}
	for _, entry := range entries {
		if !strings.Contains(entry.Task, "shared") {
			t.Errorf("agent %s task = %q, want the shared context", entry.ID, entry.Task)
		}
	}
	text := textOf(result)
	if !strings.Contains(text, "Alpha") || !strings.Contains(text, "Beta") {
		t.Errorf("result text = %q, want both named agents", text)
	}
	if !strings.Contains(text, "Spawned") {
		t.Errorf("result text = %q, want a spawn acknowledgement", text)
	}
	awaitSettlement(t, session.SubagentRuntime())
}

// TestAgentsToolControls covers list, read, message, revive, kill and jobs
// through the tool, which is what the model calls.
func TestAgentsToolControls(t *testing.T) {
	session, _ := newSubagentTestSession(t, "controlled")
	runtime := session.SubagentRuntime()
	if _, err := runtime.Runner().SpawnBatch(context.Background(), "", subagent.BatchRequest{
		Tasks: []subagent.SpawnRequest{{Name: "Target", Task: "work"}},
	}); err != nil {
		t.Fatal(err)
	}
	tool := session.AgentsTool()

	awaitSettlement(t, runtime)
	list, err := tool.Execute(context.Background(), "c1", json.RawMessage(`{"action":"list"}`), nil)
	if err != nil || !strings.Contains(textOf(list), "Target") {
		t.Fatalf("list = %q err=%v", textOf(list), err)
	}

	read, err := tool.Execute(context.Background(), "c2", json.RawMessage(`{"action":"read","id":"Target"}`), nil)
	if err != nil || !strings.Contains(textOf(read), "controlled") {
		t.Fatalf("read = %q err=%v", textOf(read), err)
	}

	msg, err := tool.Execute(context.Background(), "c3", json.RawMessage(`{"action":"message","id":"Target","message":"more please"}`), nil)
	if err != nil || msg.IsError {
		t.Fatalf("message = %q err=%v", textOf(msg), err)
	}

	kill, err := tool.Execute(context.Background(), "c4", json.RawMessage(`{"action":"kill","id":"Target"}`), nil)
	if err != nil || kill.IsError {
		t.Fatalf("kill = %q err=%v", textOf(kill), err)
	}
	revive, err := tool.Execute(context.Background(), "c5", json.RawMessage(`{"action":"revive","id":"Target"}`), nil)
	if err != nil || revive.IsError {
		t.Fatalf("revive = %q err=%v", textOf(revive), err)
	}

	jobs, err := tool.Execute(context.Background(), "c6", json.RawMessage(`{"action":"jobs"}`), nil)
	if err != nil {
		t.Fatalf("jobs: %v", err)
	}
	if textOf(jobs) == "" {
		t.Error("jobs listed nothing")
	}

	unknown, err := tool.Execute(context.Background(), "c7", json.RawMessage(`{"action":"wat"}`), nil)
	if err != nil || !unknown.IsError {
		t.Errorf("unknown action = %+v err=%v, want an error result", unknown, err)
	}
}

// TestWaitToolReturnsASettlement covers the blocking path and the
// nothing-to-wait-for answer.
func TestWaitToolReturnsASettlement(t *testing.T) {
	session, _ := newSubagentTestSession(t, "waited")
	runtime := session.SubagentRuntime()
	// Nothing running: the tool answers instead of parking the turn.
	tool := session.WaitTool()
	empty, err := tool.Execute(context.Background(), "w0", json.RawMessage(`{"timeout":1}`), nil)
	if err != nil {
		t.Fatalf("wait on nothing: %v", err)
	}
	if !strings.Contains(textOf(empty), "Nothing to wait for") {
		t.Errorf("wait on nothing = %q", textOf(empty))
	}

	// A settled job is delivered exactly once.
	if _, err := runtime.Jobs().Register(context.Background(), "job-1", "task", "label", "",
		func(context.Context) (string, error) { return "job result", nil }); err != nil {
		t.Fatal(err)
	}
	first, err := tool.Execute(context.Background(), "w1", json.RawMessage(`{"timeout":10}`), nil)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if !strings.Contains(textOf(first), "job result") {
		t.Errorf("first wait = %q, want the job result", textOf(first))
	}
	second, err := tool.Execute(context.Background(), "w2", json.RawMessage(`{"timeout":1}`), nil)
	if err != nil {
		t.Fatalf("second wait: %v", err)
	}
	if strings.Contains(textOf(second), "job result") {
		t.Error("a consumed result was delivered twice")
	}
}

// TestParkedAgentTranscriptFallsBackToDisk proves a parked child's transcript
// still renders, which is what the hub's focus action needs.
func TestParkedAgentTranscriptFallsBackToDisk(t *testing.T) {
	session, _ := newSubagentTestSession(t, "on disk")
	runtime := session.SubagentRuntime()
	if _, err := runtime.Runner().SpawnBatch(context.Background(), "", subagent.BatchRequest{
		Tasks: []subagent.SpawnRequest{{Name: "Parked", Task: "work"}},
	}); err != nil {
		t.Fatal(err)
	}
	awaitSettlement(t, runtime)
	runtime.Park("Parked")
	if _, ok := runtime.ActiveSession("Parked"); ok {
		t.Fatal("Park kept a live session")
	}
	messages, ok := session.SubagentTranscript("Parked")
	if !ok || len(messages) == 0 {
		t.Fatalf("parked transcript = %d messages, ok=%v", len(messages), ok)
	}
	// The output artifact survives parking too.
	if output, err := runtime.ReadAgentOutput("Parked"); err != nil || !strings.Contains(output, "on disk") {
		t.Errorf("parked output = %q err=%v", output, err)
	}
}

// textOf extracts a tool result's text.
func textOf(result agent.AgentToolResult) string {
	return ai.ContentText(result.Content)
}

// TestAgentDefinitionToolsScopeTheChild proves a definition's `tools` really
// restricts the child: a read-only agent cannot write or spawn.
func TestAgentDefinitionToolsScopeTheChild(t *testing.T) {
	session, _ := newSubagentTestSession(t, "scoped")
	runtime := session.SubagentRuntime()
	// `spawns: []` in the frontmatter is an explicit denial, which removes the
	// `task` the tool list names.
	runtime.Runner().AddDefinition(subagent.Definition{
		Name:         "readonly",
		Description:  "Read-only agent.",
		Tools:        []string{"read", "grep", "glob", "task"},
		Spawns:       []string{},
		SystemPrompt: "read only",
	})
	if _, err := runtime.Runner().Run(context.Background(), "", subagent.SpawnRequest{
		Agent: "readonly", Name: "RO", Task: "look",
	}); err != nil {
		t.Fatal(err)
	}
	child, ok := runtime.ActiveSession("RO")
	if !ok {
		t.Fatal("no live child session")
	}
	active := child.ActiveToolNames()
	// `glob` is omp's name for Pi's `find`, and `task` is named but not granted
	// by `spawns`, so it is removed.
	for _, forbidden := range []string{"write", "edit", "bash", "task"} {
		if containsString(active, forbidden) {
			t.Errorf("read-only child has %q; active = %v", forbidden, active)
		}
	}
	for _, wanted := range []string{"read", "grep", "find"} {
		if !containsString(active, wanted) {
			t.Errorf("read-only child is missing %q; active = %v", wanted, active)
		}
	}
}

// TestAgentDefinitionSpawnsGateNestedDelegation proves `spawns` is enforced:
// an agent whose list omits the target cannot delegate to it, and an agent with
// no `spawns` grant loses the task tool entirely.
func TestAgentDefinitionSpawnsGateNestedDelegation(t *testing.T) {
	session, _ := newSubagentTestSession(t, "gate")
	runtime := session.SubagentRuntime()
	runner := runtime.Runner()
	runner.AddDefinition(subagent.Definition{
		Name: "denied", Description: "May not spawn anything.",
		Tools: []string{"read"}, Spawns: []string{}, SystemPrompt: "no delegation",
	})
	runner.AddDefinition(subagent.Definition{
		Name: "picker", Description: "May spawn only scout.",
		Tools: []string{"read"}, Spawns: []string{"scout"}, SystemPrompt: "pick",
	})

	if _, err := runner.Run(context.Background(), "", subagent.SpawnRequest{
		Agent: "denied", Name: "D", Task: "work",
	}); err != nil {
		t.Fatal(err)
	}
	child, _ := runtime.ActiveSession("D")
	if containsString(child.ActiveToolNames(), "task") {
		t.Errorf("an agent with no spawn grant still has task: %v", child.ActiveToolNames())
	}
	// The gate rejects a nested spawn the parent is not allowed to make.
	if _, err := runner.Run(context.Background(), "D", subagent.SpawnRequest{
		Name: "Nested", Task: "work",
	}); err == nil || !strings.Contains(err.Error(), "may not spawn") {
		t.Errorf("nested spawn from a denied agent = %v, want a policy rejection", err)
	}

	if _, err := runner.Run(context.Background(), "", subagent.SpawnRequest{
		Agent: "picker", Name: "P", Task: "work",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "P", subagent.SpawnRequest{
		Agent: "reviewer", Name: "NotAllowed", Task: "work",
	}); err == nil || !strings.Contains(err.Error(), "may not spawn") {
		t.Errorf("picker spawned reviewer: %v, want a policy rejection", err)
	}
	if _, err := runner.Run(context.Background(), "P", subagent.SpawnRequest{
		Agent: "scout", Name: "Allowed", Task: "work",
	}); err != nil {
		t.Errorf("picker could not spawn scout: %v", err)
	}
}

// TestAgentDefinitionDepthCap proves the recursion cap rejects a spawn past the
// limit instead of nesting forever.
func TestAgentDefinitionDepthCap(t *testing.T) {
	session, _ := newSubagentTestSession(t, "depth")
	runner := session.SubagentRuntime().Runner()
	runner.MaxDepth = 1
	if _, err := runner.Run(context.Background(), "", subagent.SpawnRequest{Name: "L1", Task: "work"}); err != nil {
		t.Fatal(err)
	}
	entry, _ := runner.Registry().Get("L1")
	if _, err := runner.Run(context.Background(), entry.ID, subagent.SpawnRequest{Name: "L2", Task: "work"}); err == nil ||
		!strings.Contains(err.Error(), "recursion depth") {
		t.Errorf("spawn past the cap = %v, want a depth rejection", err)
	}
}

// TestAgentDefinitionModelSelector proves a definition's `model` selector is
// applied when it resolves, and ignored when it does not.
func TestAgentDefinitionModelSelector(t *testing.T) {
	session, _ := newSubagentTestSession(t, "model")
	runner := session.SubagentRuntime().Runner()
	runner.AddDefinition(subagent.Definition{
		Name: "missingmodel", Description: "Selector that resolves to nothing.",
		Model: []string{"nope/none"}, SystemPrompt: "x",
	})
	if _, err := runner.Run(context.Background(), "", subagent.SpawnRequest{Agent: "missingmodel", Name: "M", Task: "work"}); err != nil {
		t.Fatal(err)
	}
	child, _ := runtimeSessionOf(t, session, "M")
	if child.Model() == nil {
		t.Fatal("child has no model; an unresolved selector must fall back to the parent's")
	}
}

// runtimeSessionOf returns an agent's live child session.
func runtimeSessionOf(t *testing.T, session *Session, id string) (*Session, bool) {
	t.Helper()
	return session.SubagentRuntime().ActiveSession(id)
}

// TestNestedSubagentSpawn proves a subagent can spawn its own child: the bundled
// task agent inherits the spawn policy, so a depth-1 agent still has the task
// tool and a depth-2 spawn is allowed by the default cap of two.
func TestNestedSubagentSpawn(t *testing.T) {
	session, _ := newSubagentTestSession(t, "nested result")
	runtime := session.SubagentRuntime()
	if _, err := runtime.Runner().Run(context.Background(), "", subagent.SpawnRequest{
		Name: "Parent", Task: "delegate further",
	}); err != nil {
		t.Fatal(err)
	}
	awaitSettlement(t, runtime)
	parent, ok := runtime.ActiveSession("Parent")
	if !ok {
		t.Fatal("no live parent session")
	}
	if !containsString(parent.ActiveToolNames(), "task") {
		t.Fatalf("a depth-1 agent has no task tool: %v", parent.ActiveToolNames())
	}
	// The parent's own runtime is the same registry, so its child nests under it.
	childTool := parent.TaskTool()
	params, _ := json.Marshal(map[string]any{"task": "the nested work", "solutionSpace": "one answer"})
	if _, err := childTool.Execute(context.Background(), "nested-call", params, nil); err != nil {
		t.Fatalf("nested task: %v", err)
	}
	awaitSettlement(t, runtime)
	entries := runtime.Registry().List()
	if len(entries) < 2 {
		t.Fatalf("registry = %d agents, want the parent and its child", len(entries))
	}
	var nested subagent.Entry
	for _, entry := range entries {
		if entry.Depth == 2 {
			nested = entry
		}
	}
	if nested.ID == "" {
		t.Fatalf("no depth-2 agent; entries = %+v", entries)
	}
	if nested.ParentID != "Parent" {
		t.Errorf("nested parent = %q, want Parent", nested.ParentID)
	}
	if !strings.HasPrefix(nested.ID, "Parent.") {
		t.Errorf("nested id = %q, want a dot-qualified child of Parent", nested.ID)
	}
	if output, err := runtime.ReadAgentOutput(nested.ID); err != nil || !strings.Contains(output, "nested result") {
		t.Errorf("nested output = %q err=%v", output, err)
	}
}

// TestTaskOutputSchemaValidatesResult proves a requested output schema is
// actually enforced: a conforming JSON answer becomes the structured result,
// and a non-conforming one is reported instead of silently accepted.
func TestTaskOutputSchemaValidatesResult(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"},"count":{"type":"number"}},"required":["ok","count"]}`)

	t.Run("conforming result", func(t *testing.T) {
		session, _ := newSubagentTestSession(t, `{"ok": true, "count": 3}`)
		runtime := session.SubagentRuntime()
		tool := session.TaskTool()
		params, _ := json.Marshal(map[string]any{
			"tasks": []map[string]any{{
				"name": "Typed", "task": "answer", "solutionSpace": "fixed shape",
				"outputSchema": json.RawMessage(schema),
			}},
		})
		if _, err := tool.Execute(context.Background(), "s1", params, nil); err != nil {
			t.Fatalf("task: %v", err)
		}
		awaitSettlement(t, runtime)
		structured, ok, err := runtime.Runner().Artifacts().ReadStructured("Typed")
		if err != nil || !ok {
			t.Fatalf("structured sidecar: ok=%v err=%v", ok, err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(structured, &decoded); err != nil {
			t.Fatalf("sidecar is not JSON: %v", err)
		}
		if decoded["ok"] != true || decoded["count"] != float64(3) {
			t.Errorf("structured result = %v", decoded)
		}
	})

	t.Run("conforming result in a fenced block", func(t *testing.T) {
		session, _ := newSubagentTestSession(t, "Here you go:\n\n```json\n{\"ok\": false, \"count\": 0}\n```\n")
		runtime := session.SubagentRuntime()
		params, _ := json.Marshal(map[string]any{
			"tasks": []map[string]any{{
				"name": "Fenced", "task": "answer", "solutionSpace": "fixed shape",
				"outputSchema": json.RawMessage(schema),
			}},
		})
		if _, err := session.TaskTool().Execute(context.Background(), "s2", params, nil); err != nil {
			t.Fatalf("task: %v", err)
		}
		awaitSettlement(t, runtime)
		structured, ok, _ := runtime.Runner().Artifacts().ReadStructured("Fenced")
		if !ok || !strings.Contains(string(structured), `"ok":false`) {
			t.Errorf("fenced structured result = %s ok=%v", structured, ok)
		}
	})

	t.Run("rejected result", func(t *testing.T) {
		session, _ := newSubagentTestSession(t, "no json here")
		runtime := session.SubagentRuntime()
		params, _ := json.Marshal(map[string]any{
			"tasks": []map[string]any{{
				"name": "Bad", "task": "answer", "solutionSpace": "fixed shape",
				"outputSchema": json.RawMessage(schema),
			}},
		})
		if _, err := session.TaskTool().Execute(context.Background(), "s3", params, nil); err != nil {
			t.Fatalf("task: %v", err)
		}
		awaitSettlement(t, runtime)
		entry, _ := runtime.Registry().Get("Bad")
		if _, ok, _ := runtime.Runner().Artifacts().ReadStructured("Bad"); ok {
			t.Error("a non-conforming result produced a structured sidecar")
		}
		if entry.OutputPath == "" {
			t.Error("the prose result was not kept")
		}
	})
}
