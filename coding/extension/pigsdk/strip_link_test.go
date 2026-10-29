package pigsdk

import (
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const ompigModule = "github.com/tmih06/ompig"

// stripGoListDeps runs `go list [-tags tags] -deps cmd/ompig`.
func stripGoListDeps(t *testing.T, tags string) []string {
	t.Helper()
	if testing.Short() {
		t.Skip("go list of cmd/ompig in -short mode")
	}
	args := []string{"list"}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	out, err := exec.Command("go", append(args, "-deps", ompigModule+"/cmd/ompig")...).Output()
	if err != nil {
		t.Fatalf("go %s: %v", strings.Join(args, " "), err)
	}
	return strings.Fields(string(out))
}

// D92: a ompig_strip_extension_sdk_rust or ompig_strip_extension_sdk_python build
// of cmd/ompig does not link that SDK's embedded sources; the stock build does.
func TestStripExtensionSDKRustAndPythonBuildOmitsSDKPackages(t *testing.T) {
	stock := stripGoListDeps(t, "")
	for _, tc := range []struct{ tag, pkg string }{
		{"ompig_strip_extension_sdk_rust", ompigModule + "/extensions/sdk-rs"},
		{"ompig_strip_extension_sdk_python", ompigModule + "/extensions/sdk-py"},
	} {
		if !slices.Contains(stock, tc.pkg) {
			t.Errorf("the stock cmd/ompig does not link %s", tc.pkg)
		}
		if slices.Contains(stripGoListDeps(t, tc.tag), tc.pkg) {
			t.Errorf("a %s build of cmd/ompig links %s", tc.tag, tc.pkg)
		}
	}
}

// D92: the Go SDK package stays linked (fused extensions are built on it), so
// a ompig_strip_extension_sdk_go build proves its omission by symbol: pigsdk,
// the only production package that reads the embedded sdk.Source, no longer
// references it, and the linker drops the unreferenced embed. The stock
// package references it.
func TestStripExtensionSDKGoBuildOmitsSDKSource(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the pigsdk package in -short mode")
	}
	const symbol = ompigModule + "/extensions/sdk.Source"
	references := func(tags string) bool {
		archive := filepath.Join(t.TempDir(), "pigsdk.a")
		args := []string{"build", "-o", archive}
		if tags != "" {
			args = append(args, "-tags", tags)
		}
		if out, err := exec.Command("go", append(args, ompigModule+"/coding/extension/pigsdk")...).CombinedOutput(); err != nil {
			t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		out, err := exec.Command("go", "tool", "nm", archive).Output()
		if err != nil {
			t.Fatalf("go tool nm: %v", err)
		}
		return slices.ContainsFunc(strings.Split(string(out), "\n"), func(line string) bool {
			return strings.HasSuffix(strings.TrimSpace(line), " "+symbol)
		})
	}
	if !references("") {
		t.Errorf("the stock pigsdk package does not reference %s", symbol)
	}
	if references("ompig_strip_extension_sdk_go") {
		t.Errorf("a ompig_strip_extension_sdk_go build of pigsdk references %s", symbol)
	}
}

// The host's TypeBox type engine serves the Go, Rust and Python SDKs (the Node SDK runs TypeBox itself), so pigsdk registers it when it embeds any of the three and
// links it for no other reason: a build that strips all three, or a program that links the extension host without pigsdk, links no engine through the host.
func TestTypeEngineIsLinkedOnlyWhileAnSDKIsEmbedded(t *testing.T) {
	if testing.Short() {
		t.Skip("go list of pigsdk in -short mode")
	}
	const engine = ompigModule + "/internal/typebox/engine"
	deps := func(pkg, tags string) []string {
		args := []string{"list"}
		if tags != "" {
			args = append(args, "-tags", tags)
		}
		out, err := exec.Command("go", append(args, "-deps", pkg)...).Output()
		if err != nil {
			t.Fatalf("go %s: %v", strings.Join(args, " "), err)
		}
		return strings.Fields(string(out))
	}
	all := "ompig_strip_extension_sdk_go,ompig_strip_extension_sdk_rust,ompig_strip_extension_sdk_python"
	for _, tc := range []struct {
		name, tags string
		linked     bool
	}{
		{"every SDK embedded", "", true},
		{"only Go stripped", "ompig_strip_extension_sdk_go", true},
		{"only Python embedded", "ompig_strip_extension_sdk_go,ompig_strip_extension_sdk_rust", true},
		{"every SDK stripped", all, false},
	} {
		if got := slices.Contains(deps(ompigModule+"/coding/extension/pigsdk", tc.tags), engine); got != tc.linked {
			t.Errorf("%s: pigsdk links the type engine = %v, want %v", tc.name, got, tc.linked)
		}
	}
	if slices.Contains(deps(ompigModule+"/coding/extension/host/subprocess", ""), engine) {
		t.Errorf("the extension host links the type engine without pigsdk")
	}
}
