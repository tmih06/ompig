package cli

import (
	"testing"

	"github.com/tmih06/ompig/coding"
)

// main.ts:753 passes `extensionFlagValues: parsed.unknownFlags`, a Map in command-line order. Ompig's own flags (--ompiglet, --unsafe-host,
// --container-engine, --workspace) are not extension flags, and a repeated flag keeps its first position with its last value.
func TestExtensionFlagValuesKeepCommandLineOrderAndSkipPigFlags(t *testing.T) {
	flags := parseArgs([]string{"--zeta", "--ompiglet", "p", "--plan", "fast", "--unsafe-host", "--alpha=1", "--zeta=z"})
	want := []coding.ExtensionFlagValue{{Name: "zeta", Value: "z"}, {Name: "plan", Value: "fast"}, {Name: "alpha", Value: "1"}}
	got := flags.extensionFlagValues()
	if len(got) != len(want) {
		t.Fatalf("values = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("value %d = %v, want %v", i, got[i], want[i])
		}
	}
}
