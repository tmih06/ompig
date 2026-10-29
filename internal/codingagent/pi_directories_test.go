package codingagent

import (
	"os"
	"path/filepath"
	"testing"
)

// Pi 0.87.1 config.ts:528-573 selects .pi and PI_CODING_AGENT_DIR.
// D2 keeps that selection explicit in Ompig, independently of its product state.
func TestPiDirectoriesOptIn(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("OMPIG_HOME", filepath.Join(home, "ompig-root"))
	t.Setenv("OMPIG_CODING_AGENT_DIR", filepath.Join(home, "ompig-agent"))
	t.Setenv("PI_CODING_AGENT_DIR", "")
	cwd := t.TempDir()
	for _, value := range []string{"", "0", "false", "true", "1"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("OMPIG_USE_PI_DIRS", value)
			wantAgent, wantProject := filepath.Join(home, "ompig-agent"), filepath.Join(cwd, ".ompig")
			if value == "1" {
				wantAgent, wantProject = filepath.Join(home, ".pi", "agent"), filepath.Join(cwd, ".pi")
			}
			if got := AgentDir(); got != wantAgent {
				t.Errorf("AgentDir = %q, want %q", got, wantAgent)
			}
			if got := ProjectConfigDir(cwd); got != wantProject {
				t.Errorf("project = %q, want %q", got, wantProject)
			}
			if got := ConfigRoot(); got != filepath.Join(home, "ompig-root") {
				t.Errorf("product root moved: %q", got)
			}
		})
	}
	t.Setenv("OMPIG_USE_PI_DIRS", "1")
	t.Setenv("PI_CODING_AGENT_DIR", "~/shared-agent")
	if got := AgentDir(); got != filepath.Join(home, "shared-agent") {
		t.Fatalf("Pi override = %q", got)
	}
	t.Setenv("OMPIG_USE_PI_DIRS", "")
	if got := AgentDir(); got != filepath.Join(home, "ompig-agent") {
		t.Fatalf("Pi variable leaked into default mode: %q", got)
	}
}

func TestPiDirectoriesProjectSettingsAndSessions(t *testing.T) {
	t.Setenv("OMPIG_USE_PI_DIRS", "1")
	agentDir, cwd := t.TempDir(), t.TempDir()
	t.Setenv("PI_CODING_AGENT_DIR", agentDir)
	for name, theme := range map[string]string{".pi": "light", ".ompig": "dark"} {
		writeSettingsFixture(t, filepath.Join(cwd, name, "settings.json"), `{"theme":"`+theme+`"}`)
	}
	sm := NewSettingsManager(cwd, AgentDir())
	if got := sm.Get().Theme; got != "light" {
		t.Fatalf("selected project theme = %q, want light", got)
	}
	if err := sm.SetProjectPackages([]PackageSource{{Source: "./shared-package"}}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(cwd, ".ompig", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"theme":"dark"}` {
		t.Fatalf("wrote unselected project: %s", data)
	}
	if got := defaultSessionDir(cwd); got != filepath.Join(agentDir, "sessions", encodeCwdForSessionDir(cwd)) {
		t.Fatalf("session directory = %q", got)
	}
}
