package codingagent

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/tmih06/ompig/agent"
	"github.com/tmih06/ompig/coding/extension"
	"github.com/tmih06/ompig/coding/extension/host/inproc"
)

// Pi #5581, fixed by Pi 1.2.0 (PR #10739): a custom-message run emits before_agent_start with messageType "custom" and the message (agent-session.ts:1885-1894 _prepareRun). Pi 1.1.0 seeded it directly through _runAgentPrompt and emitted nothing.
func TestCustomSeedEmitsBeforeAgentStart(t *testing.T) {
	m, ctx, cancel := newLifecycleMode(t)
	var before atomic.Int32
	var seen atomic.Pointer[extension.BeforeAgentStartEvent]
	settled := make(chan struct{})
	m.newRunner = inproc.NewRunner([]extension.Extension{{Handlers: map[string][]extension.HandlerFn{
		"before_agent_start": {func(args ...any) (any, error) {
			before.Add(1)
			event := args[0].(extension.BeforeAgentStartEvent)
			seen.Store(&event)
			return nil, nil
		}},
		"agent_settled": {func(...any) (any, error) { close(settled); return nil, nil }},
	}}}, t.TempDir())
	loopDone := make(chan struct{})
	go m.drainLoop(ctx, loopDone)
	defer func() { cancel(); <-loopDone }()
	onLoop(m, ctx, func() {
		m.runCustomMessageTurn(ctx, agent.AgentMessage{Custom: map[string]any{"role": agent.RoleCustom, "customType": "wake", "content": "WAKE-UP", "display": true, "timestamp": int64(1)}})
	})
	select {
	case <-settled:
	case <-time.After(5 * time.Second):
		t.Fatal("custom run did not settle")
	}
	if before.Load() != 1 {
		t.Fatalf("custom seed emitted before_agent_start %d times, want 1", before.Load())
	}
	if event := seen.Load(); event == nil || event.MessageType != extension.BeforeAgentStartCustom || event.Message == nil || event.Message.CustomType != "wake" {
		t.Fatalf("event = %+v", event)
	}
}
