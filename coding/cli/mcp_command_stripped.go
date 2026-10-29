//go:build ompig_strip_mcp

package cli

import (
	"fmt"
	"os"

	"github.com/tmih06/ompig/internal/pigstrip"
)

// runMcpCommand reports that a build without built-in MCP has no `ompig mcp` command, and returns -1 when args are not an `mcp`
// command.
// ompig additive (D92): `ompig mcp` in a Ompiglet Binary without MCP fails with the strip message instead of becoming a prompt.
func runMcpCommand(args []string) int {
	if len(args) == 0 || args[0] != "mcp" {
		return -1
	}
	fmt.Fprintln(os.Stderr, pigstrip.Error("ompig mcp", pigstrip.ListExtensions, "mcp"))
	return 1
}
