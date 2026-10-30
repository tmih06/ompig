package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/tmih06/ompig/internal/pigstrip"
)

// assertStripBuildOmits checks that stock cmd/ompig links pkg and a build with the feature's tag does not.
func assertStripBuildOmits(t *testing.T, feature, pkg string) {
	t.Helper()
	tag := pigstrip.Tag(feature)
	if !slices.Contains(goListCmdPig(t, "", "-deps"), pkg) {
		t.Errorf("the stock cmd/ompig does not link %s", pkg)
	}
	if slices.Contains(goListCmdPig(t, tag, "-deps"), pkg) {
		t.Errorf("a %s build of cmd/ompig links %s", tag, pkg)
	}
}

// A ompig_strip_export_html build compiles internal/codingagent's session_export_html.go and coding/cli's export_flag.go out,
// so it does not link internal/codingagent/export with its HTML template, CSS, JS and vendored marked/highlight.js embeds.
func TestStripExportHTMLBuildOmitsExportPackage(t *testing.T) {
	assertStripBuildOmits(t, pigstrip.ExportHTML, "github.com/tmih06/ompig/internal/codingagent/export")
}

// A ompig_strip_changelog build compiles internal/codingagent's changelog_bundle.go out, so it does not link the root
// package and its CHANGELOG.md embed.
func TestStripChangelogBuildOmitsRootPackage(t *testing.T) {
	assertStripBuildOmits(t, pigstrip.Changelog, "github.com/tmih06/ompig")
}

// A ompig_strip_docs build compiles coding/cli's docs_command.go out, so it does not link internal/pigdocs and its
// content/*.md bundle.
func TestStripDocsBuildOmitsPigdocs(t *testing.T) {
	assertStripBuildOmits(t, pigstrip.Docs, "github.com/tmih06/ompig/internal/pigdocs")
}

// A ompig_strip_ompiglet_builder build compiles coding/cli's piglet_builder_command.go out, so it does not link
// coding/pigletbuild.
func TestStripPigletBuilderBuildOmitsPigletbuild(t *testing.T) {
	assertStripBuildOmits(t, pigstrip.PigletBuilder, "github.com/tmih06/ompig/coding/pigletbuild")
}

// Self-update lives in coding/cli's self_update.go on internal/codingagent's update support, which other commands share, so
// a ompig_strip_self_update build drops no package: it compiles self_update.go out for self_update_stripped.go.
func TestStripSelfUpdateBuildOmitsSelfUpdateFile(t *testing.T) {
	format := []string{"-f", "{{.GoFiles}}"}
	const cli = "github.com/tmih06/ompig/coding/cli"
	stock := goListPackage(t, cli, "", format...)
	if !slices.Contains(stock, "self_update.go") || slices.Contains(stock, "self_update_stripped.go") {
		t.Errorf("stock coding/cli files: %v", stock)
	}
	tagged := goListPackage(t, cli, pigstrip.Tag(pigstrip.SelfUpdate), format...)
	if slices.Contains(tagged, "self_update.go") || !slices.Contains(tagged, "self_update_stripped.go") {
		t.Errorf("ompig_strip_self_update coding/cli files: %v", tagged)
	}
}

