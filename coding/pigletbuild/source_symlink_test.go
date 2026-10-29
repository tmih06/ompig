package pigletbuild

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tmih06/ompig/internal/testenv"
	"github.com/tmih06/ompig/internal/toolchain"
)

// A Ompig checkout reached through a symbolic link, such as one under /tmp or
// /var on macOS, builds in workspace mode. The go command resolves the
// workspace's `use .` against GOWORK, which names the link, and the working
// directory against PWD; an inherited PWD naming another directory makes it
// take the physical path, which is then not a workspace module.
func TestSourceGoCommandBuildsThroughSymlinkedCheckout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not use PWD")
	}
	physical, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, filepath.Join(physical, "go.mod"), "module example.com/ompig\n\ngo 1.26\n")
	writeFixtureFile(t, filepath.Join(physical, "go.work"), "go 1.26\n\nuse .\n")
	writeFixtureFile(t, filepath.Join(physical, "p", "p.go"), "package p\n")
	link := filepath.Join(t.TempDir(), "checkout")
	testenv.RequireDirectoryLink(t, physical, link)
	// The test process's own PWD names its package directory, as an
	// interactive shell's names wherever ompig was started.
	t.Setenv("PWD", t.TempDir())
	goToolchain, err := toolchain.ResolveGo()
	if err != nil {
		t.Fatal(err)
	}
	cmd := ompigSource{Root: link}.goCommand(t.Context(), goToolchain, "build", "./p")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("workspace build through a symlinked checkout failed: %v\n%s", err, output)
	}
}
