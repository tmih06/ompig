package cli

import (
	"github.com/tmih06/ompig/internal/codingagent"
	"github.com/tmih06/ompig/internal/packagemanager"
)

// resolveCLIExtensionSource resolves one -e source to the root its extensions load from; packagemanager.ResolveTemporarySource owns the temporary install.
func resolveCLIExtensionSource(cwd, agentDir string, sm *codingagent.SettingsManager, source string, progress packagemanager.ProgressCallback) (string, error) {
	return packagemanager.ResolveTemporarySource(cwd, agentDir, sm, source, progress)
}
