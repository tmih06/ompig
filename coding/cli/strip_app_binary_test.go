package cli

import (
	"strings"
	"testing"

	"github.com/tmih06/ompig/internal/pigstrip"
)

// The fully stripped Binary answers each compiled-out app feature's CLI entry point with the strip message and exit 1.
func TestStrippedBinaryAppFeaturesReportStripped(t *testing.T) {
	for _, tc := range []struct {
		args []string
		what string
		id   string
		hint string
	}{
		{[]string{"--export", "session.jsonl"}, "HTML export", pigstrip.ExportHTML, ""},
		{[]string{"update", "self"}, "self-update", pigstrip.SelfUpdate, "; update it through its Ompiglet distribution"},
		{[]string{"docs"}, "ompig docs", pigstrip.Docs, ""},
		{[]string{"docs", "show", "README"}, "ompig docs", pigstrip.Docs, ""},
		{[]string{"ompiglet", "build", "demo", "--format", "script", "--out", "-"}, "ompig ompiglet build", pigstrip.PigletBuilder, "; use stock ompig"},
		{[]string{"ompiglet", "publish", "demo"}, "ompig ompiglet publish", pigstrip.PigletBuilder, "; use stock ompig"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			stdout, stderr, code := runStrippedPig(t, "", tc.args...)
			want := "ompig: " + pigstrip.Error(tc.what, pigstrip.ListFeatures, tc.id).Error() + tc.hint + "\n"
			if code != 1 || stdout != "" || stderr != want {
				t.Fatalf("exit %d\nstdout: %q\nstderr: %q\nwant stderr %q", code, stdout, stderr, want)
			}
		})
	}
}

// `ompig update --all` in the stripped Binary still updates packages, then reports self-update stripped and exits 1.
func TestStrippedBinaryUpdateAllUpdatesPackagesThenReportsSelfUpdate(t *testing.T) {
	stdout, stderr, code := runStrippedPig(t, "", "update", "--all")
	want := "ompig: " + pigstrip.Error("self-update", pigstrip.ListFeatures, pigstrip.SelfUpdate).Error() + "; update it through its Ompiglet distribution\n"
	if code != 1 || !strings.Contains(stdout, "Updated packages\n") || stderr != want {
		t.Fatalf("exit %d\nstdout: %q\nstderr: %q\nwant stderr %q", code, stdout, stderr, want)
	}
}
