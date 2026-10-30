package testenv

import (
	"strings"
	"testing"
)

func TestShortTempBase(t *testing.T) {
	const localAppData = `C:\Users\runner\AppData\Local`
	longTemp := `D:\a\_temp\` + strings.Repeat("d", maxShortTempDirBytes)
	for _, test := range []struct {
		name, goos, temp, localAppData, want string
	}{
		{"unix uses /tmp whatever TMPDIR is", "darwin", "/var/folders/zz/" + strings.Repeat("d", 80), localAppData, "/tmp"},
		{"windows keeps a short temp directory", "windows", `C:\T`, localAppData, `C:\T`},
		{"windows keeps a temp directory that just fits", "windows", `D:\` + strings.Repeat("d", maxShortTempDirBytes-len(`D:\`)-len(`\pe`)-10), localAppData, `D:\` + strings.Repeat("d", maxShortTempDirBytes-len(`D:\`)-len(`\pe`)-10)},
		{"windows falls back when the temp directory is one byte too long", "windows", `D:\` + strings.Repeat("d", maxShortTempDirBytes-len(`D:\`)-len(`\pe`)-10+1), localAppData, localAppData + `\ompig\s`},
		{"windows falls back from a long temp directory", "windows", longTemp, localAppData, localAppData + `\ompig\s`},
		{"windows without local app data keeps the long temp directory", "windows", longTemp, "", longTemp},
	} {
		if got := shortTempBase(test.goos, test.temp, test.localAppData, "pe"); strings.ReplaceAll(got, "/", `\`) != strings.ReplaceAll(test.want, "/", `\`) {
			t.Errorf("%s: shortTempBase = %q, want %q", test.name, got, test.want)
		}
	}
}

// The Windows CI fallback base leaves less room than the longest prefixes the
// extension-host tests use, and every such test needs a directory whose socket
// path still fits: the prefix is trimmed to the budget rather than the test
// failing, and the trimmed name still starts with the caller's prefix.
func TestShortTempPrefixFitsTheWindowsBudget(t *testing.T) {
	const base = `C:\Users\runneradmin\AppData\Local\ompig\s`
	for _, prefix := range []string{"ompig", "ompig-rust-packed-", "ompig-python-packed-"} {
		got := fitShortTempPrefix("windows", base, prefix)
		if !strings.HasPrefix(prefix, got) {
			t.Errorf("prefix %q became %q, which is not a prefix of it", prefix, got)
		}
		if total := len(base) + 1 + len(got) + uint32Digits; total > maxShortTempDirBytes {
			t.Errorf("prefix %q became %q; %d bytes with the base and suffix, want at most %d", prefix, got, total, maxShortTempDirBytes)
		}
	}
	if got := fitShortTempPrefix("linux", base, "ompig-python-packed-"); got != "ompig-python-packed-" {
		t.Errorf("linux prefix = %q, want it untouched", got)
	}
}
