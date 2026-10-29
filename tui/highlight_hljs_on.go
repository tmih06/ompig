//go:build !ompig_strip_syntax_highlight

package tui

import (
	"sync"

	"github.com/tmih06/ompig/internal/pigstrip"
	"github.com/tmih06/ompig/tui/internal/hljs"
)

// ompig additive (D92): this file links highlight.js; a Ompiglet Binary built with ompig_strip_syntax_highlight uses highlight_hljs_off.go instead.

// highlightEngine is the highlight.js instance type.
type highlightEngine = *hljs.Registry

// highlightRegistry is the highlight.js instance with the languages syntax-highlight.ts registers when it loads.
var highlightRegistry = sync.OnceValue(hljs.NewRegistry)

// highlightStripped reports whether the active Ompiglet strips syntax-highlight. Stock Ompig never does.
func highlightStripped() bool {
	return pigstrip.Has(pigstrip.ListFeatures, pigstrip.SyntaxHighlight)
}
