package ompiglet

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tmih06/ompig/coding/ompiglet/artifact"
	"github.com/tmih06/ompig/internal/testenv"
)

func TestListPreservesSameNameCrossScopeAndMatchesRecordByDigest(t *testing.T) {
	root := t.TempDir()
	workspace := t.TempDir()
	t.Setenv("OMPIG_HOME", root)
	t.Chdir(workspace)
	workspacePiglet := filepath.Join(workspace, ".ompig", "ompiglets", "same.yaml")
	userPiglet := filepath.Join(root, "ompiglets", "same.yaml")
	for path, description := range map[string]string{workspacePiglet: "workspace", userPiglet: "user"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("name: same\ndescription: "+description+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	userDigest, err := pigletSourceDigest(userPiglet)
	if err != nil {
		t.Fatal(err)
	}
	writeRecordFixtureWithPigletDigest(t, root, "same", strings.TrimPrefix(userDigest, "sha256:"))
	ompiglets, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ompiglets) != 2 || ompiglets[0].Location != "workspace" || ompiglets[1].Location != "user" || len(ompiglets[0].Records) != 0 || len(ompiglets[1].Records) != 1 {
		t.Fatalf("ompiglets = %#v", ompiglets)
	}
}

func TestListMergesSourceAndBinaryRecord(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OMPIG_HOME", root)
	t.Chdir(t.TempDir())
	pigletsDir := filepath.Join(root, "ompiglets")
	if err := os.MkdirAll(pigletsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pigletPath := filepath.Join(pigletsDir, "release.yaml")
	if err := os.WriteFile(pigletPath, []byte("name: release\ndescription: source ompiglet\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	digest, err := pigletSourceDigest(pigletPath)
	if err != nil {
		t.Fatal(err)
	}
	writeRecordFixtureWithPigletDigest(t, root, "release", strings.TrimPrefix(digest, "sha256:"))

	ompiglets, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ompiglets) != 1 || ompiglets[0].Name != "release" || ompiglets[0].Path == "" || len(ompiglets[0].Records) != 1 || ompiglets[0].Records[0].Target != "linux/amd64" {
		t.Fatalf("ompiglets = %#v", ompiglets)
	}
}

func TestListIncludesBinaryOnlyPiglet(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OMPIG_HOME", root)
	t.Chdir(t.TempDir())
	writeRecordFixture(t, root, "binary-only")
	ompiglets, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ompiglets) != 1 || ompiglets[0].Name != "binary-only" || ompiglets[0].Location != "binary" || ompiglets[0].Path != "" || len(ompiglets[0].Records) != 1 {
		t.Fatalf("ompiglets = %#v", ompiglets)
	}
}

// An updated pulled Ompiglet keeps its earlier release beside the new one. Both releases, or two
// builds from different sources, belong to one binary-only Ompiglet: before, each record was its own
// Ompiglet, so `ompig ompiglet list` printed the name twice and `ompig ompiglet show` printed only the first
// release, not the one the update installed. The list names the shared target once.
func TestListGroupsBinaryOnlyRecordsOfOnePiglet(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OMPIG_HOME", root)
	t.Chdir(t.TempDir())
	writeRecordFixtureWithPigletDigest(t, root, "binary-only", strings.Repeat("a", 64))
	writeRecordFixtureWithPigletDigest(t, root, "binary-only", strings.Repeat("b", 64))
	ompiglets, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ompiglets) != 1 || ompiglets[0].Name != "binary-only" || ompiglets[0].Location != "binary" || len(ompiglets[0].Records) != 2 {
		t.Fatalf("ompiglets = %#v", ompiglets)
	}
	var out, errOut strings.Builder
	if code := cmdList(nil, &out, &errOut); code != 0 || strings.Count(out.String(), "binary-only") != 1 || strings.Count(out.String(), "linux/amd64") != 1 {
		t.Fatalf("list code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := cmdShow([]string{"binary-only"}, &out, &errOut); code != 0 {
		t.Fatalf("show code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	for _, record := range ompiglets[0].Records {
		if !strings.Contains(out.String(), "record="+record.Path+"\n") {
			t.Fatalf("show omits the record %s:\n%s", record.Path, out.String())
		}
	}
}

func TestListRejectsTamperedManagedArtifact(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OMPIG_HOME", root)
	t.Chdir(t.TempDir())
	writeRecordFixture(t, root, "tampered")
	artifactRoot := filepath.Join(root, "artifacts", "ompiglets")
	var artifactPath string
	if err := filepath.WalkDir(artifactRoot, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			artifactPath = path
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, []byte("tampered binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := List(); err == nil || (!strings.Contains(err.Error(), "does not match Ompiglet Binary record") && !strings.Contains(err.Error(), "size/type")) {
		t.Fatalf("error = %v", err)
	}
}

func TestListRejectsOrphanManagedArtifact(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OMPIG_HOME", root)
	t.Chdir(t.TempDir())
	artifactPath := filepath.Join(root, "artifacts", "ompiglets", "orphan", "artifact")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, []byte("orphan"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := List(); err == nil || !strings.Contains(err.Error(), "orphan managed artifact") {
		t.Fatalf("error = %v", err)
	}
}

func TestListRejectsRecordAtWrongManagedPath(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OMPIG_HOME", root)
	t.Chdir(t.TempDir())
	binaryPath := writeRecordFixture(t, root, "misplaced")
	wrong := filepath.Join(filepath.Dir(binaryPath), "wrong-name.json")
	if err := os.Rename(binaryPath, wrong); err != nil {
		t.Fatal(err)
	}
	if _, err := List(); err == nil || !strings.Contains(err.Error(), "stored at the wrong path") {
		t.Fatalf("error = %v", err)
	}
}

func TestListRejectsTamperedRecord(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OMPIG_HOME", root)
	t.Chdir(t.TempDir())
	binaryPath := writeRecordFixture(t, root, "tampered-record")
	mutateRecordField(t, binaryPath, func(document map[string]any) {
		document["binary"].(map[string]any)["artifact"].(map[string]any)["digest"] = "sha256:" + strings.Repeat("8", 64)
	})
	if _, err := List(); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestListRejectsBinaryWithMissingResolution(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OMPIG_HOME", root)
	t.Chdir(t.TempDir())
	writeRecordFixture(t, root, "missing-resolution")
	resolutionRoot := filepath.Join(root, "receipts", "ompiglets", "missing-resolution", strings.Repeat("a", 64), "1.0.0", string(artifact.RecordKindResolution))
	entries, err := os.ReadDir(resolutionRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(resolutionRoot, entries[0].Name())); err != nil {
		t.Fatal(err)
	}
	if _, err := List(); err == nil || !strings.Contains(err.Error(), "references missing resolution") {
		t.Fatalf("error = %v", err)
	}
}

func TestListSurfacesComponentPlanIdentity(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OMPIG_HOME", root)
	t.Chdir(t.TempDir())
	writeRecordFixture(t, root, "surfaced")
	ompiglets, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ompiglets) != 1 || len(ompiglets[0].Records) != 1 {
		t.Fatalf("ompiglets = %#v", ompiglets)
	}
	record := ompiglets[0].Records[0]
	if !strings.HasPrefix(record.ComponentPlanDigest, "sha256:") || !strings.HasPrefix(record.ResolutionDigest, "sha256:") || !strings.HasPrefix(record.BinaryDigest, "sha256:") {
		t.Fatalf("record digests = %#v", record)
	}
}

func TestRecordComponentsMapsRealizationAndMaterialization(t *testing.T) {
	plan, err := artifact.BuildPlan([]artifact.ComponentInput{{
		Kind: artifact.ComponentKindExtension, Name: "review", Language: "go", Fusible: true,
		Origin: artifact.Origin{Source: "registry", Version: "1.0.0", Digest: "sha256:" + strings.Repeat("a", 64)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	components := recordComponents(&plan)
	if len(components) != 1 || components[0].Name != "review" || components[0].Realization != "fused" || components[0].Materialization != "binary" {
		t.Fatalf("components = %#v", components)
	}
}

func TestListRejectsMalformedRecord(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OMPIG_HOME", root)
	t.Chdir(t.TempDir())
	binaryPath := writeRecordFixture(t, root, "broken")
	if err := os.WriteFile(binaryPath, []byte(`{"version":1,"kind":"ompiglet-binary","ompiglet":"broken"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := List(); err == nil {
		t.Fatal("invalid Ompiglet record was silently omitted")
	}
}

func mutateRecordField(t *testing.T, path string, mutate func(map[string]any)) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	mutate(document)
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListRejectsMalformedPigletSource(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OMPIG_HOME", root)
	t.Chdir(t.TempDir())
	pigletsDir := filepath.Join(root, "ompiglets")
	if err := os.MkdirAll(pigletsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pigletsDir, "broken.yaml"), []byte("version: nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := List(); err == nil || !strings.Contains(err.Error(), "parse ompiglet YAML") {
		t.Fatalf("error = %v", err)
	}
}

func writeRecordFixture(t *testing.T, root, name string) string {
	t.Helper()
	return writeRecordFixtureWithPigletDigest(t, root, name, strings.Repeat("a", 64))
}

func writeRecordFixtureWithPigletDigest(t *testing.T, root, name, pigletDigest string) string {
	t.Helper()
	return writeRecordFixtureWith(t, root, name, pigletDigest, func(*artifact.BinaryInput) {})
}

// writeRecordFixtureWith writes a managed Binary record whose input edit
// adjusts (for example its strip lists).
func writeRecordFixtureWith(t *testing.T, root, name, pigletDigest string, edit func(*artifact.BinaryInput)) string {
	t.Helper()
	artifactData := []byte("fixture binary")
	artifactSum := sha256.Sum256(artifactData)
	artifactDigest := hex.EncodeToString(artifactSum[:])
	artifactName := "ompig-" + name
	artifactPath := filepath.Join(root, "artifacts", "ompiglets", name, pigletDigest, "1.0.0", "linux-amd64", artifactDigest, artifactName)
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, artifactData, 0o755); err != nil {
		t.Fatal(err)
	}
	plan, err := artifact.BuildPlan(nil)
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := artifact.NewResolutionRecord(name, "1.0.0", time.Unix(1, 0), artifact.ResolutionInput{
		SourceDigest:    "sha256:" + pigletDigest,
		EffectiveDigest: "sha256:" + strings.Repeat("f", 64), ComponentPlan: plan,
	})
	if err != nil {
		t.Fatal(err)
	}
	input := artifact.BinaryInput{
		Target: "linux/amd64", OmpigVersion: "0.1.1", OmpigSourceRevision: "revision",
		OmpigSourceDigest: "sha256:" + strings.Repeat("d", 64), Builder: "native", BuilderIdentity: "native:revision",
		Toolchains:   map[string]string{"go": "go version test"},
		Artifact:     artifact.Artifact{Digest: "sha256:" + artifactDigest, Size: int64(len(artifactData)), FileName: artifactName},
		Verification: artifact.Verification{Policy: "basic", Passed: true, Checks: []string{"artifact-sha256", "artifact-version-smoke"}},
	}
	edit(&input)
	binary, err := artifact.NewBinaryRecord(name, "1.0.0", time.Unix(2, 0), resolution, input)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(root, "receipts", "ompiglets", name, pigletDigest, "1.0.0")
	resolutionPath := filepath.Join(base, string(artifact.RecordKindResolution), strings.TrimPrefix(resolution.Digest, "sha256:")+".json")
	binaryPath := filepath.Join(base, string(artifact.RecordKindBinary), "linux-amd64", strings.TrimPrefix(binary.Digest, "sha256:")+".json")
	for path, record := range map[string]artifact.Record{resolutionPath: resolution, binaryPath: binary} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return binaryPath
}

// (main.go writes ompiglets to ConfigRoot()/ompiglets). Before the fix,
// searchPaths read only $HOME/.ompig/ompiglets, so a ompiglet installed under a
// relocated OMPIG_HOME resolved to nothing and --ompiglet <name> silently loaded
// no ompiglet.
func TestResolve_HonorsPIGHOME(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OMPIG_HOME", root)
	t.Setenv("XDG_CONFIG_HOME", "")

	pigletsDir := filepath.Join(root, "ompiglets")
	if err := os.MkdirAll(pigletsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(pigletsDir, "foo.yaml")
	if err := os.WriteFile(want, []byte("name: foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Resolve("foo")
	if err != nil {
		t.Fatalf("Resolve(foo) under OMPIG_HOME: %v", err)
	}
	if got != want {
		t.Fatalf("Resolve(foo) = %q, want %q", got, want)
	}
}

// XDG_CONFIG_HOME/ompig is the second-priority root and must also be searched.
func TestResolve_HonorsXDGConfigHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("OMPIG_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", root)

	pigletsDir := filepath.Join(root, "ompig", "ompiglets")
	if err := os.MkdirAll(pigletsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(pigletsDir, "bar.yaml")
	if err := os.WriteFile(want, []byte("name: bar\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Resolve("bar")
	if err != nil {
		t.Fatalf("Resolve(bar) under XDG_CONFIG_HOME: %v", err)
	}
	if got != want {
		t.Fatalf("Resolve(bar) = %q, want %q", got, want)
	}
}

// TestAC3PathAnchors pins independent ompiglet:/workspace: anchors across local,
// Package, and inherited source paths, plus missing/escaping workspace failures.
func TestAC3PathAnchors(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	ompiglets := filepath.Join(root, "ompiglets")
	if err := os.MkdirAll(filepath.Join(workspace, ".devcontainer"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(ompiglets, "package"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".devcontainer", "devcontainer.json"), []byte("{\"image\":\"dev:1\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(ompiglets, "base.yaml")
	if err := os.WriteFile(base, []byte(`name: base
packages:
  local: local:./package
systemPrompt:
  file: ompiglet:prompts/system.md
agentEnv:
  devContainer: workspace:.devcontainer/devcontainer.json
`), 0o644); err != nil {
		t.Fatal(err)
	}
	childDir := filepath.Join(root, "child")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(childDir, "child.yaml")
	if err := os.WriteFile(child, []byte("name: child\nextends:\n  source: local:../ompiglets/base.yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveEffectiveWithOptions(child, ResolveOptions{Workspace: workspace})
	if err != nil {
		t.Fatal(err)
	}
	canonicalPiglets, err := canonicalDirectory(ompiglets)
	if err != nil {
		t.Fatal(err)
	}
	canonicalWorkspace, err := canonicalDirectory(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if got := resolved.Ompiglet.SystemPrompt.File; got != filepath.Join(canonicalPiglets, "prompts", "system.md") {
		t.Fatalf("ompiglet anchor = %q", got)
	}
	if got := resolved.Ompiglet.Packages["local"]; got != "local:"+filepath.Join(canonicalPiglets, "package") {
		t.Fatalf("Package source anchor = %q", got)
	}
	environment, err := ResolveAgentEnvironment(resolved.Ompiglet)
	if err != nil {
		t.Fatal(err)
	}
	if environment.Value != filepath.Join(canonicalWorkspace, ".devcontainer", "devcontainer.json") {
		t.Fatalf("workspace anchor = %q", environment.Value)
	}
	otherWorkspace := filepath.Join(root, "other-workspace")
	if err := os.MkdirAll(filepath.Join(otherWorkspace, ".devcontainer"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(otherWorkspace, ".devcontainer", "devcontainer.json"), []byte("{\"image\":\"dev:1\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	other, err := ResolveEffectiveWithOptions(child, ResolveOptions{Workspace: otherWorkspace})
	if err != nil {
		t.Fatal(err)
	}
	if other.EffectiveDigest != resolved.EffectiveDigest || other.GraphDigest != resolved.GraphDigest {
		t.Fatalf("machine-local workspace changed portable identity: first=%s/%s second=%s/%s", resolved.EffectiveDigest, resolved.GraphDigest, other.EffectiveDigest, other.GraphDigest)
	}
	if _, err := ResolveEffective(child); err == nil || !strings.Contains(err.Error(), "requires a workspace anchor") {
		t.Fatalf("missing workspace error = %v", err)
	}

	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	testenv.RequireDirectoryLink(t, outside, filepath.Join(workspace, "escape"))
	escaping := filepath.Join(ompiglets, "escaping.yaml")
	if err := os.WriteFile(escaping, []byte("name: escaping\nagentEnv:\n  devContainer: workspace:escape/devcontainer.json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveEffectiveWithOptions(escaping, ResolveOptions{Workspace: workspace}); err == nil || !strings.Contains(err.Error(), "outside the Ompiglet anchor") {
		// ensureNoSymlinkEscape uses a generic anchor error; the field context
		// still identifies agentEnv.devContainer.
		if err == nil || !strings.Contains(err.Error(), "outside") {
			t.Fatalf("workspace symlink error = %v", err)
		}
	}
}
