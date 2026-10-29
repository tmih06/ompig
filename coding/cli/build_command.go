package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tmih06/ompig/internal/buildprogress"
	"github.com/tmih06/ompig/internal/configroot"
	"github.com/tmih06/ompig/internal/linkerexec"
	"github.com/tmih06/ompig/internal/toolchain"
)

const ompigModulePath = "github.com/tmih06/ompig"

func runBuildCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "build" {
		return -1
	}
	if len(args) > 1 && (args[1] == "-h" || args[1] == "--help") {
		_, _ = fmt.Fprint(stdout, `Usage: ompig build [--verbose]

Build the ompig binary from the current source checkout and install it to
$OMPIG_BIN, or ~/.local/bin/ompig when OMPIG_BIN is unset.

Run this from inside the github.com/tmih06/ompig source tree, or from a
checkout root containing ./ompig.
Use --verbose to stream toolchain output. Progress is written to stderr.
`)
		return 0
	}
	verbose := false
	for _, arg := range args[1:] {
		if arg != "--verbose" {
			_, _ = fmt.Fprintf(stderr, "unknown ompig build argument %q\n", arg)
			return 2
		}
		verbose = true
	}
	// ompig additive (D18): explicit builds report real work without changing compiler inputs.
	progress := buildprogress.New(stderr, verbose)
	defer progress.Close()
	ctx := buildprogress.Observe(context.Background(), progress.Handle, verbose)
	fail := func(err error) int {
		writeCLIError(stderr, fmt.Sprint(progress.Failure(err)), false)
		return 1
	}
	buildprogress.Phase(ctx, "Resolving source", "Locating the Ompig checkout and output path")

	root, err := findPigSourceRoot()
	if err != nil {
		return fail(fmt.Errorf("%w; run from the ompig checkout or set OMPIG_SOURCE_ROOT=/path/to/github.com/tmih06/ompig", err))
	}
	target, err := ompigBuildTarget()
	if err != nil {
		return fail(err)
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fail(err)
	}

	build := gitShortCommit(root)
	ldflags := fmt.Sprintf("-s -w -X github.com/tmih06/ompig/coding/cli.Build=%s", build)
	buildprogress.Phase(ctx, "Compiling and linking Go binary", "Ompig → "+target+" (includes module resolution)")
	buildArgs := buildprogress.ToolArgs(ctx, "go", []string{"build", "-buildvcs=false", "-trimpath", "-ldflags", ldflags, "-o", target, "./cmd/ompig"})
	goToolchain, err := toolchain.ResolveGo()
	if err != nil {
		return fail(err)
	}
	cmd := linkerexec.CommandContext(ctx, goToolchain.Command, buildArgs...)
	cmd.Dir = root
	cmd.Env = toolchain.WorkDirEnv(root, goToolchain.Environ(os.Environ()))
	if err := buildprogress.Run(buildprogress.Member(ctx, "ompig"), cmd); err != nil {
		return fail(fmt.Errorf("go build: %w", err))
	}
	if runtime.GOOS == "darwin" {
		signPigBuild(ctx, progress, target, stderr)
	}
	info, err := os.Stat(target)
	if err != nil {
		return fail(err)
	}
	progress.Close()
	_, _ = fmt.Fprintf(stdout, "installed: %s → %s\n", builtPigVersion(target), target)
	reportActivePigBinary(target, stdout)
	progress.Success(target, info.Size())
	return 0
}

func signPigBuild(ctx context.Context, progress *buildprogress.Reporter, target string, stderr io.Writer) {
	codesign, err := exec.LookPath("codesign")
	if err != nil {
		return
	}
	buildprogress.Phase(ctx, "Signing binary", "Ad-hoc codesign: "+target)
	cmd := exec.CommandContext(ctx, codesign, "--force", "--sign", "-", target)
	if err := buildprogress.Run(buildprogress.Member(ctx, "ompig"), cmd); err != nil {
		// ompig additive (D18): ad-hoc signing remains best-effort, but its failure is visible.
		_, _ = fmt.Fprintf(stderr, "warning: %v\n", progress.Failure(fmt.Errorf("ad-hoc codesign: %w", err)))
	}
}

