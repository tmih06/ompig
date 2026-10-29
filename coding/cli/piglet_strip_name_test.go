package cli

import (
	"strings"
	"testing"

	ompiglet "github.com/tmih06/ompig/coding/ompiglet"
	"github.com/tmih06/ompig/internal/pigstrip"
)

// D92 (opt-in strip): the active Ompiglet's strip list also names the Ompiglet, so a request for a stripped built-in says which Ompiglet strips it and how to keep it.
func TestApplyPigletStripNamesThePigletInStripErrors(t *testing.T) {
	p := &ompiglet.Ompiglet{Name: "a2a", Strip: &ompiglet.StripSpec{Features: []string{pigstrip.NodeExtensions}}}
	undo := applyPigletStrip(p, &Args{})
	got := pigstrip.Error("The TypeScript and JavaScript extension runtime", pigstrip.ListFeatures, pigstrip.NodeExtensions).Error()
	for _, want := range []string{`the Ompiglet "a2a"`, "strip.features: node-extensions", "remove node-extensions from strip.features in the Ompiglet manifest and rebuild"} {
		if !strings.Contains(got, want) {
			t.Errorf("error %q lacks %q", got, want)
		}
	}
	undo()
	if got := pigstrip.Error("x", pigstrip.ListFeatures, pigstrip.NodeExtensions).Error(); strings.Contains(got, "a2a") {
		t.Errorf("after undo the error still names the Ompiglet: %q", got)
	}
}
