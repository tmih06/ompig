package codingagent

import (
	"os"
	"strings"
)

// experimentalFeaturesEnabled reports whether experimental features are on.
// Upstream core/experimental.ts areExperimentalFeaturesEnabled checks
// process.env.PI_EXPERIMENTAL === "1". PI_EXPERIMENTAL mirrors Pi exactly (strict
// "1"); OMPIG_EXPERIMENTAL is Ompig's product-neutral alias, accepting the same
// truthy spellings as OMPIG_OFFLINE (see updateChecksOffline).
func experimentalFeaturesEnabled() bool {
	if os.Getenv("PI_EXPERIMENTAL") == "1" {
		return true
	}
	value := strings.ToLower(strings.TrimSpace(os.Getenv("OMPIG_EXPERIMENTAL")))
	return value == "1" || value == "true" || value == "yes"
}

// AreExperimentalFeaturesEnabled reports whether experimental features are on.
func AreExperimentalFeaturesEnabled() bool { return experimentalFeaturesEnabled() }
