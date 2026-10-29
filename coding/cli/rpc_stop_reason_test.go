package cli

import (
	"testing"

	"github.com/tmih06/ompig/agent"
	"github.com/tmih06/ompig/ai"
)

// Upstream writes the session's own AssistantMessage with JSON.stringify: a
// message without a stop reason has no stopReason key. Ompig reported it as
// "stop", which a client reads as a completed response. GUARD-16.
func TestRPCAssistantMessagePassesMissingStopReasonThrough(t *testing.T) {
	for _, tc := range []struct {
		reason ai.StopReason
		want   any
		has    bool
	}{
		{"", nil, false},
		{ai.StopReasonStop, string(ai.StopReasonStop), true},
		{ai.StopReasonToolUse, string(ai.StopReasonToolUse), true},
	} {
		wire, err := rpcAgentMessage(agent.AgentMessage{Assistant: &agent.AssistantMessage{Role: agent.RoleAssistant, StopReason: tc.reason}})
		if err != nil {
			t.Fatal(err)
		}
		got, has := decodeRPCEvent(t, wire)["stopReason"]
		if has != tc.has || (has && got != tc.want) {
			t.Errorf("stop reason %q: stopReason = %v (present %v), want %v (present %v)", tc.reason, got, has, tc.want, tc.has)
		}
	}
}
