//go:build !ompig_strip_mcp

package cli

import (
	"path/filepath"
	"slices"
	"testing"
)

// Pi 0.99.2 `pi --mode rpc` get_commands lists the built-in extension commands in builtInExtensions order, llama then mcp
// (probed with the 0.99.2 binary: `llama extension builtin:llama.cpp`, `mcp extension builtin:mcp`).
// ompig divergence (D2): Ompig's own built-in `ompig-login` follows them and registers /sprite.
// Ports .upstream/v0.99.2/packages/coding-agent/src/extensions/index.ts:7-14 with src/modes/rpc/rpc-mode.ts:683.
func TestRPCGetCommandsListsBuiltinExtensionCommandsInIndexOrder(t *testing.T) {
	home := t.TempDir()
	p := startRPCProcessAt(t, t.TempDir(), []string{
		"HOME=" + home, "USERPROFILE=" + home, "OMPIG_HOME=" + home,
		"OMPIG_CODING_AGENT_DIR=" + filepath.Join(home, "agent"), "OMPIG_TEST_FAUX=1", "OMPIG_OFFLINE=1",
	}, "--no-session")
	p.send(`{"id":"commands","type":"get_commands"}`)
	p.await("the command catalog", func(record rpcRecord) bool {
		if record["type"] != "response" || record["id"] != "commands" {
			return false
		}
		data, _ := record["data"].(map[string]any)
		commands, _ := data["commands"].([]any)
		var builtin []string
		for _, value := range commands {
			command, _ := value.(map[string]any)
			if info, _ := command["sourceInfo"].(map[string]any); info["source"] == "builtin" {
				builtin = append(builtin, command["name"].(string))
			}
		}
		if want := []string{"llama", "mcp", "sprite"}; !slices.Equal(builtin, want) {
			t.Fatalf("built-in extension commands = %v, want %v", builtin, want)
		}
		return true
	})
	p.closeAndWait("the catalog was read")
}
