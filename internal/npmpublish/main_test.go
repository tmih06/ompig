package npmpublish_test

import (
	"os"
	"testing"

	"github.com/tmih06/ompig/internal/testenv"
)

// Command reads the npmCommand setting from the agent directory under that file's lock, so every test that publishes or resolves a package reaches the directory the environment names.
func TestMain(m *testing.M) {
	os.Exit(testenv.RunScoped(m, "ompig-npmpublish-"))
}

func TestPackageTestsDoNotReachTheExportedAgentDirectory(t *testing.T) {
	testenv.RequireIsolatedAgentEnv(t)
}