func reportActivePigBinary(target string, stdout io.Writer) {
	active, err := exec.LookPath("ompig")
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "warning: no ompig found on PATH; add %s to PATH or set OMPIG_BIN to your active binary\n", filepath.Dir(target))
		return
	}
	targetReal := resolvedPath(target)
	activeReal := resolvedPath(active)
	if targetReal == activeReal {
		_, _ = fmt.Fprintf(stdout, "active: %s\n", active)
		return
	}
	_, _ = fmt.Fprintf(stdout, "warning: your shell resolves ompig to %s, not %s\n", active, target)
	_, _ = fmt.Fprintf(stdout, "         run OMPIG_BIN=%q ompig build, update PATH, or run hash -r after changing shells\n", active)
}

func resolvedPath(path string) string {
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	real, err := filepath.EvalSymlinks(path)
	if err == nil {
		return real
	}
	return filepath.Clean(path)
}

func ompigBuildTarget() (string, error) {
	if target := os.Getenv("OMPIG_BIN"); target != "" {
		return filepath.Clean(target), nil
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("cannot resolve home directory; set OMPIG_BIN")
	}
	// Windows starts a program by name only when it has an executable extension.
	name := "ompig"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(home, ".local", "bin", name), nil
}

func findPigSourceRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if root, ok := findPigSourceRootFrom(cwd); ok {
		return root, nil
	}
	if root, ok := findPigSourceRootFrom(filepath.Join(cwd, "ompig")); ok {
		return root, nil
	}
	for _, candidate := range ompigSourceRootCandidates() {
		if root, ok := findPigSourceRootFrom(candidate); ok {
			return root, nil
		}
	}
	return "", errors.New("ompig build must be run from inside the github.com/tmih06/ompig source checkout")
}

func findPigSourceRootFrom(start string) (string, bool) {
	if start == "" {
		return "", false
	}
	cwd, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	if real, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = real
	}
	for {
		modPath := filepath.Join(cwd, "go.mod")
		data, err := os.ReadFile(modPath)
		if err == nil && strings.Contains(string(data), "module "+ompigModulePath) {
			return cwd, true
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			return "", false
		}
		cwd = parent
	}
}

func ompigSourceRootCandidates() []string {
	var candidates []string
	if root := os.Getenv("OMPIG_SOURCE_ROOT"); root != "" {
		candidates = append(candidates, root)
	}
	if root, err := configroot.Resolve(); err == nil {
		candidates = append(candidates, filepath.Join(root, "source"))
	}
	return candidates
}

func gitShortCommit(root string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = root
	linkerexec.Prepare(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "dev"
	}
	build := strings.TrimSpace(string(out))
	if build == "" {
		return "dev"
	}
	return build
}

// builtPigVersion names the installed binary from the fields of `ompig version`,
// which every Ompig build prints.
func builtPigVersion(path string) string {
	if out, err := linkerexec.Command(path, "version").Output(); err == nil {
		if identity, ok := ompigVersionIdentity(string(out)); ok {
			return identity
		}
	}
	return "ompig"
}

// ompigVersionIdentity reads detailedVersionString output into "ompig X+Y", the
// composite version (coding.Version).
func ompigVersionIdentity(out string) (string, bool) {
	var ompig, pi string
	for line := range strings.Lines(out) {
		key, value, _ := strings.Cut(strings.TrimSpace(line), ": ")
		switch key {
		case "ompig":
			ompig = value
		case "upstream pi":
			pi = value
		}
	}
	switch {
	case ompig == "":
		return "", false
	case pi == "":
		return "ompig " + ompig, true
	default:
		// D63: semver allows one plus sign, so a commit pin's "1.1.0+42a3497d" joins as build metadata "1.1.0.42a3497d", as in Version.
		return fmt.Sprintf("ompig %s+%s", ompig, strings.ReplaceAll(pi, "+", ".")), true
	}
}
