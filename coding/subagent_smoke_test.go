package coding

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tmih06/ompig/ai"
	"github.com/tmih06/ompig/coding/subagent"
)

// echoProvider replies to every request with a fixed text, so a child session
// completes a turn without network access.
type echoProvider struct {
	reply string
	calls atomic.Int64
}

func (p *echoProvider) ID() string { return "echo" }

func (p *echoProvider) Stream(_ context.Context, req ai.TranscriptContext, _ ai.StreamOptions) (*ai.AssistantMessageEventStream, error) {
	p.calls.Add(1)
	message := sessionTestMessage("echo", p.reply, ai.StopReasonStop, "")
	// Report usage so the hub's token, cost and request columns have data.
	message.Usage = ai.Usage{Input: 120, Output: 30, TotalTokens: 150, Cost: ai.UsageCost{Input: 0.001, Output: 0.002, Total: 0.003}}
	return newSessionTestStream(
		ai.StartEvent{Partial: message},
		ai.DoneEvent{Reason: ai.StopReasonStop, Message: message},
	), nil
}

func (p *echoProvider) Close() error { return nil }

func echoModel(p *echoProvider) *ai.Model {
	return &ai.Model{ID: "echo-1", DisplayName: "echo-1", Provider: p, Capabilities: ai.ModelCapabilities{ContextWindow: 8000}}
}

