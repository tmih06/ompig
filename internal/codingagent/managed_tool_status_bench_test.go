package codingagent

import (
	"context"
	"testing"

	"github.com/tmih06/ompig/internal/codingagent/tools"
	"github.com/tmih06/ompig/tui"
)

func BenchmarkManagedToolStartup(b *testing.B) {
	b.Setenv("PATH", b.TempDir())
	b.Setenv("OMPIG_OFFLINE", "1")
	tm := tools.NewToolsManager(b.TempDir())
	b.ReportAllocs()
	for b.Loop() {
		m := &InteractiveMode{chatContainer: tui.NewContainer()}
		m.ensureManagedTools(context.Background(), tm)
		_ = m.chatContainer.Render(120)
	}
}
