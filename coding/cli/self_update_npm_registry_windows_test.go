//go:build windows

package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestWindowsNpmInstalledPigUpdatesThroughRealNpm is TestNpmInstalledPigUpdatesThroughRealNpm on Windows: npm's global prefix
// holds node_modules directly and the launcher runs through ompig.cmd, and ompig
// must quarantine its running ompig.exe before npm replaces the launcher tree
// (upstream package-manager-cli.ts prepareWindowsNpmSelfUpdate). The
// quarantine sits in the prefix's node_modules beside the launcher, as
// upstream's sits beside its installed package, so npm replaces the launcher
// without a cleanup warning or a retired directory left behind, and the next
// start removes the quarantine. The new release is this ompig with an overlay,
// so it runs and is told apart from the old one.
func TestWindowsNpmInstalledPigUpdatesThroughRealNpm(t *testing.T) {
	// On Windows "python3" may be the WindowsApps alias that only opens the
	// Store; take the first name that runs.
	python := ""
	for _, name := range []string{"python3", "python"} {
		if path, err := exec.LookPath(name); err == nil && exec.Command(path, "--version").Run() == nil {
			python = path
			break
		}
	}
	if python == "" {
		t.Skip("python is unavailable")
	}
	execPath, err := exec.Command("node", "-p", "process.execPath").Output()
	if err != nil {
		t.Skip("node is unavailable")
	}
	nodeBin := filepath.Dir(strings.TrimSpace(string(execPath)))
	npm := filepath.Join(nodeBin, "npm.cmd")
	if _, err := os.Stat(npm); err != nil {
		t.Fatalf("npm.cmd is not beside node: %v", err)
	}
	ompig := buildPigBinaryForSignalTest(t)
	ompigBytes, err := os.ReadFile(ompig)
	if err != nil {
		t.Fatal(err)
	}
	newBinary := append(slices.Clone(ompigBytes), []byte("\nreleased 9.9.9\n")...)

	registry := newLocalNpmRegistry(t)
	packNpmRelease(t, python, registry, "0.0.1", ompigBytes)
	packNpmRelease(t, python, registry, "9.9.9", newBinary)
	decoy := t.TempDir()
	if err := os.WriteFile(filepath.Join(decoy, "package.json"), []byte(`{"name":"ompig","version":"9.9.9","bin":{"ompig-decoy":"decoy.js"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(decoy, "decoy.js"), []byte("#!/usr/bin/env node\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	registry.publish(t, decoy)

	work := t.TempDir()
	prefix := filepath.Join(work, "prefix")
	for _, dir := range []string{"home", "appdata", "localappdata", "tmp"} {
		if err := os.MkdirAll(filepath.Join(work, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	systemRoot := os.Getenv("SystemRoot")
	env := []string{
		"SystemRoot=" + systemRoot,
		"ComSpec=" + os.Getenv("ComSpec"),
		"PATHEXT=" + os.Getenv("PATHEXT"),
		"USERPROFILE=" + filepath.Join(work, "home"),
		"APPDATA=" + filepath.Join(work, "appdata"),
		"LOCALAPPDATA=" + filepath.Join(work, "localappdata"),
		"TEMP=" + filepath.Join(work, "tmp"),
		"TMP=" + filepath.Join(work, "tmp"),
		"PATH=" + nodeBin + ";" + filepath.Join(systemRoot, "System32") + ";" + systemRoot,
		"npm_config_registry=" + registry.server.URL + "/",
		"npm_config_prefix=" + prefix,
		"npm_config_cache=" + filepath.Join(work, "npm-cache"),
		"npm_config_userconfig=" + filepath.Join(work, "npmrc"),
		"npm_config_globalconfig=" + filepath.Join(work, "global-npmrc"),
		"npm_config_audit=false",
		"npm_config_fund=false",
		"npm_config_update_notifier=false",
	}
	install := exec.Command(npm, "install", "-g", "@ompig/ompig@0.0.1")
	install.Env = env
	if data, err := install.CombinedOutput(); err != nil {
		t.Fatalf("npm install -g @ompig/ompig@0.0.1: %v\n%s", err, data)
	}
	modules := filepath.Join(prefix, "node_modules")
	launcher := filepath.Join(modules, "@ompig", "ompig")
	native := filepath.Join(npmNestedPlatformPackageDir(modules), "ompig.exe")
	if data, err := os.ReadFile(native); err != nil || !bytes.Equal(data, ompigBytes) {
		t.Fatalf("npm did not nest the platform package's binary at %s: %v", native, err)
	}

	generated := t.TempDir()
	if err := os.WriteFile(filepath.Join(generated, "ompig-windows-amd64.exe"), newBinary, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest, err := exec.Command(python, filepath.Join(fixtureSourceRoot, "automation", "release", "gen-update-manifest.py"),
		"--version", "9.9.9", "--base-url", "https://updates.example", "--dir", generated).Output()
	if err != nil {
		t.Fatalf("gen-update-manifest.py: %v", err)
	}
	source := signedManifestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(manifest) }))
	defer source.Close()
	ompigEnv := slices.Concat(env, []string{
		"OMPIG_HOME=" + filepath.Join(work, "ompig-home"),
		"OMPIG_CODING_AGENT_DIR=" + filepath.Join(work, "agent"),
		"PI_CODING_AGENT_DIR=" + filepath.Join(work, "pi-agent"),
		"OMPIG_UPDATE_URL=" + source.URL,
		"OMPIG_UPDATE_TRUST_ROOT=" + os.Getenv("OMPIG_UPDATE_TRUST_ROOT"),
		"OMPIG_UPDATE_ALLOW_LOOPBACK_HTTP=1",
	})
	runPig := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(filepath.Join(prefix, "ompig.cmd"), args...)
		cmd.Dir = work
		cmd.Env = ompigEnv
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("ompig %s: %v\n%s", strings.Join(args, " "), err, output)
		}
		return string(output)
	}

	output := runPig("update", "--self")
	if !strings.Contains(output, "Updated to ompig 9.9.9.") {
		t.Fatalf("ompig update --self output:\n%s", output)
	}
	if strings.Contains(output, "npm warn cleanup") {
		t.Fatalf("npm could not remove the replaced launcher:\n%s", output)
	}
	if data, err := os.ReadFile(native); err != nil || !bytes.Equal(data, newBinary) {
		t.Fatalf("npm did not install the 9.9.9 platform binary at %s: %v\n%s", native, err, output)
	}
	var installed struct{ Version string }
	if data, err := os.ReadFile(filepath.Join(launcher, "package.json")); err != nil || json.Unmarshal(data, &installed) != nil || installed.Version != "9.9.9" {
		t.Fatalf("launcher package after update = %+v (%v)\n%s", installed, err, output)
	}
	// npm logs each invocation's arguments; the update ran exactly these.
	if got, want := npmInvocations(t, filepath.Join(work, "npm-cache")), []string{
		`"install" "--global" "@ompig/ompig@0.0.1"`,
		`"root" "--global"`,
		`"install" "--global" "--ignore-scripts" "--min-release-age" "0" "@ompig/ompig@9.9.9"`,
	}; !slices.Equal(got, want) {
		t.Fatalf("npm invocations:\n got: %q\nwant: %q\n%s", got, want, output)
	}
	if slices.Contains(registry.requested(), "ompig") {
		t.Fatalf("the update asked the registry for the unscoped package ompig: %q\n%s", registry.requested(), output)
	}
	scope, err := os.ReadDir(filepath.Join(modules, "@ompig"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range scope {
		if entry.Name() != "ompig" {
			t.Errorf("npm left %s beside the launcher", filepath.Join(modules, "@ompig", entry.Name()))
		}
	}
	quarantine := filepath.Join(modules, ".ompig-native-quarantine")
	if quarantined, err := filepath.Glob(filepath.Join(quarantine, "*", "ompig.exe")); err != nil || len(quarantined) != 1 {
		t.Fatalf("quarantined executables = %v (err=%v), want the one ompig.exe that ran", quarantined, err)
	}

	runPig("--version")
	if _, err := os.Stat(quarantine); !os.IsNotExist(err) {
		t.Fatalf("quarantine after the next start: %v, want it removed", err)
	}
}
