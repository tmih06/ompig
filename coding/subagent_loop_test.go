package coding

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/tmih06/ompig/ai"
	"github.com/tmih06/ompig/coding/subagent"
)

// delegatingProvider answers the parent's first request with a `task` tool call
// and the child's request with plain text. It is how the test proves the model
// reaches the subagent runtime through the real agent loop.
type delegatingProvider struct {
	taskArgs   string
	childReply string
	calls      atomic.Int64
}

func (p *delegatingProvider) ID() string { return "scripted" }

func (p *delegatingProvider) Stream(_ context.Context, req ai.TranscriptContext, _ ai.StreamOptions) (*ai.AssistantMessageEventStream, error) {
	call := p.calls.Add(1)
	// A child session runs the same provider; its request carries the agent
	// definition's prompt rather than the parent's.
	if isDelegatedChildRequest(req) {
		message := sessionTestMessage("scripted", p.childReply, ai.StopReasonStop, "")
		return newSessionTestStream(
			ai.StartEvent{Partial: message},
			ai.DoneEvent{Reason: ai.StopReasonStop, Message: message},
		), nil
	}
	if call == 1 {
		message := sessionTestMessage("scripted", "", ai.StopReasonToolUse, "")
		message.Content = []ai.AssistantContentBlock{ai.ToolCall{
			ID:        "call-1",
			Name:      "task",
			Arguments: ai.JsonObject{"task": p.taskArgs, "solutionSpace": "one answer"},
		}}
		return newSessionTestStream(
			ai.StartEvent{Partial: message},
			ai.DoneEvent{Reason: ai.StopReasonToolUse, Message: message},
		), nil
	}
	message := sessionTestMessage("scripted", "parent done", ai.StopReasonStop, "")
	return newSessionTestStream(
		ai.StartEvent{Partial: message},
		ai.DoneEvent{Reason: ai.StopReasonStop, Message: message},
	), nil
}

func (p *delegatingProvider) Close() error { return nil }

// isDelegatedChildRequest reports whether a request is a subagent's: the child
// runs the general task prompt, which the parent does not carry.
func isDelegatedChildRequest(req ai.TranscriptContext) bool {
	return strings.Contains(ai.GetCurrentSystemPrompt(req.Messages()), "general-purpose subagent")
}

func delegatingModel(p *delegatingProvider) *ai.Model {
	return &ai.Model{ID: "scripted-1", DisplayName: "scripted-1", Provider: p, Capabilities: ai.ModelCapabilities{ContextWindow: 32000}}
}

