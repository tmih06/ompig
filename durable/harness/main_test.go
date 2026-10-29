package harness

import (
	"os"
	"testing"

	"github.com/tmih06/ompig/internal/testenv"
)

func TestMain(m *testing.M) {
	os.Exit(testenv.RunWithoutGoroutineLeaks(m, endCrashedProcesses))
}
