package subprocess

import (
	"encoding/json"
	"testing"

	"github.com/tmih06/ompig/coding/extension"
	"github.com/tmih06/ompig/tui"
)

// The awaited-dialog boundary serializes one current snapshot off the UI loop. This benchmark includes palette and editor state, with no Session-log subscription.
func BenchmarkDialogStateSnapshot(b *testing.B) {
	ui := &dialogExpansionUI{UIContext: extension.NoopUIContext}
	ui.expanded.Store(true)
	bridge := NewUIBridge(func() {})
	bridge.SetUIContext(ui)
	theme := tui.ActiveTheme()
	bridge.SetThemeFunc(func() any {
		foregrounds, backgrounds := theme.ANSIPalette()
		return map[string]any{"name": theme.Name, "foregrounds": foregrounds, "backgrounds": backgrounds, "mode": theme.GetColorMode()}
	})
	b.ReportAllocs()
	for b.Loop() {
		if _, err := json.Marshal(bridge.Snapshot(nil, 0, false)); err != nil {
			b.Fatal(err)
		}
	}
}