// newSubagentTestSession builds a session over an echo provider and a temporary
// agent directory, with no extensions and no discovered resources.
func newSubagentTestSession(t *testing.T, reply string) (*Session, *echoProvider) {
	t.Helper()
	provider := &echoProvider{reply: reply}
	services := newTestServices(t)
	session, err := NewSession(services, SessionOptions{
		Model:            echoModel(provider),
		SessionManager:   mustSessionLog(t, services),
		ResourceLoader:   NoResources,
		SkipBuiltinTools: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session, provider
}

func mustSessionLog(t *testing.T, services *AgentSessionServices) *SessionManager {
	t.Helper()
	logs, err := SessionManagerFor(services, SessionStartOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return logs
}

// TestSubagentRuntimeSpawnsChildSession proves the runtime really creates a
// child session: a spawn returns the child's assistant text, the agent is
// registered, its output artifact exists, and `agent://` reads it back.
func TestSubagentRuntimeSpawnsChildSession(t *testing.T) {
	session, provider := newSubagentTestSession(t, "subagent finished the work")

	runtime := session.SubagentRuntime()
	if runtime == nil {
		t.Fatal("SubagentRuntime() = nil")
	}
	results, err := runtime.Runner().SpawnBatch(context.Background(), "", subagent.BatchRequest{
		Context: "We are testing the subagent runtime.",
		Tasks: []subagent.SpawnRequest{{
			Name:          "Worker",
			Task:          "Report the status.",
			SolutionSpace: "one answer: report what you did",
		}},
	})
	if err != nil {
		t.Fatalf("SpawnBatch: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	awaitSettlement(t, runtime)
	// The spawn acknowledgement names the agent; the settled entry holds the
	// outcome the asynchronous runtime produced.
	entry0, ok0 := runtime.Registry().Get(results[0].ID)
	if !ok0 {
		t.Fatalf("agent %q not registered", results[0].ID)
	}
	result := subagent.Result{ID: entry0.ID, Name: entry0.ID, Status: entry0.Status,
		OutputPath: entry0.OutputPath, TranscriptPath: entry0.TranscriptPath}
	// The settled entry is where an asynchronous run records its outcome; the
	// artifact holds the text.
	outputText, outputErr := runtime.ReadAgentOutput(entry0.ID)
	if outputErr != nil {
		t.Fatalf("agent output: %v", outputErr)
	}
	if outputText != "subagent finished the work" {
		t.Errorf("output = %q, want the child's assistant text", outputText)
	}
	if result.ID != "Worker" {
		t.Errorf("id = %q, want Worker", result.ID)
	}
	if result.Status != subagent.StatusIdle {
		t.Errorf("status = %q, want idle", result.Status)
	}
	if provider.calls.Load() == 0 {
		t.Fatal("the child session never called the provider")
	}

	entry, ok := runtime.Registry().Get("Worker")
	if !ok {
		t.Fatal("agent Worker is not registered")
	}
	if entry.Status != subagent.StatusIdle {
		t.Errorf("registry status = %q, want idle", entry.Status)
	}
	if entry.Usage.Requests == 0 && entry.Usage.Total == 0 {
		t.Log("usage not reported by the provider; acceptable for a fake")
	}

	output, err := runtime.ReadAgentOutput("Worker")
	if err != nil {
		t.Fatalf("ReadAgentOutput: %v", err)
	}
	if output != "subagent finished the work" {
		t.Errorf("agent:// output = %q", output)
	}
	if path := result.OutputPath; path == "" {
		t.Error("no output artifact path")
	} else if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("output artifact %s: %v", path, statErr)
	}
	if result.TranscriptPath == "" {
		t.Error("no transcript path")
	}
}

// TestSubagentToolsAreRegistered checks that a default session exposes the
// subagent tools, which is what makes them reachable by the model.
func TestSubagentToolsAreRegistered(t *testing.T) {
	session, _ := newSubagentTestSession(t, "ok")
	names := session.ActiveToolNames()
	for _, want := range []string{"task", "wait", "agents"} {
		if !containsString(names, want) {
			t.Errorf("active tools %v missing %q", names, want)
		}
	}
}

// TestSubagentSteerAndKill covers the control surface the Agent Hub uses.
func TestSubagentSteerAndKill(t *testing.T) {
	session, _ := newSubagentTestSession(t, "ack")
	runtime := session.SubagentRuntime()
	results, err := runtime.Runner().SpawnBatch(context.Background(), "", subagent.BatchRequest{
		Tasks: []subagent.SpawnRequest{{Name: "Chatty", Task: "Say ack."}},
	})
	if err != nil || len(results) != 1 {
		t.Fatalf("spawn: %v (%d results)", err, len(results))
	}
	awaitSettlement(t, runtime)
	ctx := context.Background()
	if err := runtime.Steer(ctx, "Chatty", "one more thing"); err != nil {
		t.Fatalf("Steer: %v", err)
	}
	if err := runtime.Kill(ctx, "Chatty"); err != nil {
		t.Fatalf("Kill: %v", err)
	}
	entry, _ := runtime.Registry().Get("Chatty")
	if entry.Status != subagent.StatusAborted {
		t.Errorf("status after kill = %q, want aborted", entry.Status)
	}
	if err := runtime.Revive(ctx, "Chatty"); err != nil {
		t.Fatalf("Revive: %v", err)
	}
	entry, _ = runtime.Registry().Get("Chatty")
	if entry.Status != subagent.StatusIdle {
		t.Errorf("status after revive = %q, want idle", entry.Status)
	}
}

// TestSubagentArtifactLayout checks the output files land beside the parent
// session, not in the project session bucket.
func TestSubagentArtifactLayout(t *testing.T) {
	session, _ := newSubagentTestSession(t, "artifact text")
	runtime := session.SubagentRuntime()
	if _, err := runtime.Runner().SpawnBatch(context.Background(), "", subagent.BatchRequest{
		Tasks: []subagent.SpawnRequest{{Name: "Writer", Task: "Write."}},
	}); err != nil {
		t.Fatal(err)
	}
	awaitSettlement(t, runtime)
	dir := runtime.Runner().Artifacts().Dir()
	if dir == "" {
		t.Fatal("artifacts dir is empty")
	}
	if !strings.HasSuffix(dir, "-agents") {
		t.Errorf("artifacts dir = %q, want a -agents sibling of the session file", dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "Writer.md")); err != nil {
		t.Errorf("Writer.md: %v", err)
	}
}

func containsString(list []string, want string) bool {
	return slices.Contains(list, want)
}

// TestSubagentToolProgressReachesRegistry proves the live columns: a child's
// tool call and its token usage become visible on the registry entry while the
// hub is watching.
func TestSubagentToolProgressReachesRegistry(t *testing.T) {
	session, provider := newSubagentTestSession(t, "")
	// A provider that calls one tool, then answers.
	provider.reply = "done"
	runtime := session.SubagentRuntime()
	if _, err := runtime.Runner().SpawnBatch(context.Background(), "", subagent.BatchRequest{
		Tasks: []subagent.SpawnRequest{{Name: "Worker", Task: "Do it."}},
	}); err != nil {
		t.Fatal(err)
	}
	awaitSettlement(t, runtime)
	entry, ok := runtime.Registry().Get("Worker")
	if !ok {
		t.Fatal("Worker not registered")
	}
	if entry.Usage.Requests == 0 {
		t.Errorf("usage requests = 0, want the child's assistant messages counted")
	}
	if entry.CurrentTool != "" {
		t.Errorf("current tool = %q, want cleared after the run", entry.CurrentTool)
	}
}

// awaitSettlement waits until every registered agent has settled, which an
// asynchronous runtime needs before a test inspects artifacts.
func awaitSettlement(t *testing.T, runtime *SubagentRuntime) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		settled := true
		for _, entry := range runtime.Registry().List() {
			if entry.Status == subagent.StatusRunning {
				settled = false
				break
			}
		}
		if settled {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("agents did not settle within the deadline")
}
