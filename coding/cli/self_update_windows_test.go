//go:build windows && !ompig_strip_self_update

package cli

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmih06/ompig/internal/codingagent"
)

// On Windows a standalone installation resolves to the unsupported
// self-update tier (D39): upstream self-updates only npm, pnpm and managed
// installs there, and a running Windows executable is not replaced in place.
// An explicit update of a receipted standalone ompig.exe with a newer release
// available is refused with reinstall guidance, and nothing is replaced.
func TestSelfUpdateOnWindowsRefusesReceiptedStandalone(t *testing.T) {
	manSrv, binSrv := newerManifest(t, []byte("new"))
	defer manSrv.Close()
	defer binSrv.Close()
	seedRunningStandaloneReceipt(t, manSrv.URL)
	t.Setenv("OMPIG_INSTALL_TIER", "")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(exe)
	if err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := captureStdoutStderr(t, func() int { return runSelfUpdate(false) })
	if code != 1 || stdout != "" {
		t.Fatalf("code = %d, stdout = %q; want 1 and no output", code, stdout)
	}
	if !strings.Contains(stderr, "cannot self-update this installation") || !strings.Contains(stderr, "Executable:") {
		t.Fatalf("stderr missing reinstall remediation:\n%s", stderr)
	}
	after, err := os.Stat(exe)
	if err != nil || !after.ModTime().Equal(before.ModTime()) || after.Size() != before.Size() {
		t.Fatalf("running executable changed: before=%v after=%v err=%v", before, after, err)
	}

	target := filepath.Join(t.TempDir(), "ompig.exe")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := applyStandaloneUpdate(target, false); err == nil || !strings.Contains(err.Error(), "not supported on Windows") {
		t.Fatalf("applyStandaloneUpdate error = %v, want the Windows refusal", err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "old" {
		t.Fatalf("standalone target changed: %q (err=%v)", got, err)
	}
}

// Upstream self-updates an npm install on Windows (package-manager-cli.ts)
// after quarantining the native images it loaded from the package directory
// (prepareWindowsNpmSelfUpdate), because Windows refuses to delete a running
// image. ompig's running image there is ompig.exe itself, in the platform package
// npm nests inside the launcher: without the quarantine npm cannot remove the
// launcher tree ompig runs from. The quarantine sits in the global node_modules
// beside the launcher, outside that tree, as upstream's sits beside its
// installed package. The next start on Windows clears it.
func TestWindowsNpmSelfUpdateReplacesTheRunningInstallation(t *testing.T) {
	manifest := `{"version":"9.9.9","packageName":"@ompig/ompig","binaries":{}}`
	srv := signedManifestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(manifest))
	}))
	defer srv.Close()
	t.Setenv("OMPIG_UPDATE_URL", srv.URL)
	t.Setenv("OMPIG_INSTALL_TIER", "")

	root := filepath.Join(t.TempDir(), "node_modules")
	ompigExe := filepath.Join(npmNestedPlatformPackageDir(root), "ompig.exe")
	tools := t.TempDir()
	copyTestBinary(t, ompigExe)
	copyTestBinary(t, filepath.Join(tools, "npm.exe"))

	update := exec.Command(ompigExe)
	update.Dir = t.TempDir()
	update.Env = append(os.Environ(), npmUpdateFixtureRootEnv+"="+root, "PATH="+tools)
	out, err := update.CombinedOutput()
	if err != nil {
		t.Fatalf("ompig update in an npm install: %v\n%s", err, out)
	}
	got, err := os.ReadFile(ompigExe)
	if err != nil || string(got) != "install -g --ignore-scripts --min-release-age=0 @ompig/ompig@9.9.9" {
		t.Fatalf("npm did not replace the package: %q (err=%v)\n%s", got, err, out)
	}
	quarantine := filepath.Join(root, ".ompig-native-quarantine")
	quarantined, err := filepath.Glob(filepath.Join(quarantine, "*", "ompig.exe"))
	if err != nil || len(quarantined) != 1 {
		t.Fatalf("quarantined executables = %v (err=%v), want the one ompig.exe that ran", quarantined, err)
	}
	codingagent.CleanupWindowsSelfUpdateQuarantine(filepath.Dir(ompigExe))
	if _, err := os.Stat(quarantine); !os.IsNotExist(err) {
		t.Fatalf("quarantine after cleanup: %v, want it removed", err)
	}
}
