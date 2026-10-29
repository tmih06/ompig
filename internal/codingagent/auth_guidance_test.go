package codingagent

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmih06/ompig/internal/codingagent/prompts"
	"github.com/tmih06/ompig/internal/configroot"
	"github.com/tmih06/ompig/internal/pigdocs"
)

// Login help names the providers and models pages of this build's materialized bundle (auth-guidance.ts:9-10 names Pi's package docs).
func TestProviderLoginHelpPrefersMaterializedPigDocs(t *testing.T) {
	ompigHome := t.TempDir()
	t.Setenv("OMPIG_HOME", ompigHome)
	if err := pigdocs.EnsureSynced(ompigHome); err != nil {
		t.Fatal(err)
	}

	msg := ProviderLoginHelp()
	wantProviders := filepath.Join(pigdocs.DocsDir(ompigHome), "providers.md")
	wantModels := filepath.Join(pigdocs.DocsDir(ompigHome), "models.md")
	if !strings.Contains(msg, wantProviders) || !strings.Contains(msg, wantModels) {
		t.Fatalf("ProviderLoginHelp = %q, want materialized ompig docs %q and %q", msg, wantProviders, wantModels)
	}
	if strings.Contains(msg, ".upstream/current") {
		t.Fatalf("ProviderLoginHelp should not prefer source docs when ompig docs exist: %q", msg)
	}
}

// When startup could not write the documentation bundle to a config root that held none of this build (D22), no text names it: login help keeps
// its /login sentence and names no page, OmpigDocsFile reports the page absent, and the system prompt has no docs section.
func TestNoTextNamesADocsBundleStartupCouldNotWrite(t *testing.T) {
	ompigHome := t.TempDir()
	t.Setenv("OMPIG_HOME", ompigHome)
	configroot.SetDocsPresent(false)
	t.Cleanup(func() { configroot.SetDocsPresent(true) })

	if path, ok := OmpigDocsFile("providers.md"); ok {
		t.Fatalf("OmpigDocsFile(providers.md) = %q, true; want false while the bundle is missing", path)
	}
	help := ProviderLoginHelp()
	if !strings.Contains(help, "/login") || strings.Contains(help, filepath.Join(ompigHome, "docs")) {
		t.Fatalf("ProviderLoginHelp = %q, want the /login sentence and no docs page", help)
	}
	if prompt := prompts.BuildDefaultPrompt(prompts.Options{Cwd: "/w"}); strings.Contains(prompt, "Ompig documentation") || strings.Contains(prompt, filepath.Join(ompigHome, "docs")) {
		t.Fatalf("the system prompt names the missing bundle:\n%s", prompt)
	}
}
