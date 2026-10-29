package ompiglet

import (
	"slices"
	"strings"
)

// MCPSourcePrefix identifies tools registered by an MCP adapter for one server.
const MCPSourcePrefix = "mcp:"

// MCPAdapterExtensionName is the conventional owner of MCP tools.
const MCPAdapterExtensionName = "ompig-mcp-adapter"

// ToolInfo is the minimal metadata needed for Ompiglet tool scoping.
type ToolInfo struct {
	Name   string
	Source string
}

// ScopeTools intersects registered tools with root and extension allowlists.
// Resource loading/discovery decides which extensions exist; this function does
// not use Ompiglet membership as a second discovery gate.
func ScopeTools(ompiglet *Ompiglet, allTools []ToolInfo) []string {
	extensionTools := make(map[string]*[]string, len(ompiglet.Extensions))
	for i := range ompiglet.Extensions {
		extensionTools[ompiglet.Extensions[i].Name] = ompiglet.Extensions[i].Tools
	}

	active := make([]string, 0, len(allTools))
	for _, tool := range allTools {
		source := tool.Source
		switch {
		case source == "" || source == "builtin":
			if toolAllowed(ompiglet.BuiltinTools, tool.Name) {
				active = append(active, tool.Name)
			}
		case strings.HasPrefix(source, MCPSourcePrefix):
			if toolAllowed(extensionTools[MCPAdapterExtensionName], tool.Name) {
				active = append(active, tool.Name)
			}
		default:
			if toolAllowed(extensionTools[source], tool.Name) {
				active = append(active, tool.Name)
			}
		}
	}
	return active
}

func toolAllowed(allowlist *[]string, name string) bool {
	return allowlist == nil || slices.Contains(*allowlist, name)
}
