package codingagent

import (
	"strings"
	"testing"
)

// TestAgentsSlashCommandOpensTheHub calls the real handler: it must open the
// roster through the context hook, and report unavailability instead of
// silently doing nothing when the session cannot spawn subagents.
func TestAgentsSlashCommandOpensTheHub(t *testing.T) {
	r := NewSlashRegistry()

	t.Run("opens the roster", func(t *testing.T) {
		opened := 0
		sc := &SlashContext{
			Append:       func(string) {},
			AppendText:   func(string) {},
			OpenAgentHub: func() { opened++ },
		}
		if err := r.Dispatch(sc, "/agents", nil); err != nil {
			t.Fatalf("/agents: %v", err)
		}
		if opened != 1 {
			t.Errorf("OpenAgentHub called %d times, want 1", opened)
		}
	})

	t.Run("reports unavailability", func(t *testing.T) {
		var out strings.Builder
		sc := &SlashContext{
			Append:     func(s string) { out.WriteString(s) },
			AppendText: func(s string) { out.WriteString(s) },
		}
		if err := r.Dispatch(sc, "/agents", nil); err != nil {
			t.Fatalf("/agents: %v", err)
		}
		if !strings.Contains(out.String(), "unavailable") {
			t.Errorf("output = %q, want an unavailability note", out.String())
		}
	})

	t.Run("is a known command", func(t *testing.T) {
		match, ok := r.Match("/agents")
		if !ok || match.Builtin == nil || match.Builtin.Name != "agents" {
			t.Errorf("/agents did not resolve to the roster command: %+v ok=%v", match, ok)
		}
	})
}
