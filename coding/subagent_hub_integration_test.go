package coding

import (
	"context"
	"strings"
	"testing"

	"github.com/tmih06/ompig/agent"
	"github.com/tmih06/ompig/coding/subagent"
)

// TestAgentHubDrivesTheLiveRuntime exercises the Agent Hub's control surface
// against a real session: the roster it reads is the same one the task tool
// populates, and its actions reach the child sessions.
func TestAgentHubDrivesTheLiveRuntime(t *testing.T) {
	session, _ := newSubagentTestSession(t, "hub reply")
	runtime := session.SubagentRuntime()
	if _, err := runtime.Runner().SpawnBatch(context.Background(), "", subagent.BatchRequest{
		Tasks: []subagent.SpawnRequest{
			{Name: "One", Task: "first job"},
			{Name: "Two", Task: "second job"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	awaitSettlement(t, runtime)
	// The roster is the registry, so a hub reading it sees both agents.
	entries := session.SubagentRegistry().Sorted()
	if len(entries) != 2 {
		t.Fatalf("roster = %d agents, want 2", len(entries))
	}
	if entries[0].Status != subagent.StatusIdle || entries[1].Status != subagent.StatusIdle {
		t.Errorf("statuses = %q, %q; want idle", entries[0].Status, entries[1].Status)
	}
	for _, entry := range entries {
		if entry.Usage.Total == 0 {
			t.Errorf("agent %s reports no tokens", entry.ID)
		}
		if entry.TranscriptPath == "" {
			t.Errorf("agent %s has no transcript path", entry.ID)
		}
	}

	// The transcript view opens from the live session.
	messages, ok := session.SubagentTranscript("One")
	if !ok {
		t.Fatal("no transcript for agent One")
	}
	if len(messages) == 0 {
		t.Error("transcript is empty")
	}

	// Message: steers the idle agent, which runs another turn.
	if err := session.SubagentSteer(context.Background(), "One", "one more"); err != nil {
		t.Fatalf("SubagentSteer: %v", err)
	}
	awaitSettlement(t, runtime)
	entry, _ := session.SubagentRegistry().Get("One")
	if entry.Unread == 0 {
		t.Error("a delivered message did not record as unread")
	}

	// Kill then revive.
	if err := session.SubagentKill(context.Background(), "Two"); err != nil {
		t.Fatalf("SubagentKill: %v", err)
	}
	if entry, _ := session.SubagentRegistry().Get("Two"); entry.Status != subagent.StatusAborted {
		t.Errorf("status after kill = %q, want aborted", entry.Status)
	}
	if err := session.SubagentRevive(context.Background(), "Two"); err != nil {
		t.Fatalf("SubagentRevive: %v", err)
	}
	if entry, _ := session.SubagentRegistry().Get("Two"); entry.Status != subagent.StatusIdle {
		t.Errorf("status after revive = %q, want idle", entry.Status)
	}

	// A parked agent still renders its transcript from disk.
	runtime.Park("One")
	messages, ok = session.SubagentTranscript("One")
	if !ok || len(messages) == 0 {
		t.Error("a parked agent's transcript is unavailable")
	}
}

// TestAgentHubHandleIsWired guards the seam the interactive Alt+A key uses: the
// session must expose every method the hub controller calls.
func TestAgentHubHandleIsWired(t *testing.T) {
	session, _ := newSubagentTestSession(t, "ok")
	if session.SubagentRegistry() == nil {
		t.Error("SubagentRegistry() = nil, so Alt+A would report subagents unavailable")
	}
	if session.SubagentJobs() == nil {
		t.Error("SubagentJobs() = nil, so the activity pane would be empty")
	}
	if _, ok := session.SubagentTranscript("missing"); ok {
		t.Error("an unknown agent produced a transcript")
	}
	if err := session.SubagentKill(context.Background(), "missing"); err == nil {
		t.Error("killing an unknown agent succeeded")
	} else if !strings.Contains(err.Error(), "unknown agent") {
		t.Errorf("kill error = %v, want an unknown-agent message", err)
	}
}

// TestSubagentAdvisorRowsAreReadOnly covers the hub's advisor rule: an advisor
// transcript is observable but cannot be messaged, revived or killed.
func TestSubagentAdvisorRowsAreReadOnly(t *testing.T) {
	session, _ := newSubagentTestSession(t, "ok")
	runtime := session.SubagentRuntime()
	entry := runtime.Registry().Register(subagent.Entry{
		Name:      "Advisor",
		Kind:      subagent.KindAdvisor,
		Status:    subagent.StatusIdle,
		AgentType: "advisor",
	})
	for name, action := range map[string]func() error{
		"steer":  func() error { return session.SubagentSteer(context.Background(), entry.ID, "hi") },
		"revive": func() error { return session.SubagentRevive(context.Background(), entry.ID) },
		"kill":   func() error { return session.SubagentKill(context.Background(), entry.ID) },
	} {
		err := action()
		if err == nil || !strings.Contains(err.Error(), "advisor") {
			t.Errorf("%s on an advisor = %v, want an advisor rejection", name, err)
		}
	}
	// An advisor row is still listed.
	found := false
	for _, candidate := range session.SubagentRegistry().Sorted() {
		if candidate.ID == entry.ID {
			found = true
		}
	}
	if !found {
		t.Error("analyst advisor row is missing from the roster")
	}
}

var _ = agent.RoleAssistant

// TestAgentHubRestoresAResumedRoster proves a resumed session's hub shows the
// agents that session already spawned: the roster is rebuilt from the artifact
// directory as parked rows, so the transcripts and outputs stay reachable.
func TestAgentHubRestoresAResumedRoster(t *testing.T) {
	session, _ := newSubagentTestSession(t, "restored output")
	runtime := session.SubagentRuntime()
	if _, err := runtime.Runner().Run(context.Background(), "", subagent.SpawnRequest{
		Name: "Persisted", Task: "durable work",
	}); err != nil {
		t.Fatal(err)
	}
	awaitSettlement(t, runtime)
	entryBefore, ok := runtime.Registry().Get("Persisted")
	if !ok {
		t.Fatal("Persisted not registered")
	}
	if entryBefore.OutputPath == "" {
		t.Fatal("no output artifact to restore from")
	}

	// A fresh session over the same artifact directory stands in for a restart.
	fresh, _ := newSubagentTestSession(t, "unused")
	freshRuntime := fresh.SubagentRuntime()
	// Give the fresh runtime no agents of its own before restoring, so the scan
	// sees exactly the persisted roster.
	freshRuntime.Runner().SetArtifacts(subagent.NewArtifacts(t.TempDir()))
	restoredDir := runtime.Runner().Artifacts().Dir()
	freshRuntime.SetArtifactsForTest(subagent.NewArtifacts(restoredDir))
	if got := freshRuntime.RestoreRoster(); got != 1 {
		t.Fatalf("restored %d agents, want 1", got)
	}
	restored, ok := freshRuntime.Registry().Get("Persisted")
	if !ok {
		t.Fatal("the resumed roster is missing the agent")
	}
	if restored.Status != subagent.StatusParked {
		t.Errorf("restored status = %q, want parked", restored.Status)
	}
	if restored.TranscriptPath != entryBefore.TranscriptPath {
		t.Errorf("restored transcript = %q, want %q", restored.TranscriptPath, entryBefore.TranscriptPath)
	}
	if output, err := freshRuntime.ReadAgentOutput("Persisted"); err != nil || !strings.Contains(output, "restored output") {
		t.Errorf("restored output = %q err=%v", output, err)
	}
}
