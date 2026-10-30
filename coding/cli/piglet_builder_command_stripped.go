//go:build ompig_strip_ompiglet_builder

package cli

import "github.com/tmih06/ompig/internal/pigstrip"

// ompig additive (D92): this Ompiglet Binary compiled out the Ompiglet builder (coding/pigletbuild).
func init() { pigstrip.Strip(pigstrip.ListFeatures, pigstrip.PigletBuilder) }

// runPigletBuilderCommand reports that `ompig ompiglet build` and `ompig ompiglet publish` are unavailable in this build.
func runPigletBuilderCommand(args []string) int {
	if len(args) < 2 || args[0] != "ompiglet" || (args[1] != "build" && args[1] != "publish") {
		return -1
	}
	return reportStrippedFeature("ompig ompiglet "+args[1], pigstrip.PigletBuilder, "; use stock ompig")
}
