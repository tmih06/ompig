package codingagent

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmih06/ompig/coding/piglogin"
)

// /sprite create asks the model to write the sprite extension into the global extensions directory of the agent directory
// the CLI uses (AgentDir), so OMPIG_HOME, XDG_CONFIG_HOME, OMPIG_CODING_AGENT_DIR and shared-mode runs load it on /reload
// instead of the model writing it under ~/.ompig/agent.
func TestSpriteCreatePromptNamesTheRunsExtensionsDirectory(t *testing.T) {
	home := t.TempDir()
	cases := []struct {
		name string
		env  map[string]string
	}{
		{"default", nil},
		{"OMPIG_HOME", map[string]string{"OMPIG_HOME": filepath.Join(home, "pighome")}},
		{"OMPIG_HOME tilde", map[string]string{"OMPIG_HOME": "~/tilde-home"}},
		{"XDG_CONFIG_HOME", map[string]string{"XDG_CONFIG_HOME": filepath.Join(home, "xdg")}},
		{"OMPIG_HOME over XDG_CONFIG_HOME", map[string]string{"OMPIG_HOME": filepath.Join(home, "pighome"), "XDG_CONFIG_HOME": filepath.Join(home, "xdg")}},
		{"OMPIG_CODING_AGENT_DIR", map[string]string{"OMPIG_CODING_AGENT_DIR": "~/custom-agent", "OMPIG_HOME": filepath.Join(home, "pighome")}},
		{"shared mode", map[string]string{"OMPIG_USE_PI_DIRS": "1", "OMPIG_CODING_AGENT_DIR": filepath.Join(home, "ignored")}},
		{"shared mode PI_CODING_AGENT_DIR", map[string]string{"OMPIG_USE_PI_DIRS": "1", "PI_CODING_AGENT_DIR": filepath.Join(home, "pi-agent")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			for _, name := range []string{"OMPIG_HOME", "XDG_CONFIG_HOME", "OMPIG_CODING_AGENT_DIR", "OMPIG_USE_PI_DIRS", "PI_CODING_AGENT_DIR"} {
				t.Setenv(name, tc.env[name])
			}
			want := "`" + filepath.Join(AgentDir(), "extensions", "<id>.ts") + "`"
			prompt, err := piglogin.CreatePrompt()
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(prompt, want) {
				t.Fatalf("prompt does not name %s:\n%s", want, prompt)
			}
		})
	}
}
