package codingagent

import "github.com/tmih06/ompig/coding/extension"

// rawMermaidMarkdown is the Mermaid transformer of a Ompiglet that strips mermaid: it returns the markdown unchanged, as the transformer does in mermaid mode "off".
//
// ompig additive (D92): shared by mermaid_transform.go (runtime strip) and mermaid_off.go (compiled out).
func rawMermaidMarkdown(markdown string, _ extension.MarkdownTransformContext) string {
	return markdown
}
