//go:build ompig_strip_extension_sdk_go

package pigsdk

import "github.com/tmih06/ompig/internal/pigstrip"

// ompig additive (D92): this Ompiglet Binary compiled out the embedded Go SDK, so it stages none and a Go build that needs it reports the strip.
func init() { pigstrip.Strip(pigstrip.ListFeatures, pigstrip.ExtensionSDKGo) }

func goBundle() (sdkBundle, bool) { return sdkBundle{}, false }
