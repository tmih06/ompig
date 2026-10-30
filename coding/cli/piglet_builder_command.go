//go:build !ompig_strip_ompiglet_builder

package cli

import (
	"os"

	"github.com/tmih06/ompig/coding/pigletbuild"
	"github.com/tmih06/ompig/internal/pigstrip"
)

// This file and its coding/pigletbuild import are the boundary a Ompiglet
// Binary compiles out with the ompig_strip_ompiglet_builder tag.

// runPigletBuilderCommand runs `ompig ompiglet build` and `ompig ompiglet publish`
// and returns the exit code, or -1 when args target another command.
func runPigletBuilderCommand(args []string) int {
	if len(args) < 2 || args[0] != "ompiglet" || (args[1] != "build" && args[1] != "publish") {
		return -1
	}
	// ompig additive (D92): a Ompiglet that strips the builder reports it absent, as its Binary does.
	if pigstrip.Has(pigstrip.ListFeatures, pigstrip.PigletBuilder) {
		return reportStrippedFeature("ompig ompiglet "+args[1], pigstrip.PigletBuilder, "; use stock ompig")
	}
	if args[1] == "build" {
		return pigletbuild.RunPigletBuildCommand(args[2:], os.Stdout, os.Stderr)
	}
	return pigletbuild.RunPigletPublishCommand(args[2:], os.Stdout, os.Stderr)
}
