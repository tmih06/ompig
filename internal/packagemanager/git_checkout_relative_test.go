package packagemanager

import (
	"path/filepath"
	"strings"
	"testing"

	sourceref "github.com/tmih06/ompig/coding/source"
)

func checkoutRelative(t *testing.T, goos, source string) string {
	t.Helper()
	ref, err := sourceref.Parse(source, sourceref.Options{Bare: sourceref.BareReject})
	if err != nil {
		t.Fatalf("parse %s: %v", source, err)
	}
	got, err := GitCheckoutRelative(goos, "root", ref)
	if err != nil {
		t.Fatalf("%s %s: %v", goos, source, err)
	}
	return got
}

func TestGitCheckoutRelativeFileSourceIsShort(t *testing.T) {
	for _, tc := range []struct{ goos, source, want string }{
		{"linux", "git:file://localhost/tmp/x/ompiglet", filepath.Join("localhost", "72460493469a", "ompiglet")},
		{"windows", "git:file://localhost/C:/Users/me/owner/ompiglet.git", filepath.Join("localhost", "a7ceb009cf5b", "ompiglet")},
		{"windows", "git:file://localhost/c:/Users/me/owner/ompiglet.git", filepath.Join("localhost", "a7ceb009cf5b", "ompiglet")},
	} {
		if got := checkoutRelative(t, tc.goos, tc.source); got != tc.want {
			t.Errorf("%s %s = %q; want %q", tc.goos, tc.source, got, tc.want)
		}
	}
}

func TestGitCheckoutRelativeSameBasenameDiffers(t *testing.T) {
	a := checkoutRelative(t, "linux", "git:file://localhost/tmp/a/ompiglet")
	b := checkoutRelative(t, "linux", "git:file://localhost/tmp/b/ompiglet")
	if a == b {
		t.Fatalf("different repositories share checkout %q", a)
	}
}

func TestGitCheckoutRelativeNonFileKeepsPiLayout(t *testing.T) {
	for source, want := range map[string]string{
		"git:https://example.com/owner/repo":     filepath.Join("example.com", "owner", "repo"),
		"git:https://example.com/owner/repo.git": filepath.Join("example.com", "owner", "repo"),
		"ssh://git@example.com/owner/sub/repo":   filepath.Join("example.com", "owner", "sub", "repo"),
		"git://example.com/owner/repo":           filepath.Join("example.com", "owner", "repo"),
		"git:git@example.com:owner/repo.git":     filepath.Join("example.com", "owner", "repo"),
	} {
		for _, goos := range []string{"linux", "windows"} {
			if got := checkoutRelative(t, goos, source); got != want {
				t.Errorf("%s %s = %q; want %q", goos, source, got, want)
			}
		}
	}
}

func TestGitCheckoutRelativeFileSourceLengthBound(t *testing.T) {
	long := strings.Repeat("d", 40) + "/" + strings.Repeat("e", 40) + "/" + strings.Repeat("f", 40) + "/" + strings.Repeat("g", 40) + "/repo"
	got := checkoutRelative(t, "linux", "git:file://localhost/"+long)
	if len(got) > len("localhost/")+12+len("/repo") {
		t.Fatalf("checkout %q is %d bytes; want a short fixed-size path", got, len(got))
	}
}

// A validate-only scratch root (<temp>\ompig-validate-*\git) and the install roots
// all map a source through GitCheckoutRelative, so a long Windows file:// source
// adds only a short fixed-size segment to the root.
func TestGitCheckoutRelativeWindowsFileSourceUnderScratchRootIsShort(t *testing.T) {
	long := "C:/Users/RUNNER~1/AppData/Local/Temp/" + strings.Repeat("TestPigletValidateInstallsGitPackagesWithoutRunningScripts/", 3) + "010"
	ref, err := sourceref.Parse("git:file://localhost/"+long, sourceref.Options{Bare: sourceref.BareReject})
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join("scratch", "ompig-validate-123", "git")
	got, err := GitCheckoutRelative("windows", root, ref)
	if err != nil {
		t.Fatal(err)
	}
	if want := len("localhost/") + 12 + len("/010"); len(got) != want {
		t.Fatalf("checkout %q is %d bytes; want %d", got, len(got), want)
	}
}
