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

	"github.com/tmih06/ompig/internal/linkerexec"
	"github.com/tmih06/ompig/internal/packagemanager"
	"github.com/tmih06/ompig/internal/toolchain"
)

const acpPackageSource = "npm:@ompig/pigpen-acp"

// acpSourceRel is where the Package ships the ompig-acp source, relative to its root.
var acpSourceRel = filepath.Join("extensions", "acp", "cmd", "ompig-acp")

// runACPCommand handles `ompig acp install`. ompig additive (D18): the Pigpen release flow publishes no prebuilt
// ompig-acp, so this builds the source the installed Package ships with the local Go toolchain; it needs no network.
func runACPCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "acp" {
		return -1
	}
	if len(args) != 2 || args[1] != "install" {
		_, _ = fmt.Fprintln(stderr, "Usage: ompig acp install\n\nBuild ompig-acp from the installed "+acpPackageSource+" Package and put it in the Ompig bin directory ($OMPIG_BIN's directory, else ~/.local/bin).")
		if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
			return 0
		}
		return 2
	}
	cwd, agentDir, sm, err := packageContext()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ompig acp install: %v\n", err)
		return 1
	}
	src := ""
	for _, local := range []bool{false, true} {
		if root := packagemanager.InstalledPathForSource(cwd, agentDir, sm, acpPackageSource, local); root != "" {
			if _, err := os.Stat(filepath.Join(root, acpSourceRel, "go.mod")); err == nil { //perf:ok cold: one stat per `ompig acp install`
				src = filepath.Join(root, acpSourceRel)
				break
			}
		}
	}
	if src == "" {
		_, _ = fmt.Fprintf(stderr, "ompig acp install: the %s Package is not installed; run `ompig install %s` first\n", acpPackageSource, acpPackageSource)
		return 1
	}
	target, err := acpBuildTarget()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ompig acp install: %v\n", err)
		return 1
	}
	if err := buildACP(context.Background(), src, target, stderr); err != nil { //portlint:allow ctxbackground CLI entry point owns the build
		_, _ = fmt.Fprintf(stderr, "ompig acp install: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "installed ompig-acp: %s\n", target)
	if _, err := exec.LookPath("ompig-acp"); err != nil {
		_, _ = fmt.Fprintf(stdout, "%s is not on your PATH; add it, or use the full path as the editor command.\n", filepath.Dir(target))
	}
	return 0
}

func acpBuildTarget() (string, error) {
	ompig, err := ompigBuildTarget()
	if err != nil {
		return "", err
	}
	name := "ompig-acp"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(filepath.Dir(ompig), name), nil
}

func buildACP(ctx context.Context, src, target string, stderr io.Writer) error {
	goTool, err := toolchain.ResolveGo()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("building ompig-acp needs Go %s or newer, and none was found; run `ompig setup go` or install it from https://go.dev/dl/: %w", toolchain.MinGoVersion, err)
		}
		return err
	}
	if goTool.BelowFloor() {
		return errors.New(goTool.FloorRemedy())
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	cmd := linkerexec.CommandContext(ctx, goTool.Command, "build", "-buildvcs=false", "-trimpath", "-o", target, ".")
	cmd.Dir = src
	cmd.Env = toolchain.WorkDirEnv(src, goTool.Environ(os.Environ()))
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}
	return nil
}
