package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	ompiglet "github.com/tmih06/ompig/coding/ompiglet"
)

// TestResolveAndLoadSkills_PigletSkillsUnionWithMarketplace is the regression
// test for the dual bug:
//  1. loadSkills was called with NoSkills=true (kill-switch) → zero skills
//  2. ompiglet branch replaced skillInputs (discarding marketplace skills)
//
// After the fix: ompiglet skills are loaded IN ADDITION TO marketplace skills.
// If a ompiglet skill has the same name as a marketplace skill, the ompiglet
// version wins. The result is a union, not a replacement.
func TestResolveAndLoadSkills_PigletSkillsUnionWithMarketplace(t *testing.T) {
	t.Setenv("OMPIG_HOME", t.TempDir())
	root := t.TempDir()

	// Marketplace skills (simulating what collectSkillInputs returns)
	marketSkillPaths := []string{
		makeSkillDir(t, root, "commit"),
		makeSkillDir(t, root, "github"),
		makeSkillDir(t, root, "browser-cdp"),
	}
	// Ompiglet skills: 2 overlap with marketplace (commit, github), 1 is new
	pigletSkillDir := makeSkillDir(t, root, "llm-wiki")
	p := &ompiglet.Ompiglet{
		Name: "test-ompiglet",
		Skills: []ompiglet.SkillEntry{
			{Name: "commit", Origins: []string{"local:" + marketSkillPaths[0]}},
			{Name: "github", Origins: []string{"local:" + marketSkillPaths[1]}},
			{Name: "llm-wiki", Origins: []string{"local:" + pigletSkillDir}},
		},
	}

	slr, err := resolveAndLoadSkills(p, marketSkillPaths)
	if err != nil {
		t.Fatalf("resolveAndLoadSkills error: %v", err)
	}

	// Union: 3 marketplace + 1 ompiglet-only (llm-wiki) - 2 overlaps = 4 unique
	if len(slr.Defs) != 4 {
		t.Fatalf("len(Defs) = %d, want 4 (union of 3 marketplace + 1 ompiglet-only, deduped by name)\nDefs: %v", len(slr.Defs), slr.Defs)
	}

	names := map[string]bool{}
	for _, s := range slr.Defs {
		names[s.Name] = true
	}
	for _, want := range []string{"commit", "github", "browser-cdp", "llm-wiki"} {
		if !names[want] {
			t.Errorf("missing skill %q in union set %v", want, names)
		}
	}

	// Paths should be the union too (for /reload)
	if len(slr.Paths) != 4 {
		t.Errorf("len(Paths) = %d, want 4", len(slr.Paths))
	}
}

// TestResolveAndLoadSkills_NilPigletUsesCollectedInputs verifies the bare-ompig
// path (no ompiglet): collected skill inputs from packages/convention dirs load.
func TestResolveAndLoadSkills_NilPigletUsesCollectedInputs(t *testing.T) {
	t.Setenv("OMPIG_HOME", t.TempDir())
	root := t.TempDir()
	collected := []string{
		makeSkillDir(t, root, "pkg-skill-a"),
		makeSkillDir(t, root, "pkg-skill-b"),
	}
	slr, err := resolveAndLoadSkills(nil, collected)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(slr.Defs) != 2 {
		t.Fatalf("len(Defs) = %d, want 2", len(slr.Defs))
	}
}

