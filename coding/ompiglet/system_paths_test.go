package ompiglet

import (
	"strings"
	"testing"
)

// On Windows a rooted path without a volume ("/etc/ompig/ompiglets") names a directory on the current drive, which any local user can create. The
// machine-wide Ompiglet directory and the image-baked Ompiglet file are therefore not searched there, so one user cannot plant a Ompiglet that every
// other user's ompig loads without a flag.
func TestSystemPigletLocationsAreUnixOnly(t *testing.T) {
	if got := systemPigletsDirFor("windows"); got != "" {
		t.Fatalf("Windows system Ompiglet dir = %q, want none", got)
	}
	if got := ambientPigletPathFor("windows"); got != "" {
		t.Fatalf("Windows ambient Ompiglet path = %q, want none", got)
	}
	for _, goos := range []string{"linux", "darwin"} {
		if got := systemPigletsDirFor(goos); got != "/etc/ompig/ompiglets" {
			t.Fatalf("%s system Ompiglet dir = %q", goos, got)
		}
	}
}

func TestClassifyLocationNamesTheSystemPigletDirectory(t *testing.T) {
	if dir := SystemPigletsDir(); dir != "" {
		if got := classifyLocation(dir); got != "system" {
			t.Fatalf("classifyLocation(%q) = %q, want system", dir, got)
		}
	}
	if got := classifyLocation("/etc/elsewhere"); got != "unknown" {
		t.Fatalf("classifyLocation(/etc/elsewhere) = %q, want unknown", got)
	}
}

// The usage lists the system Ompiglet directory only where the platform searches one.
func TestPigletHelpListsTheSystemDirectoryOnlyWhereItIsSearched(t *testing.T) {
	const line = "/etc/ompig/ompiglets/     system"
	if got := pigletUsageFor("/etc/ompig/ompiglets"); !strings.Contains(got, line) {
		t.Fatalf("usage omits the system directory:\n%s", got)
	}
	if got := pigletUsageFor(""); strings.Contains(got, line) || !strings.Contains(got, "~/.ompig/ompiglets/       user") {
		t.Fatalf("usage without a system directory:\n%s", got)
	}
}
