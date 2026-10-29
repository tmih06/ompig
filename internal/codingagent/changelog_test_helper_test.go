package codingagent

import ompig "github.com/tmih06/ompig"

// getBundledChangelogForTest exists so changelog_test.go can read the
// embedded CHANGELOG.md without importing the root ompig package
// directly. Test-only indirection.
func getBundledChangelogForTest() string { return ompig.Changelog() }
