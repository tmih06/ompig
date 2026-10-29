//go:build ompig_strip_mcp

package coding

import (
	"github.com/tmih06/ompig/coding/extension"
	"github.com/tmih06/ompig/coding/extension/host/subprocess"
)

func (f graphExtensionFactory) mcpFactory(subprocess.SessionGraphScope, string) (extension.InlineExtension, error) {
	return nil, f.absentError()
}
