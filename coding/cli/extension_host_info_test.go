package cli

import (
	"testing"

	"github.com/tmih06/ompig/coding"
)

func TestExtensionHostInfoNamesPigsConfigurationTree(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OMPIG_CODING_AGENT_DIR", dir)
	t.Setenv("OMPIG_USE_PI_DIRS", "")
	info := extensionHostInfo()
	if info.AgentDir != dir || info.ConfigDirName != ".ompig" || info.Version != coding.UpstreamPackageVersion {
		t.Fatalf("host info = %+v, want agent dir %q, .ompig and Pi %s", info, dir, coding.UpstreamPackageVersion)
	}
}

func TestExtensionHostInfoSharingPisTreeUsesPisNames(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OMPIG_USE_PI_DIRS", "1")
	t.Setenv("PI_CODING_AGENT_DIR", dir)
	t.Setenv("OMPIG_CODING_AGENT_DIR", t.TempDir())
	info := extensionHostInfo()
	if info.AgentDir != dir || info.ConfigDirName != ".pi" {
		t.Fatalf("host info = %+v, want Pi's agent dir %q and .pi", info, dir)
	}
}