// A Ompiglet that strips docs, at runtime or by compiling them out of its Binary, answers `ompig docs` with the stripped
// message and exit 1 and writes no bundle. Stock prints the docs directory.
func TestStripDocsCommandReportsStripped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("OMPIG_HOME", home)
	if !pigstrip.Has(pigstrip.ListFeatures, pigstrip.Docs) {
		stdout, _, code := captureStdoutStderr(t, func() int { return runPigPreSessionCommand([]string{"docs", "path"}) })
		if code != 0 || !strings.HasPrefix(stdout, home) {
			t.Fatalf("stock `ompig docs path`: exit %d, stdout %q", code, stdout)
		}
	}
	// Stock `ompig docs path` wrote its bundle; the stripped startup must leave home as it is.
	before := fileTreeOf(t, home)
	t.Cleanup(pigstrip.Strip(pigstrip.ListFeatures, pigstrip.Docs))
	stdout, stderr, code := captureStdoutStderr(t, func() int { return runPigPreSessionCommand([]string{"docs", "path"}) })
	if want := "ompig: ompig docs is stripped from this Ompiglet (strip.features: docs)\n"; code != 1 || stdout != "" || stderr != want {
		t.Fatalf("stripped `ompig docs path`: exit %d, stdout %q, stderr %q, want %q", code, stdout, stderr, want)
	}
	prepareDocsBundle()
	writeDocsBundle()
	if after := fileTreeOf(t, home); !slices.Equal(after, before) {
		t.Fatalf("stripped startup wrote under OMPIG_HOME: before %v, after %v", before, after)
	}
}

// A Ompiglet that strips the Ompiglet builder, at runtime or by compiling it out of its Binary, answers `ompig ompiglet build`
// and `ompig ompiglet publish` with the stripped message and exit 1. Stock runs the builder.
func TestStripPigletBuilderCommandReportsStripped(t *testing.T) {
	if !pigstrip.Has(pigstrip.ListFeatures, pigstrip.PigletBuilder) {
		stdout, _, code := captureStdoutStderr(t, func() int { return runPigPreSessionCommand([]string{"ompiglet", "build", "--help"}) })
		if code != 0 || !strings.Contains(stdout, "ompig ompiglet build <name>") {
			t.Fatalf("stock `ompig ompiglet build --help`: exit %d, stdout %q", code, stdout)
		}
	}
	t.Cleanup(pigstrip.Strip(pigstrip.ListFeatures, pigstrip.PigletBuilder))
	for _, sub := range []string{"build", "publish"} {
		stdout, stderr, code := captureStdoutStderr(t, func() int { return runPigPreSessionCommand([]string{"ompiglet", sub, "--help"}) })
		want := "ompig: ompig ompiglet " + sub + " is stripped from this Ompiglet (strip.features: ompiglet-builder); use stock ompig\n"
		if code != 1 || stdout != "" || stderr != want {
			t.Fatalf("stripped `ompig ompiglet %s`: exit %d, stdout %q, stderr %q, want %q", sub, code, stdout, stderr, want)
		}
	}
}

// A Ompiglet that strips self-update, at runtime or by compiling it out of its Binary, refuses `ompig update` with the
// stripped message and exit 1 and shows no startup new-version notice. Stock resolves an installation tier.
func TestStripSelfUpdateRefusesUpdate(t *testing.T) {
	t.Setenv("OMPIG_HOME", t.TempDir())
	t.Setenv("OMPIG_UPDATE_URL", "")
	if !pigstrip.Has(pigstrip.ListFeatures, pigstrip.SelfUpdate) {
		_, stderr, code := captureStdoutStderr(t, func() int { return runSelfUpdate(false) })
		if code != 1 || !strings.Contains(stderr, "cannot self-update this installation") {
			t.Fatalf("stock unreceipted update: exit %d, stderr %q", code, stderr)
		}
		if binaryUpdateChecker() == nil {
			t.Fatal("stock interactive mode has no binary update check")
		}
	}
	t.Cleanup(pigstrip.Strip(pigstrip.ListFeatures, pigstrip.SelfUpdate))
	stdout, stderr, code := captureStdoutStderr(t, func() int { return runSelfUpdate(false) })
	want := "ompig: self-update is stripped from this Ompiglet (strip.features: self-update); update it through its Ompiglet distribution\n"
	if code != 1 || stdout != "" || stderr != want {
		t.Fatalf("stripped update: exit %d, stdout %q, stderr %q, want %q", code, stdout, stderr, want)
	}
	if binaryUpdateChecker() != nil {
		t.Fatal("stripped self-update still checks for a new version at startup")
	}
}
