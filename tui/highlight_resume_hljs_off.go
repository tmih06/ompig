//go:build ompig_strip_syntax_highlight

package tui

// ompig additive (D92): a Ompiglet Binary built with ompig_strip_syntax_highlight links no highlight.js, so no code resumes.

func resetHighlightResumesLocked() {}

func highlightResumable(highlightEngine, string, string, bool, *Theme) ([]string, bool) {
	return nil, false
}
