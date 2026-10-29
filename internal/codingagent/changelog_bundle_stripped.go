//go:build ompig_strip_changelog

package codingagent

import "github.com/tmih06/ompig/internal/pigstrip"

// ompig additive (D92): this Ompiglet Binary compiled out the bundled changelog (the root package's CHANGELOG.md embed).
func init() { pigstrip.Strip(pigstrip.ListFeatures, pigstrip.Changelog) }

// bundledChangelog is empty: this build has no changelog, so the startup What's New never shows.
func bundledChangelog() string { return "" }
