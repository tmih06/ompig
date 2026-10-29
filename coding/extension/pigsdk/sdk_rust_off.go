//go:build ompig_strip_extension_sdk_rust

package pigsdk

import "github.com/tmih06/ompig/internal/pigstrip"

// ompig additive (D92): this Ompiglet Binary compiled out the embedded Rust SDK, so it stages none and a Rust build that needs it reports the strip.
func init() { pigstrip.Strip(pigstrip.ListFeatures, pigstrip.ExtensionSDKRust) }

func rustBundle() (sdkBundle, bool) { return sdkBundle{}, false }
