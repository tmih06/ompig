//go:build parity

package runner

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmih06/ompig/coding"
)

// A parity run that cannot find its comparator or its ompig must fail, not skip: a skipped TestParity exits 0 with zero scenarios compared, which `make parity-family` used to report as ok. OMPIG_PARITY_ALLOW_ZERO=1 is the explicit request to accept that outcome, the same opt-out require-parity-ran honors.
const missingBinaryHelperEnv = "OMPIG_TEST_MISSING_BINARY_HELPER"

// TestMissingBinaryHelperProcess is a subprocess entry point, not a test: a failing or skipping subtest cannot be observed in-process.
func TestMissingBinaryHelperProcess(t *testing.T) {
	switch os.Getenv(missingBinaryHelperEnv) {
	case "pi":
		ResolveUpstreamPiBin(t)
	case "ompig":
		ResolvePigBin(t)
	default:
		t.Skip("helper process entry point")
	}
}

func runMissingBinaryHelper(t *testing.T, resolver string, extraEnv ...string) helperOutcome {
	t.Helper()
	overridden := map[string]bool{missingBinaryHelperEnv: true}
	for _, kv := range extraEnv {
		key, _, _ := strings.Cut(kv, "=")
		overridden[key] = true
	}
	env := []string{missingBinaryHelperEnv + "=" + resolver}
	for _, kv := range os.Environ() {
		if key, _, _ := strings.Cut(kv, "="); !overridden[key] {
			env = append(env, kv)
		}
	}
	env = append(env, extraEnv...)
	cmd := exec.Command(os.Args[0], "-test.run", "^TestMissingBinaryHelperProcess$", "-test.v")
	cmd.Env = env
	if resolver == "pi" {
		// The resolver finds the source-built oracle by walking up from the working directory to extensions/sdk-ts; a directory outside the repository has none.
		cmd.Dir = t.TempDir()
	}
	out, err := cmd.CombinedOutput()
	outcome := helperOutcome{output: string(out)}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		outcome.skipped = strings.Contains(outcome.output, "--- SKIP")
		outcome.passed = !outcome.skipped
	case errors.As(err, &exitErr):
		outcome.failed = true
	default:
		t.Fatalf("run missing-binary helper: %v\n%s", err, out)
	}
	return outcome
}

func emptyHomeEnv(t *testing.T) []string {
	home := t.TempDir()
	return []string{"HOME=" + home, "USERPROFILE=" + home, "OMPIG_PARITY_PI_BIN=", "OMPIG_PARITY_REAL_AUTH="}
}

func TestResolveUpstreamPiBinFailsWhenNotInstalledByDefault(t *testing.T) {
	outcome := runMissingBinaryHelper(t, "pi", append(emptyHomeEnv(t), "OMPIG_PARITY_ALLOW_ZERO=")...)
	if !outcome.failed || outcome.skipped || outcome.passed {
		t.Fatalf("no installed pi: %+v, want failed=true (a missing comparator must not skip the parity run)\n%s", outcome, outcome.output)
	}
}

func TestResolveUpstreamPiBinFailsWhenOverrideIsMissingByDefault(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-pi")
	outcome := runMissingBinaryHelper(t, "pi", "OMPIG_PARITY_PI_BIN="+missing, "OMPIG_PARITY_ALLOW_ZERO=")
	if !outcome.failed || outcome.skipped || outcome.passed {
		t.Fatalf("missing OMPIG_PARITY_PI_BIN: %+v, want failed=true\n%s", outcome, outcome.output)
	}
}

func TestResolveUpstreamPiBinMissingSkipsOnlyWithExplicitOptOut(t *testing.T) {
	outcome := runMissingBinaryHelper(t, "pi", append(emptyHomeEnv(t), "OMPIG_PARITY_ALLOW_ZERO=1")...)
	if outcome.failed || !outcome.skipped {
		t.Fatalf("no installed pi with OMPIG_PARITY_ALLOW_ZERO=1: %+v, want skipped=true\n%s", outcome, outcome.output)
	}
}

func TestResolvePigBinFailsWhenUnsetByDefault(t *testing.T) {
	outcome := runMissingBinaryHelper(t, "ompig", "OMPIG_PARITY_OMPIG_BIN=", "OMPIG_BIN=", "OMPIG_PARITY_ALLOW_ZERO=")
	if !outcome.failed || outcome.skipped || outcome.passed {
		t.Fatalf("no ompig binary under test: %+v, want failed=true\n%s", outcome, outcome.output)
	}
}

func TestResolvePigBinFailsWhenMissingByDefault(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-ompig")
	outcome := runMissingBinaryHelper(t, "ompig", "OMPIG_PARITY_OMPIG_BIN="+missing, "OMPIG_BIN=", "OMPIG_PARITY_ALLOW_ZERO=")
	if !outcome.failed || outcome.skipped || outcome.passed {
		t.Fatalf("missing ompig binary: %+v, want failed=true\n%s", outcome, outcome.output)
	}
}

func TestResolvePigBinMissingSkipsOnlyWithExplicitOptOut(t *testing.T) {
	outcome := runMissingBinaryHelper(t, "ompig", "OMPIG_PARITY_OMPIG_BIN=", "OMPIG_BIN=", "OMPIG_PARITY_ALLOW_ZERO=1")
	if outcome.failed || !outcome.skipped {
		t.Fatalf("no ompig binary with OMPIG_PARITY_ALLOW_ZERO=1: %+v, want skipped=true\n%s", outcome, outcome.output)
	}
}

// A published Pi that mise installed must not become the oracle on its own: a run without OMPIG_PARITY_PI_PUBLISHED=1 and without a
// source-built oracle fails instead of comparing against the published release (the silent fallback an unset OMPIG_PARITY_PI_BIN used to take).
func TestResolveUpstreamPiBinNeverFallsBackToAMiseInstallWithoutOptIn(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "share", "mise", "installs", upstreamPiInstallDirs[0], coding.UpstreamPackageVersion, "bin", "pi")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho "+coding.UpstreamPackageVersion+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	env := []string{"HOME=" + home, "USERPROFILE=" + home, "OMPIG_PARITY_PI_BIN=", "OMPIG_PARITY_REAL_AUTH=", "OMPIG_PARITY_ALLOW_ZERO="}
	if outcome := runMissingBinaryHelper(t, "pi", append(env, "OMPIG_PARITY_PI_PUBLISHED=")...); !outcome.failed {
		t.Fatalf("a mise-installed Pi was used without the opt-in: %+v\n%s", outcome, outcome.output)
	}
	if coding.UpstreamVersion == coding.UpstreamPackageVersion {
		if outcome := runMissingBinaryHelper(t, "pi", append(env, "OMPIG_PARITY_PI_PUBLISHED=1")...); outcome.failed {
			t.Fatalf("the mise-installed Pi was refused despite the opt-in: %+v\n%s", outcome, outcome.output)
		}
	}
}