// TestSubagentSpawnThroughAgentLoop is the end-to-end proof: the parent model
// calls `task`, the tool runs a real child session, and the parent's next turn
// sees the child's output.
func TestSubagentSpawnThroughAgentLoop(t *testing.T) {
	provider := &delegatingProvider{taskArgs: "Report the repo status.", childReply: "child says: all green"}
	services := newTestServices(t)
	session, err := NewSession(services, SessionOptions{
		Model:          delegatingModel(provider),
		SessionManager: mustSessionLog(t, services),
		ResourceLoader: NoResources,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	messages, err := session.Send(context.Background(), "Delegate this work.")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	// The runtime is asynchronous, so the parent's tool result acknowledges the
	// spawn; the child's answer arrives through `wait`.
	runtime := session.SubagentRuntime()
	awaitSettlement(t, runtime)
	if provider.calls.Load() < 2 {
		t.Fatalf("provider calls = %d, want at least the parent call plus the child's", provider.calls.Load())
	}
	waitTool := session.WaitTool()
	waited, err := waitTool.Execute(context.Background(), "wait-1", json.RawMessage(`{"timeout":10}`), nil)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if !strings.Contains(textOf(waited), "child says: all green") {
		t.Errorf("wait result = %q, want the child's answer", textOf(waited))
	}
	var spawnAck string
	for _, message := range messages {
		if message.ToolResult != nil && message.ToolResult.ToolName == "task" {
			spawnAck += ai.ContentText(message.ToolResult.Content)
		}
	}
	if !strings.Contains(spawnAck, "Spawned") {
		t.Errorf("task tool result = %q, want a spawn acknowledgement", spawnAck)
	}
	entries := runtime.Registry().List()
	if len(entries) != 1 {
		t.Fatalf("registered agents = %d, want 1", len(entries))
	}
	if entries[0].Status != subagent.StatusIdle {
		t.Errorf("agent status = %q, want idle", entries[0].Status)
	}
	output, err := runtime.ReadAgentOutput(entries[0].ID)
	if err != nil {
		t.Fatalf("agent output: %v", err)
	}
	if !strings.Contains(output, "all green") {
		t.Errorf("agent:// output = %q", output)
	}
}

// TestSubagentBatchSpawnsEveryItem checks the batch shape the model is told to
// prefer.
func TestSubagentBatchSpawnsEveryItem(t *testing.T) {
	session, _ := newSubagentTestSession(t, "batched")
	runtime := session.SubagentRuntime()
	batch := subagent.BatchRequest{
		Context: "shared background",
		Tasks: []subagent.SpawnRequest{
			{Name: "One", Task: "first"},
			{Name: "Two", Task: "second"},
			{Name: "Three", Task: "third"},
		},
	}
	results, err := runtime.Runner().SpawnBatch(context.Background(), "", batch)
	if err != nil {
		t.Fatalf("SpawnBatch: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	awaitSettlement(t, runtime)
	entries := runtime.Registry().List()
	if len(entries) != 3 {
		t.Fatalf("registered = %d, want 3", len(entries))
	}
	for _, entry := range entries {
		if !strings.Contains(entry.Task, "shared background") {
			t.Errorf("agent %s task = %q, want the shared context", entry.ID, entry.Task)
		}
	}
}

// TestSubagentNameCollisionGetsSuffix proves the id allocator keeps names
// unique instead of overwriting an artifact.
func TestSubagentNameCollisionGetsSuffix(t *testing.T) {
	session, _ := newSubagentTestSession(t, "ok")
	runtime := session.SubagentRuntime()
	for i := 0; i < 3; i++ {
		if _, err := runtime.Runner().Run(context.Background(), "", subagent.SpawnRequest{
			Name: "Same", Task: "work",
		}); err != nil {
			t.Fatalf("spawn %d: %v", i, err)
		}
	}
	entries := runtime.Registry().List()
	if len(entries) != 3 {
		t.Fatalf("registered = %d, want 3", len(entries))
	}
	ids := map[string]bool{}
	for _, entry := range entries {
		if ids[entry.ID] {
			t.Errorf("duplicate id %q", entry.ID)
		}
		ids[entry.ID] = true
	}
	for _, want := range []string{"Same", "Same-2", "Same-3"} {
		if !ids[want] {
			t.Errorf("missing id %q, got %v", want, ids)
		}
	}
}

// TestSubagentUnknownAgentIsRejected checks preflight: an unknown agent name
// fails before a child is created.
func TestSubagentUnknownAgentIsRejected(t *testing.T) {
	session, _ := newSubagentTestSession(t, "ok")
	_, err := session.SubagentRuntime().Runner().Run(context.Background(), "", subagent.SpawnRequest{
		Agent: "nope", Task: "work",
	})
	if err == nil || !strings.Contains(err.Error(), "unknown agent") {
		t.Fatalf("Run error = %v, want an unknown-agent rejection", err)
	}
	if len(session.SubagentRuntime().Registry().List()) != 0 {
		t.Error("a rejected spawn still registered an agent")
	}
}

// TestSubagentDefinitionsDiscoveredFromProject proves a project agent file is
// loadable and selectable by name.
func TestSubagentDefinitionsDiscoveredFromProject(t *testing.T) {
	def, err := subagent.ParseDefinition(`---
name: custom
description: A custom agent.
tools: read, grep
model: "@slow"
thinking-level: high
---

Do the custom thing.
`)
	if err != nil {
		t.Fatalf("ParseDefinition: %v", err)
	}
	if def.Name != "custom" || def.Description != "A custom agent." {
		t.Errorf("definition = %+v", def)
	}
	if len(def.Tools) != 2 || def.Tools[0] != "read" || def.Tools[1] != "grep" {
		t.Errorf("tools = %v, want [read grep]", def.Tools)
	}
	if def.ThinkingLevel != "high" {
		t.Errorf("thinking level = %q, want high", def.ThinkingLevel)
	}
	if !strings.Contains(def.SystemPrompt, "custom thing") {
		t.Errorf("system prompt = %q", def.SystemPrompt)
	}
	if _, err := subagent.ParseDefinition("---\nname: main\ndescription: x\n---\nbody"); err == nil {
		t.Error("a reserved name was accepted")
	}
}

// TestSubagentOutputSchemaRoundTrip checks the structured sidecar.
func TestSubagentOutputSchemaRoundTrip(t *testing.T) {
	artifacts := subagent.NewArtifacts(t.TempDir())
	payload := json.RawMessage(`{"reports":[{"data":{"ok":true}}]}`)
	if _, err := artifacts.WriteOutput("Structured", "text", payload); err != nil {
		t.Fatalf("WriteOutput: %v", err)
	}
	value, ok, err := artifacts.JSONPath("Structured", []string{"reports", "0", "data", "ok"})
	if err != nil || !ok {
		t.Fatalf("JSONPath: %v ok=%v", err, ok)
	}
	if string(value) != "true" {
		t.Errorf("JSONPath value = %s, want true", value)
	}
	// A later output without structured data must clear the stale sidecar.
	if _, err := artifacts.WriteOutput("Structured", "plain", nil); err != nil {
		t.Fatalf("WriteOutput: %v", err)
	}
	if _, ok, _ := artifacts.ReadStructured("Structured"); ok {
		t.Error("stale structured sidecar survived a plain output")
	}
}
