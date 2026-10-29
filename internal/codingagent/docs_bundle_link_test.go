//go:build !ompig_strip_docs

package codingagent

// The ompig binary links internal/pigdocs, which registers this build's documentation bundle with configroot; these tests run with the same
// registration, as a binary that strips docs does not.
import _ "github.com/tmih06/ompig/internal/pigdocs"
