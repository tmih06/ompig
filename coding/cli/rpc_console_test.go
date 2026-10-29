package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// Pi modes/rpc/rpc-mode.ts starts its protocol loop without printing host tool inventories. Extension-authored console output remains visible on stderr; host setup is not extension console output.
func TestRPCDoesNotPrintHostToolInventories(t *testing.T) {
	home := t.TempDir()
	p := startRPCUIFixture(t, []string{
		"OMPIG_HOME=" + home,
		"OMPIG_CODING_AGENT_DIR=" + filepath.Join(home, "agent"),
		"OMPIG_OFFLINE=1",
	}, "--no-extensions", "--no-session")
	p.closeAndWait("after command listing")
	if stderr := p.stderr.String(); strings.Contains(stderr, "ompig --rpc: loaded") || strings.Contains(stderr, "ompig --rpc: agent has") || strings.Contains(stderr, "ompig --rpc: extension") {
		t.Fatalf("RPC setup leaked host debugging output: %s", stderr)
	}
}
