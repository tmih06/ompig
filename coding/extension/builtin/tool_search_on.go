//go:build !ompig_strip_tool_search

package builtin

import (
	"github.com/tmih06/ompig/coding/extension"
	"github.com/tmih06/ompig/coding/extension/builtin/toolsearch"
	"github.com/tmih06/ompig/extensions/sdk/json"
	"github.com/tmih06/ompig/internal/compiledin"
	"github.com/tmih06/ompig/internal/pigstrip"
)

// registerToolSearchReference makes the tool-search extension the one an extension names with createToolSearchExtension() in a loader it constructs (internal/compiledin).
func registerToolSearchReference() { compiledin.Register("tool-search", createToolSearchReference) }

func createToolSearchReference(json.RawMessage) (extension.InlineExtension, error) {
	return toolsearch.CreateToolSearchExtension(), nil
}

// toolSearchEntries is the tool-search registry row, absent while tool-search is stripped at runtime as in a build without it.
// ompig additive (D92): a runtime stripped tool-search registers nothing, like the ompig_strip_tool_search build.
func toolSearchEntries() []Extension {
	if pigstrip.Has(pigstrip.ListExtensions, "tool-search") {
		return nil
	}
	return []Extension{{Name: "tool-search", Replaceable: true, Factory: toolsearch.CreateToolSearchExtension()}}
}
