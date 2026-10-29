//go:build integration

package integration

import (
	"testing"

	"github.com/tmih06/ompig/internal/testenv"
)

// Guards the TestMain temporary-directory scope: ompig and tmux children write scratch files under the run's directory, which TestMain removes.
func TestTempDirIsScopedToTheIntegrationRun(t *testing.T) {
	testenv.RequireScopedTempDir(t, "ompig-integration-")
}
