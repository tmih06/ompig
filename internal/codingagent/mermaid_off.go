//go:build ompig_strip_mermaid

package codingagent

import (
	"github.com/tmih06/ompig/coding/extension"
	"github.com/tmih06/ompig/internal/pigstrip"
	"github.com/tmih06/ompig/tui"
)

// ompig additive (D92): a Ompiglet Binary built with ompig_strip_mermaid does not link internal/mermaid; mermaid code blocks stay raw, as in mermaid mode "off".

func init() { pigstrip.Strip(pigstrip.ListFeatures, pigstrip.Mermaid) }

func createMermaidMarkdownTransformer(func() string, *tui.Theme) extension.MarkdownTransformer {
	return rawMermaidMarkdown
}
