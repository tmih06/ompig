package typebox

import (
	"testing"

	"github.com/tmih06/ompig/internal/lazyregexp"
)

// The format expressions are created on first use, so coding/cli's TestLazyRegexpPatternsAllCompile, which compiles the patterns created
// at init, does not see them. Every one of them must compile.
func TestFormatPatternsAllCompile(t *testing.T) {
	if len(Formats()) == 0 {
		t.Fatal("no formats")
	}
	if invalid := lazyregexp.CompileAll(); invalid != "" {
		t.Fatal(invalid)
	}
}
