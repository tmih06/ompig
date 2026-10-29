package subprocess

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"
)

func TestNewHostUsesEnvironmentHomeForConfigRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("OMPIG_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	// The home directory is HOME on Unix and USERPROFILE on Windows.
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	host := NewHost(t.TempDir())
	want := filepath.Join(home, ".ompig")
	if host.builder.configRoot != want {
		t.Fatalf("builder config root = %q, want %q", host.builder.configRoot, want)
	}
	if got := os.Getenv("HOME"); got != home {
		t.Fatalf("HOME = %q, want %q", got, home)
	}
}

func TestNewHostUsesXDGConfigRoot(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("OMPIG_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	host := NewHost(t.TempDir())
	want := filepath.Join(xdg, "ompig")
	if host.builder.configRoot != want {
		t.Fatalf("builder config root = %q, want %q", host.builder.configRoot, want)
	}
}

func TestNewHostWithConfigRootIgnoresConflictingEnvironment(t *testing.T) {
	t.Setenv("OMPIG_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	explicit := filepath.Join(t.TempDir(), "ompig-root")

	host := NewHostWithConfigRoot(t.TempDir(), explicit)
	if host.builder.configRoot != explicit {
		t.Fatalf("builder config root = %q, want explicit %q", host.builder.configRoot, explicit)
	}
}

func TestBuilderUsesExplicitConfigRootInGeneratedGoMod(t *testing.T) {
	configRoot := filepath.Join(t.TempDir(), "isolated-ompig")
	staged := stagedSDKDir(t, configRoot, "go")
	if err := os.MkdirAll(staged, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staged, "go.mod"), []byte("module github.com/tmih06/ompig/extensions/sdk\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "go.mod"), []byte("module example.com/mascot\n\ngo 1.26\n\nrequire github.com/tmih06/ompig/extensions/sdk v0.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	builder := NewBuilderWithConfigRoot(t.TempDir(), configRoot)
	resolved, err := builder.resolveStagedSDK(source, "go")
	if err != nil {
		t.Fatal(err)
	}
	if resolved != staged {
		t.Fatalf("resolved SDK = %q, want %q", resolved, staged)
	}
	modPath, cleanup, err := stagedGoModFile(source, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	data, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "replace github.com/tmih06/ompig/extensions/sdk => "+modfile.AutoQuote(filepath.ToSlash(staged))) {
		t.Fatalf("generated go.mod does not use isolated config root:\n%s", data)
	}
}
