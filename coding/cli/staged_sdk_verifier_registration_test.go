package cli_test

import (
	"testing"

	_ "github.com/tmih06/ompig/coding/cli"
	"github.com/tmih06/ompig/coding/extension/host/subprocess"
)

// The ompig command links the SDKs, which registers their staged-SDK check, so hosts that coding.DiscoverAndLoadExtensions starts
// inside it verify the staged SDK as the command's own extension hosts do.
func TestThePigCommandRegistersTheStagedSDKVerifier(t *testing.T) {
	if subprocess.DefaultStagedSDKVerifier(t.TempDir()) == nil {
		t.Fatal("importing the ompig command registered no staged-SDK verifier")
	}
}
