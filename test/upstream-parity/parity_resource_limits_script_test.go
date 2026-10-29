package parity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmih06/ompig/internal/testenv"
)

func TestParityResourcePigletsBoundConcurrentPigPiPairs(t *testing.T) {
	root := ompigRepoRoot(t)
	script := filepath.Join(root, "automation", "ci", "parity-resource-limits.sh")
	want := map[string]string{
		"low":    "2 process=2,process-ext=1,tmux=2,tmux-ext=1,ht=1,rpc=1",
		"medium": "3 process=3,process-ext=1,tmux=3,tmux-ext=1,ht=2,rpc=2",
		"high":   "4 process=4,process-ext=1,tmux=4,tmux-ext=1,ht=2,rpc=3",
	}
	for ompiglet, expected := range want {
		t.Run(ompiglet, func(t *testing.T) {
			cmd := exec.Command(testenv.Bash(t), script)
			cmd.Env = append(os.Environ(), "OMPIG_PARITY_PROFILE="+ompiglet)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("ompiglet failed: %v\n%s", err, output)
			}
			if got := strings.TrimSpace(string(output)); got != expected {
				t.Fatalf("ompiglet = %q, want %q", got, expected)
			}
		})
	}
}
