package pigletbuild

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Pi config.ts:528-534 honors the agent-directory override before HOME, and trust-manager.ts:210-225 locks that directory's trust.json. A test launcher must isolate both paths, not merely replace HOME.
func assertPigletStartupIgnoresParentHomes(t *testing.T, artifact string) {
	t.Helper()
	for _, tc := range []struct {
		name     string
		env      string
		value    string
		agentDir string
		piDirs   string
	}{
		{name: "HOME", agentDir: ".ompig/agent"},
		{name: "OMPIG_HOME", env: "OMPIG_HOME", value: "ompig-home", agentDir: "ompig-home/agent"},
		{name: "XDG_CONFIG_HOME", env: "XDG_CONFIG_HOME", value: "config", agentDir: "config/ompig/agent"},
		{name: "OMPIG_CODING_AGENT_DIR", env: "OMPIG_CODING_AGENT_DIR", value: "ompig-agent", agentDir: "ompig-agent"},
		{name: "Pi HOME", piDirs: "1", agentDir: ".pi/agent"},
		{name: "PI_CODING_AGENT_DIR", piDirs: "1", env: "PI_CODING_AGENT_DIR", value: "pi-agent", agentDir: "pi-agent"},
	} {
		t.Run("startup ignores parent "+tc.name, func(t *testing.T) {
			parentHome := t.TempDir()
			for _, key := range []string{"OMPIG_HOME", "XDG_CONFIG_HOME", "OMPIG_CODING_AGENT_DIR", "PI_CODING_AGENT_DIR"} {
				t.Setenv(key, "")
			}
			t.Setenv("HOME", parentHome)
			t.Setenv("USERPROFILE", parentHome)
			t.Setenv("OMPIG_USE_PI_DIRS", tc.piDirs)
			if tc.env != "" {
				t.Setenv(tc.env, filepath.Join(parentHome, tc.value))
			}
			lock := filepath.Join(parentHome, tc.agentDir, "trust.json.lock")
			if err := os.MkdirAll(filepath.Dir(lock), 0o700); err != nil {
				t.Fatal(err)
			}
			const sentinel = "parent lock must remain untouched"
			if err := os.WriteFile(lock, []byte(sentinel), 0o600); err != nil {
				t.Fatal(err)
			}
			before := treeFiles(t, parentHome)
			if code, output := startPigletBinary(t, artifact); code != 0 {
				t.Fatalf("unsigned Ompiglet Binary inherited parent trust: exit %d\n%s", code, output)
			}
			if data, err := os.ReadFile(lock); err != nil || string(data) != sentinel {
				t.Fatalf("parent lock changed: %q, %v", data, err)
			}
			if after := treeFiles(t, parentHome); !slices.Equal(after, before) {
				t.Fatalf("startup wrote into parent HOME: before %v, after %v", before, after)
			}
		})
	}
}
