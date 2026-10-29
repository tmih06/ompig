//go:build ompig_strip_extension_sdk_go

package cli

import (
	"context"
	"errors"

	"github.com/tmih06/ompig/coding/extension/host/subprocess"
	"github.com/tmih06/ompig/extensions/sdk/upgrade"
)

// ompig additive (D92): a Ompiglet Binary that compiled out the embedded Go SDK cannot type check a Go extension against it, and linking the type checker (go/types, go/importer) only for this analysis would keep about 1 MiB of code in a Binary that stripped the Go SDK.
func (u *extensionUpgrader) analyzeExtension(context.Context, subprocess.ExtConfig) (*upgrade.Result, error) {
	return nil, errors.New("this Ompiglet Binary compiled out the Go extension SDK, so it cannot analyze a Go extension for an upgrade")
}

func (u *extensionUpgrader) checkRewrite(context.Context, subprocess.ExtConfig, *upgrade.Result) (rewriteCheck, error) {
	return rewriteCheck{}, errors.New("this Ompiglet Binary compiled out the Go extension SDK, so it cannot build a Go extension for an upgrade")
}
