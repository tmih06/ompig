package node

import (
	"testing"

	"github.com/tmih06/ompig/internal/testenv"
)

// Guards the TestMain temporary-directory scope: temporary directories and overflow logs made through the execution environment must land in the directory TestMain removes.
func TestTempDirIsScopedToThePackageRun(t *testing.T) {
	testenv.RequireScopedTempDir(t, "ompig-env-")
}
