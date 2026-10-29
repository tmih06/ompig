//go:build parity

package runner

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// prepareChangelogFixture changes data only: Go embeds the pinned document, gzipped, through a build overlay,
// while Pi reads a private package view through its own PI_PACKAGE_DIR override (config.ts:389-394,455-456).
// Builds finish before terminal timing starts. Neither checkout nor installed Pi files are modified.
func prepareChangelogFixture(ctx context.Context, t *testing.T, sc *Scenario, ompig, pi BinaryRef) (BinaryRef, BinaryRef, error) {
	t.Helper()
	if sc.ChangelogFixture == "" {
		return ompig, pi, nil
	}
	root, err := findRepoRoot()
	if err != nil {
		return ompig, pi, err
	}
	data, err := os.ReadFile(resolveScenarioCWD(sc.SourcePath, sc.ChangelogFixture))
	if err != nil {
		return ompig, pi, err
	}
	tmp := t.TempDir()
	// The binary embeds CHANGELOG.md.gz (embed.go), so the overlay replaces that file with the gzipped fixture.
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	if _, err := zw.Write(data); err != nil {
		return ompig, pi, err
	}
	if err := zw.Close(); err != nil {
		return ompig, pi, err
	}
	fixture := filepath.Join(tmp, "CHANGELOG.md.gz")
	if err := os.WriteFile(fixture, compressed.Bytes(), 0o600); err != nil {
		return ompig, pi, err
	}
	overlay, err := json.Marshal(map[string]any{"Replace": map[string]string{filepath.Join(root, "CHANGELOG.md.gz"): fixture}})
	if err != nil {
		return ompig, pi, err
	}
	overlayPath := filepath.Join(tmp, "overlay.json")
	if err := os.WriteFile(overlayPath, overlay, 0o600); err != nil {
		return ompig, pi, err
	}
	ompig.Path = filepath.Join(tmp, "ompig")
	cmd := exec.CommandContext(ctx, "go", "build", "-overlay", overlayPath, "-o", ompig.Path, "./cmd/ompig")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		return ompig, pi, fmt.Errorf("build changelog fixture: %w\n%s", err, out)
	}
	packageRoot, err := piPackageRoot(pi.Path)
	if err != nil {
		return ompig, pi, err
	}
	packageDir := filepath.Join(tmp, "pi-package")
	if err := os.Mkdir(packageDir, 0o700); err != nil {
		return ompig, pi, err
	}
	entries, err := os.ReadDir(packageRoot)
	if err != nil {
		return ompig, pi, err
	}
	for _, entry := range entries {
		if entry.Name() == "CHANGELOG.md" {
			continue
		}
		if err := os.Symlink(filepath.Join(packageRoot, entry.Name()), filepath.Join(packageDir, entry.Name())); err != nil {
			return ompig, pi, err
		}
	}
	if err := os.WriteFile(filepath.Join(packageDir, "CHANGELOG.md"), data, 0o600); err != nil {
		return ompig, pi, err
	}
	pi.Env = slices.DeleteFunc(slices.Clone(pi.Env), func(value string) bool { return strings.HasPrefix(value, "PI_PACKAGE_DIR=") })
	pi.Env = append(pi.Env, "PI_PACKAGE_DIR="+packageDir)
	return ompig, pi, nil
}

// findRepoRoot walks up from CWD looking for go.mod with the ompig module path.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		modfile := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(modfile); err == nil {
			if strings.Contains(string(data), "module github.com/tmih06/ompig") {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repo root not found")
		}
		dir = parent
	}
}
