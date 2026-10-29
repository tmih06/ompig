//go:build !ompig_strip_extension_sdk_rust

package pigsdk

import (
	"slices"
	"sync"

	"github.com/tmih06/ompig/internal/pigstrip"

	rssdk "github.com/tmih06/ompig/extensions/sdk-rs"
)

// ompig additive (D92): only a build that embeds the Rust SDK registers its staged directory.
func init() { registerBundle("rust", rustBundle) }

// rustBundle is the embedded Rust SDK, absent when the active Ompiglet strips it.
// ompig additive (D92): a Ompiglet Binary built with ompig_strip_extension_sdk_rust links sdk_rust_off.go instead and embeds no Rust SDK.
func rustBundle() (sdkBundle, bool) {
	if pigstrip.Has(pigstrip.ListFeatures, pigstrip.ExtensionSDKRust) {
		return sdkBundle{}, false
	}
	return sdkBundle{lang: "rust", files: rustBundleFiles(), fsys: rssdk.Source}, true
}

// rustBundleFiles lists the embedded Rust SDK once per process: the embedded files never change, and staging and the staged directory's hash both read the list, which they never write; its capacity is clipped, so an append copies it.
var rustBundleFiles = sync.OnceValue(func() []string { return slices.Clip(rssdk.BundledFiles()) })
