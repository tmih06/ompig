package cli

import (
	"testing"

	ompiglet "github.com/tmih06/ompig/coding/ompiglet"
)

// TestApplyPigletPreStart_SystemPrompt locks ompiglet system-prompt application:
// an active ompiglet's prompt becomes the base prompt unless the user passed
// --system-prompt, and a nil / prompt-less ompiglet changes nothing.
func TestApplyPigletPreStart_SystemPrompt(t *testing.T) {
	withPrompt := &ompiglet.Ompiglet{Name: "p", SystemPrompt: &ompiglet.PromptRef{Text: "BAKED ROLE"}}

	t.Run("applies ompiglet prompt", func(t *testing.T) {
		var f Args
		applyPigletPreStart(withPrompt, &f)
		if f.SystemPrompt != "BAKED ROLE" {
			t.Fatalf("SystemPrompt = %q, want BAKED ROLE", f.SystemPrompt)
		}
	})

	t.Run("explicit --system-prompt wins", func(t *testing.T) {
		f := Args{SystemPrompt: "USER SET"}
		applyPigletPreStart(withPrompt, &f)
		if f.SystemPrompt != "USER SET" {
			t.Fatalf("SystemPrompt = %q, want the user's value", f.SystemPrompt)
		}
	})

	t.Run("nil ompiglet is a no-op", func(t *testing.T) {
		var f Args
		applyPigletPreStart(nil, &f)
		if f.SystemPrompt != "" {
			t.Fatalf("SystemPrompt = %q, want empty", f.SystemPrompt)
		}
	})

	t.Run("prompt-less ompiglet leaves it empty", func(t *testing.T) {
		var f Args
		applyPigletPreStart(&ompiglet.Ompiglet{Name: "x"}, &f)
		if f.SystemPrompt != "" {
			t.Fatalf("SystemPrompt = %q, want empty", f.SystemPrompt)
		}
	})
}
