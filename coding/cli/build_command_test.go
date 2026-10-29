package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPigVersionIdentityNamesPigAndPi(t *testing.T) {
	want := "ompig " + OmpigVersion + "+" + strings.ReplaceAll(UpstreamVersion, "+", ".")
	if got, ok := ompigVersionIdentity(detailedVersionString()); !ok || got != want {
		t.Fatalf("ompigVersionIdentity(detailedVersionString()) = %q, %v; want %q", got, ok, want)
	}
	if got, ok := ompigVersionIdentity("ompig: 1.2.3\n"); !ok || got != "ompig 1.2.3" {
		t.Fatalf("ompigVersionIdentity without upstream = %q, %v", got, ok)
	}
	if _, ok := ompigVersionIdentity(UpstreamVersion + "\n"); ok {
		t.Fatal("ompigVersionIdentity accepted --version output")
	}
}

func TestRunBuildCommandHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runBuildCommand([]string{"build", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runBuildCommand help = %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage: ompig build") {
		t.Fatalf("help missing usage: %s", stdout.String())
	}
}

func TestRunBuildCommandRejectsForeignArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runBuildCommand([]string{"docs"}, &stdout, &stderr); code != -1 {
		t.Fatalf("runBuildCommand(non-build) = %d, want -1", code)
	}
}

func TestRunBuildCommandBuildsFromLinkedWorktreeBelowInvalidGitDir(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}

	base := t.TempDir()
	repository := filepath.Join(base, "repository")
	container := filepath.Join(base, "container")
	worktree := filepath.Join(container, "worktree")
	for _, dir := range []string{filepath.Join(repository, "cmd", "ompig"), filepath.Join(repository, "coding", "cli"), filepath.Join(container, ".git")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// The repository has ompig's layout: cmd/ompig is the program and coding/cli owns the build identity that ompig build stamps.
	cliSource := `package cli

import (
	"fmt"
	"os"
)

var Build = "dev"

func Main(args []string) {
	if len(args) > 0 && args[0] == "version" {
		fmt.Fprint(os.Stdout, "ompig: ` + OmpigVersion + `\nupstream pi: ` + UpstreamVersion + `\nbuild: ", Build, "\n")
	}
}
`
	mainSource := `package main

import (
	"os"

	"` + ompigModulePath + `/coding/cli"
)

func main() { cli.Main(os.Args[1:]) }
`
	for path, source := range map[string]string{
		"go.mod":                                 "module " + ompigModulePath + "\n\ngo 1.26\n",
		filepath.Join("coding", "cli", "cli.go"): cliSource,
		filepath.Join("cmd", "ompig", "main.go"): mainSource,
	} {
		if err := os.WriteFile(filepath.Join(repository, path), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command(git, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	runGit(base, "init", "--quiet", repository)
	runGit(repository, "add", ".")
	runGit(repository, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "--quiet", "-m", "initial")
	runGit(repository, "worktree", "add", "--quiet", "--detach", worktree)

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(worktree); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOFLAGS", "")
	target := testExecutable(filepath.Join(base, "ompig"))
	t.Setenv("OMPIG_BIN", target)

	var stdout, stderr bytes.Buffer
	if code := runBuildCommand([]string{"build"}, &stdout, &stderr); code != 0 {
		t.Fatalf("runBuildCommand = %d, stderr=%s", code, stderr.String())
	}
	wantBuild := gitShortCommit(worktree)
	version, err := exec.Command(target, "version").Output()
	if err != nil {
		t.Fatalf("built Ompig version: %v", err)
	}
	wantVersion := "ompig: " + OmpigVersion + "\nupstream pi: " + UpstreamVersion + "\nbuild: " + wantBuild + "\n"
	if got := string(version); got != wantVersion {
		t.Fatalf("built Ompig version = %q, want explicit build identity %q", got, wantVersion)
	}
	if want := "installed: ompig " + OmpigVersion + "+" + strings.ReplaceAll(UpstreamVersion, "+", ".") + " → " + target + "\n"; !strings.Contains(stdout.String(), want) {
		t.Fatalf("build output missing %q: %s", want, stdout.String())
	}
}

func TestPigBuildTargetUsesEnv(t *testing.T) {
	want := filepath.Join(t.TempDir(), "ompig-test-bin")
	t.Setenv("OMPIG_BIN", want)
	target, err := ompigBuildTarget()
	if err != nil {
		t.Fatal(err)
	}
	if target != want {
		t.Fatalf("target = %q, want %q", target, want)
	}
}

// The default install target must be runnable by name: Windows starts only
// files with an executable extension, so there it is ompig.exe.
func TestPigBuildTargetDefaultIsRunnableByName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("OMPIG_BIN", "")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	target, err := ompigBuildTarget()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".local", "bin", "ompig")
	if runtime.GOOS == "windows" {
		want += ".exe"
	}
	if target != want {
		t.Fatalf("default target = %q, want %q", target, want)
	}
}

func TestFindPigSourceRootUsesEnvFallback(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+ompigModulePath+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(outside); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OMPIG_SOURCE_ROOT", root)

	got, err := findPigSourceRoot()
	if err != nil {
		t.Fatal(err)
	}
	want := root
	if real, err := filepath.EvalSymlinks(want); err == nil {
		want = real
	}
	if got != want {
		t.Fatalf("source root = %q, want %q", got, want)
	}
}

func TestFindPigSourceRootFindsNestedCheckoutBeforeFallback(t *testing.T) {
	repo := t.TempDir()
	ompigRoot := filepath.Join(repo, "ompig")
	if err := os.MkdirAll(ompigRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ompigRoot, "go.mod"), []byte("module "+ompigModulePath+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	staleFallback := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(staleFallback, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staleFallback, "go.mod"), []byte("module "+ompigModulePath+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldwd) }()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OMPIG_SOURCE_ROOT", staleFallback)

	got, err := findPigSourceRoot()
	if err != nil {
		t.Fatal(err)
	}
	want := ompigRoot
	if real, err := filepath.EvalSymlinks(want); err == nil {
		want = real
	}
	if got != want {
		t.Fatalf("source root = %q, want nested checkout %q", got, want)
	}
}

func TestReportActivePigBinaryWarnsWhenPigNotOnPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var stdout bytes.Buffer
	reportActivePigBinary("/tmp/ompig-test-bin", &stdout)
	if !strings.Contains(stdout.String(), "warning: no ompig found on PATH") {
		t.Fatalf("missing PATH warning: %s", stdout.String())
	}
}

func TestBuiltPigVersionUsesDetailedIdentity(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	path := filepath.Join(dir, "ompig.exe")
	body := `package main
import ("fmt"; "os")
func main() {
 if len(os.Args) != 2 || os.Args[1] != "version" { os.Exit(1) }
 fmt.Print(os.Getenv("OMPIG_TEST_BUILT_IDENTITY"))
}
`
	if err := os.WriteFile(source, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("go", "build", "-o", path, source).CombinedOutput(); err != nil {
		t.Fatalf("build fixture: %v\n%s", err, output)
	}
	for _, tc := range []struct{ name, output, want string }{
		{"detailed", "ompig: own\nupstream pi: target\n", "ompig own+target"},
		{"pin only", "target\n", "ompig"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OMPIG_TEST_BUILT_IDENTITY", tc.output)
			if got := builtPigVersion(path); got != tc.want {
				t.Errorf("builtPigVersion = %q, want %q", got, tc.want)
			}
		})
	}
}