// TestResolveAndLoadSkills_InlineContentSkills verifies baked-in Content skills
// load without filesystem I/O and are included in the union.
func TestResolveAndLoadSkills_InlineContentSkills(t *testing.T) {
	t.Setenv("OMPIG_HOME", t.TempDir())
	p := &ompiglet.Ompiglet{
		Name: "inline-ompiglet",
		Skills: []ompiglet.SkillEntry{
			{Name: "baked", Description: "inline", Content: "# baked skill body"},
		},
	}
	slr, err := resolveAndLoadSkills(p, nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(slr.Defs) != 1 {
		t.Fatalf("len(Defs) = %d, want 1", len(slr.Defs))
	}
	if slr.Defs[0].Name != "baked" {
		t.Errorf("Defs[0].Name = %q, want baked", slr.Defs[0].Name)
	}
	if slr.Defs[0].Body != "# baked skill body" {
		t.Errorf("Defs[0].Body = %q, want inline content", slr.Defs[0].Body)
	}
}

// TestResolveAndLoadSkills_ResolvedPigletPath verifies an effective Ompiglet's
// already-anchored absolute skill path loads without additional expansion.
func TestResolveAndLoadSkills_ResolvedPigletPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OMPIG_HOME", home)

	skillDir := filepath.Join(home, ".ompig", "skills", "my-wiki")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: my-wiki\ndescription: wiki\n---\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := &ompiglet.Ompiglet{
		Name: "tilde-ompiglet",
		Skills: []ompiglet.SkillEntry{
			{Name: "my-wiki", Origins: []string{"local:" + skillDir}},
		},
	}
	slr, err := resolveAndLoadSkills(p, nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(slr.Defs) != 1 || slr.Defs[0].Name != "my-wiki" {
		t.Fatalf("Defs = %v, want [my-wiki]", slr.Defs)
	}
}

// TestResolveAndLoadSkills_PigletWinsOnNameConflict verifies that when a
// ompiglet skill and a marketplace skill have the same name but different
// paths, the ompiglet version wins.
func TestResolveAndLoadSkills_PigletWinsOnNameConflict(t *testing.T) {
	t.Setenv("OMPIG_HOME", t.TempDir())
	root := t.TempDir()

	// Marketplace "commit" skill (named 'commit' in frontmatter, at market/ path)
	marketCommit := filepath.Join(root, "market", "commit")
	if err := os.MkdirAll(marketCommit, 0o755); err != nil {
		t.Fatal(err)
	}
	marketBody := "---\nname: commit\ndescription: market version\n---\n# market commit\n"
	if err := os.WriteFile(filepath.Join(marketCommit, "SKILL.md"), []byte(marketBody), 0o644); err != nil {
		t.Fatal(err)
	}
	// Ompiglet "commit" skill (same name, different path)
	pigletCommit := filepath.Join(root, "ompiglet-commit")
	if err := os.MkdirAll(pigletCommit, 0o755); err != nil {
		t.Fatal(err)
	}
	pigletBody := "---\nname: commit\ndescription: ompiglet version\n---\n# ompiglet commit\n"
	if err := os.WriteFile(filepath.Join(pigletCommit, "SKILL.md"), []byte(pigletBody), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &ompiglet.Ompiglet{
		Name: "conflict-ompiglet",
		Skills: []ompiglet.SkillEntry{
			{Name: "commit", Origins: []string{"local:" + pigletCommit}},
		},
	}
	slr, err := resolveAndLoadSkills(p, []string{marketCommit})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Should have exactly 1 "commit" skill (not 2)
	if len(slr.Defs) != 1 {
		t.Fatalf("len(Defs) = %d, want 1 (deduped by name)", len(slr.Defs))
	}
	// And it should be the ompiglet version
	if slr.Defs[0].FilePath != filepath.Join(pigletCommit, "SKILL.md") {
		t.Errorf("Defs[0].Path = %q, want %q (ompiglet version should win)", slr.Defs[0].FilePath, pigletCommit)
	}
}

// TestResolveAndLoadSkills_BugRegression_PassingNoSkillsKillsLoad is a
// documentation test that proves the original bug: if loadSkills is called
// with noSkills=true, zero skills load. This test documents the behavior that
// resolveAndLoadSkills must NOT do.
func TestResolveAndLoadSkills_BugRegression_PassingNoSkillsKillsLoad(t *testing.T) {
	t.Setenv("OMPIG_HOME", t.TempDir())
	root := t.TempDir()
	skillPath := makeSkillDir(t, root, "alpha")

	// The bug: passing true as noSkills short-circuits to nil
	bugged, _, err := loadSkills([]string{skillPath}, true)
	if err != nil {
		t.Fatalf("loadSkills(true) error: %v", err)
	}
	if len(bugged) != 0 {
		t.Fatalf("buggy loadSkills(noSkills=true) = %d skills, want 0 (documents pre-fix behavior)", len(bugged))
	}

	// The fix: passing false loads the skill
	got, _, err := loadSkills([]string{skillPath}, false)
	if err != nil {
		t.Fatalf("loadSkills(false) error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("loadSkills(noSkills=false) = %d skills, want 1", len(got))
	}
}

func TestResolveAndLoadSkills_UnresolvedPigletSkillFailsClosed(t *testing.T) {
	p := &ompiglet.Ompiglet{
		Name:   "missing",
		Skills: []ompiglet.SkillEntry{{Name: "missing", Origins: []string{"local:" + filepath.Join(t.TempDir(), "absent")}}},
	}
	if _, err := resolveAndLoadSkills(p, nil); err == nil || !strings.Contains(err.Error(), "resolve Ompiglet skills") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveAndLoadSkills_InlinePigletWinsOnNameConflict(t *testing.T) {
	root := t.TempDir()
	market := makeSkillDir(t, root, "commit")
	p := &ompiglet.Ompiglet{
		Name:   "inline-wins",
		Skills: []ompiglet.SkillEntry{{Name: "commit", Description: "inline", Content: "INLINE OMPIGLET BODY"}},
	}
	result, err := resolveAndLoadSkills(p, []string{market})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Defs) != 1 || result.Defs[0].Name != "commit" || result.Defs[0].Body != "INLINE OMPIGLET BODY" {
		t.Fatalf("defs = %#v", result.Defs)
	}
}

func TestLoadSkillsSkipsMissingDescriptionAndMalformedSibling(t *testing.T) {
	root := t.TempDir()
	valid := makeSkillDir(t, root, "valid")
	missing := filepath.Join(root, "missing")
	malformed := filepath.Join(root, "malformed")
	for _, dir := range []string{missing, malformed} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(missing, "SKILL.md"), []byte("---\nname: missing\n---\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(malformed, "SKILL.md"), []byte("---\ndescription: [bad\n---\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _, err := loadSkills([]string{malformed, missing, valid}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "valid" {
		t.Fatalf("skills = %#v", got)
	}
}

func TestLoadSkillsUsesFirstSameNameDefinition(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "package", "review")
	last := filepath.Join(root, "workspace", "review")
	for _, item := range []struct {
		path, description string
	}{{first, "package copy"}, {last, "workspace copy"}} {
		if err := os.MkdirAll(item.path, 0o755); err != nil {
			t.Fatal(err)
		}
		body := "---\nname: review\ndescription: " + item.description + "\n---\nbody\n"
		if err := os.WriteFile(filepath.Join(item.path, "SKILL.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, _, err := loadSkills([]string{first, last}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Description != "package copy" || !samePath(got[0].FilePath, filepath.Join(first, "SKILL.md")) {
		t.Fatalf("same-name skills were not Pi-compatible first-wins deduplicated: %+v", got)
	}
}
