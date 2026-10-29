package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/tmih06/ompig/internal/testenv"

	"github.com/tmih06/ompig/coding/extension/host/subprocess"
	"github.com/tmih06/ompig/coding/ompiglet"
	"github.com/tmih06/ompig/internal/codingagent"
)

func TestStockPigDependencyGraphExcludesProductResources(t *testing.T) {
	root := testenv.ModuleRoot(t)
	cmd := exec.Command("go", "list", "-deps", "./cmd/ompig")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list Stock Ompig dependencies: %v\n%s", err, output)
	}
	for _, forbidden := range []string{
		"github.com/tmih06/ompig/ompiglets/standard",
		"github.com/tmih06/ompig/easter-eggs",
	} {
		if strings.Contains(string(output), forbidden) {
			t.Fatalf("Stock Ompig dependency graph contains product package %q", forbidden)
		}
	}
}

func TestOmpigStandardRequiresAndResolvesOnlyFusedExtensions(t *testing.T) {
	root := testenv.ModuleRoot(t)
	standardPath := filepath.Join(root, "ompiglets", "standard", "ompig-standard.yaml")
	standard, err := ompiglet.Parse(standardPath)
	if err != nil {
		t.Fatalf("parse Ompig Standard: %v", err)
	}
	if standard.Build == nil || standard.Build.ExtensionRealization != "fused" {
		t.Fatalf("Ompig Standard build.extensionRealization = %#v, want fused", standard.Build)
	}
	if standard.Release == nil || standard.Release.Version != OmpigVersion+"-dev" && standard.Release.Version != OmpigVersion {
		t.Fatalf("Ompig Standard release.version = %#v, want %q or %q", standard.Release, OmpigVersion+"-dev", OmpigVersion)
	}
	standardExtensions := make(map[string]bool, len(standard.Extensions))
	for _, entry := range standard.Extensions {
		standardExtensions[entry.Name] = true
	}
	for _, required := range []string{"piglogin", "pigrunner"} {
		if !standardExtensions[required] {
			t.Fatalf("Ompig Standard does not select required extension %q", required)
		}
	}
	resolved, resolutionErrors := ompiglet.ResolveExtensions(standard)
	if len(resolutionErrors) > 0 {
		t.Fatalf("resolve Ompig Standard extensions: %v", resolutionErrors)
	}
	if len(resolved) != len(standard.Extensions) {
		t.Fatalf("resolved %d of %d Ompig Standard extensions", len(resolved), len(standard.Extensions))
	}
	configs := make([]subprocess.ExtConfig, 0, len(resolved))
	for _, extension := range resolved {
		config, _, err := subprocess.ResolveExtConfigWithIdentity(extension.Path, extension.Entry.Name)
		if err != nil {
			t.Fatalf("resolve Ompig Standard extension %q: %v", extension.Entry.Name, err)
		}
		config.Enabled = true
		configs = append(configs, config)
	}
	cells := subprocess.PlanCells(configs, nil)
	planned := 0
	for _, cell := range cells {
		planned += len(cell.Extensions)
		if cell.Strategy != subprocess.CellStrategyPackedGo {
			t.Errorf("Ompig Standard cell %q uses %s, want fused-compatible packed-go", cell.Key, cell.Strategy)
		}
	}
	if planned != len(standard.Extensions) {
		t.Fatalf("planned %d of %d Ompig Standard extensions", planned, len(standard.Extensions))
	}
}

func TestStockPigDoesNotRegisterStandardCommands(t *testing.T) {
	commands := map[string]bool{}
	for _, command := range codingagent.BuiltinSlashCommands() {
		commands[command.Name] = true
	}
	for _, productCommand := range []string{"sprite", "runner", "ompig-runner", "onboard", "harness"} {
		if commands[productCommand] {
			t.Fatalf("Stock Ompig registers Ompig Standard command %q", productCommand)
		}
	}
	for _, args := range [][]string{{"marketplace"}, {"runner"}} {
		if code := runPigPreSessionCommand(args); code != -1 {
			t.Fatalf("Stock Ompig handled product command %q with code %d", args[0], code)
		}
	}
}

func TestRawPigContainsNoUnnamespacedAdditiveState(t *testing.T) {
	root := testenv.ModuleRoot(t)
	forbidden := []string{
		"platform.json", "marketplace-credentials.json", "marketplaces.json",
		"capability-index.json", "harness-overlay", "harness.json",
		"web-search.json", "source/extensions/sdk", "PLATFORM_DIR",
		`ConfigRoot(), "builders.json"`, `ConfigRoot(), "plugins"`,
	}
	var found []string
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".upstream" || entry.Name() == "testdata" || entry.Name() == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, phrase := range forbidden {
			if strings.Contains(string(data), phrase) {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					rel = path
				}
				found = append(found, filepath.ToSlash(rel)+": "+phrase)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("scan raw Ompig sources: %v", err)
	}
	if len(found) > 0 {
		t.Fatalf("raw Ompig contains unnamespaced additive state: %s", strings.Join(found, ", "))
	}
}

// TestRawPigContainsNoPlatformAPIPaths locks the product boundary: platform
// HTTP routes belong to external extensions, not Stock Ompig.
func TestRawPigContainsNoPlatformAPIPaths(t *testing.T) {
	root := testenv.ModuleRoot(t)
	var found []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".upstream" || entry.Name() == "testdata" || entry.Name() == "tmp" {
				return filepath.SkipDir
			}
			// The Kubernetes benchmark runner is not part of the Ompig binary and talks to the Kubernetes API
			// (kubectl get --raw /api/v1/nodes/...), not to a platform HTTP route.
			if rel, relErr := filepath.Rel(root, path); relErr == nil && filepath.ToSlash(rel) == "bench/k8s" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		containsPlatformPath, err := containsPlatformAPIPath(data)
		if err != nil {
			return err
		}
		if containsPlatformPath {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				rel = path
			}
			found = append(found, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan raw Ompig sources: %v", err)
	}
	if len(found) > 0 {
		t.Fatalf("raw Ompig contains platform API paths in production sources: %s", strings.Join(found, ", "))
	}
}

func containsPlatformAPIPath(source []byte) (bool, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "source.go", source, 0)
	if err != nil {
		return false, err
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, unquoteErr := strconv.Unquote(literal.Value)
		if unquoteErr != nil || !strings.Contains(value, "/api/v1/") {
			return true
		}
		parsed, parseErr := url.Parse(value)
		if parseErr == nil && parsed.IsAbs() && parsed.Host != "" {
			return true
		}
		found = true
		return false
	})
	return found, nil
}

func TestContainsPlatformAPIPathDistinguishesExternalURLs(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		wanted bool
	}{
		{name: "relative platform route", value: `package p; const route = "/api/v1/agents"`, wanted: true},
		{name: "external provider URL", value: `package p; const endpoint = "https://openrouter.ai/api/v1/auth/keys"`},
		{name: "comment only", value: "package p // /api/v1/agents\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := containsPlatformAPIPath([]byte(test.value))
			if err != nil {
				t.Fatalf("containsPlatformAPIPath: %v", err)
			}
			if got != test.wanted {
				t.Fatalf("containsPlatformAPIPath = %v, want %v", got, test.wanted)
			}
		})
	}
}
