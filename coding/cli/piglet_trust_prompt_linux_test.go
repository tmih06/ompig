//go:build linux

package cli

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/tmih06/ompig/internal/codingagent"
	"github.com/tmih06/ompig/internal/testbudget"
)

// Startup names the workspace Ompiglet that project trust withheld in the trust prompt: `--ompiglet demo` in an undecided project that defines
// `.ompig/ompiglets/demo.yaml` runs the user's demo, and the prompt says the project's demo replaces it once the project is trusted. This drives the
// real binary, so it guards the wiring from the --ompiglet flag through the runtime builder to the prompt, not only the prompt formatter.
func TestStartupTrustPromptNamesTheWithheldWorkspacePiglet(t *testing.T) {
	if testing.Short() {
		t.Skip("starts the ompig binary")
	}
	binary := buildPigBinaryForSignalTest(t)
	home := shortTempDir(t)
	agentDir := filepath.Join(home, "agent")
	seedFirstRunDone(t, agentDir)
	t.Setenv("OMPIG_HOME", home)
	project := t.TempDir()
	for path, body := range map[string]string{
		filepath.Join(codingagent.PigletsDir(), "demo.yaml"):                           "name: demo\n",
		filepath.Join(codingagent.ProjectConfigDir(project), "ompiglets", "demo.yaml"): "name: demo\n",
		filepath.Join(project, ".agents", "skills", "planted", "SKILL.md"):             "---\nname: planted\ndescription: planted\n---\nplanted\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	master, slave := openPTY(t, 40, 160)
	t.Cleanup(func() { _ = master.Close() })
	cmd := exec.Command(binary, "--ompiglet", "demo", "--model", "test-faux/faux-1", "--no-extensions", "--session-dir", filepath.Join(home, "sessions"))
	cmd.Dir = project
	cmd.Env = append(os.Environ(), "OMPIG_HOME="+home, "OMPIG_CODING_AGENT_DIR="+agentDir, "OMPIG_TEST_FAUX=1", "TERM=xterm-256color")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start ompig: %v", err)
	}
	_ = slave.Close()
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		<-exited
	})
	output := &ptyOutput{}
	go func() { _, _ = io.Copy(output, master) }()
	want := []byte(`This project also defines the Ompiglet "demo" in .ompig/ompiglets.`)
	deadline := time.Now().Add(testbudget.Wait(t))
	for !bytes.Contains(bytes.ReplaceAll(output.since(0), []byte("\r\n"), []byte(" ")), want) {
		if time.Now().After(deadline) {
			t.Fatalf("the trust prompt never named the withheld workspace Ompiglet; screen tail: %q", tail(output.since(0), 2000))
		}
		time.Sleep(50 * time.Millisecond)
	}
}
