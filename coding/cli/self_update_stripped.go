//go:build ompig_strip_self_update

package cli

import (
	"github.com/tmih06/ompig/internal/codingagent"
	"github.com/tmih06/ompig/internal/pigstrip"
)

// ompig additive (D92): this Ompiglet Binary compiled out self-update; its Ompiglet distribution updates it.
func init() { pigstrip.Strip(pigstrip.ListFeatures, pigstrip.SelfUpdate) }

// runSelfUpdate refuses: this build cannot replace itself. `ompig update --all`
// still updates packages first and then exits 1 with this message.
func runSelfUpdate(bool) int {
	return reportStrippedFeature("self-update", pigstrip.SelfUpdate, selfUpdateStrippedHint)
}

// binaryUpdateChecker is nil: this build shows no new-version notice.
func binaryUpdateChecker() func() *codingagent.BinaryUpdate { return nil }
