package cli

import (
	"fmt"
	"os"

	"github.com/tmih06/ompig/internal/pigstrip"
)

// selfUpdateStrippedHint follows the stripped self-update message: a Ompiglet
// Binary is updated by whatever distributes it.
const selfUpdateStrippedHint = "; update it through its Ompiglet distribution"

// reportStrippedFeature prints that the user requested a feature this Ompiglet
// strips, followed by hint, and returns the exit code 1.
// ompig additive (D92): one message for a stripped feature's CLI entry point, compiled out or stripped at runtime.
func reportStrippedFeature(what, id, hint string) int {
	fmt.Fprintf(os.Stderr, "ompig: %v%s\n", pigstrip.Error(what, pigstrip.ListFeatures, id), hint)
	return 1
}
