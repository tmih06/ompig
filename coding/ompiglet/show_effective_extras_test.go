package ompiglet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The effective view of a resolved Ompiglet keeps the resolved local paths and
// the source metadata that lets them validate, and revalidates cleanly.
func TestPigletWithEffectiveDefaultsPreservesResolvedSourceMetadata(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "ext"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "ompiglet.yaml")
	if err := os.WriteFile(source, []byte("name: demo\nextensions:\n  - name: demo\n    origins: [local:./ext]\nagentEnv:\n  image: ghcr.io/acme/dev:1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolution, err := ResolveEffectiveWithOptions(source, ResolveOptions{Workspace: root})
	if err != nil {
		t.Fatal(err)
	}
	effective, err := pigletWithEffectiveDefaults(resolution.Ompiglet)
	if err != nil {
		t.Fatal(err)
	}
	if effective.SourcePath() != resolution.Ompiglet.SourcePath() || effective.SourcePath() == "" {
		t.Fatalf("source path = %q, want %q", effective.SourcePath(), resolution.Ompiglet.SourcePath())
	}
	if effective.AgentEnv == nil || effective.AgentEnv.OmpigRuntime == nil || effective.AgentEnv.OmpigRuntime.Mode != "inject" || effective.AgentEnv.Policy == nil || effective.AgentEnv.Policy.Preset != "standard" {
		t.Fatalf("effective agent environment defaults missing: %#v", effective.AgentEnv)
	}
	if resolution.Ompiglet.AgentEnv.OmpigRuntime != nil {
		t.Fatal("effective view mutated the resolved Ompiglet")
	}
	if err := effective.Validate(); err != nil {
		t.Fatalf("effective Ompiglet does not revalidate: %v", err)
	}
	if effective == resolution.Ompiglet {
		t.Fatal("effective view aliases the resolved Ompiglet")
	}
}

// Issue #98 through extends: each lineage source anchors its own local
// origins before the merge, so the effective view holds absolute origins from
// two directories plus an inherited package alias. Every one must render, in
// both output modes, at the exact anchored path.
func TestRunCommandShowEffectiveAcceptsInheritedLocalPaths(t *testing.T) {
	t.Setenv("OMPIG_HOME", t.TempDir())
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	for _, dir := range []string{"base/ext", "base/package", "base/skills/helper", "child/ext"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"base/ompiglet.yaml":  "name: base\nrelease:\n  version: 1.0.0\npackages:\n  shared: local:./package\nextensions:\n  - name: basic\n    origins: [local:./ext]\nskills:\n  - name: helper\n    origins: [local:./skills/helper]\n",
		"child/ompiglet.yaml": "name: child\nextends:\n  source: local:../base/ompiglet.yaml\n  version: ^1.0.0\nextensions:\n  - name: extra\n    origins: [package:shared, local:./ext]\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	local := func(path string) string { return "local:" + filepath.Join(root, filepath.FromSlash(path)) }
	source := filepath.Join(root, "child", "ompiglet.yaml")

	var stdout, stderr strings.Builder
	if code := RunCommand([]string{"ompiglet", "show", source, "--effective"}, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("human: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	for _, line := range []string{
		"Ompiglet: child\n",
		"  basic  tools=all  origin=" + local("base/ext") + "\n",
		"  extra  tools=all  origin=package:shared\n",
		"  helper  origin=" + local("base/skills/helper") + "\n",
	} {
		if !strings.Contains(stdout.String(), line) {
			t.Fatalf("human output lacks %q:\n%s", line, stdout.String())
		}
	}

	stdout.Reset()
	if code := RunCommand([]string{"ompiglet", "show", source, "--effective", "--json"}, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("json: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var document struct {
		Effective bool `json:"effective"`
		Ompiglet  struct {
			Name       string            `json:"name"`
			Packages   map[string]string `json:"packages"`
			Extensions []struct {
				Name    string   `json:"name"`
				Origins []string `json:"origins"`
			} `json:"extensions"`
			Skills []struct {
				Name    string   `json:"name"`
				Origins []string `json:"origins"`
			} `json:"skills"`
		} `json:"ompiglet"`
	}
	if err := json.Unmarshal([]byte(stdout.String()), &document); err != nil {
		t.Fatalf("decode: %v\n%s", err, stdout.String())
	}
	got := document.Ompiglet
	if !document.Effective || got.Name != "child" || len(got.Packages) != 1 || got.Packages["shared"] != local("base/package") ||
		len(got.Extensions) != 2 || got.Extensions[0].Name != "basic" || !slices.Equal(got.Extensions[0].Origins, []string{local("base/ext")}) ||
		got.Extensions[1].Name != "extra" || !slices.Equal(got.Extensions[1].Origins, []string{"package:shared", local("child/ext")}) ||
		len(got.Skills) != 1 || got.Skills[0].Name != "helper" || !slices.Equal(got.Skills[0].Origins, []string{local("base/skills/helper")}) {
		t.Fatalf("effective JSON = %s", stdout.String())
	}
}
