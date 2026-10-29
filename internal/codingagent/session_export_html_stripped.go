//go:build ompig_strip_export_html

package codingagent

import (
	"github.com/tmih06/ompig/coding/extension"
	"github.com/tmih06/ompig/coding/extension/host/inproc"
	"github.com/tmih06/ompig/internal/pigstrip"
)

// ompig additive (D92): this Ompiglet Binary compiled out HTML session export (internal/codingagent/export and its template embeds).
func init() { pigstrip.Strip(pigstrip.ListFeatures, pigstrip.ExportHTML) }

// ExportToolRenderers draws no tool through renderers: this build has no HTML export.
func ExportToolRenderers(*inproc.Runner) func(name string) *extension.ToolRenderers { return nil }

// ExportSessionToHTML reports that HTML export is stripped. /export to a .jsonl path and /share do not use it.
func ExportSessionToHTML(string, string, func(name string) *extension.ToolRenderers, string, ShareState, string) (string, error) {
	return "", pigstrip.Error("HTML export", pigstrip.ListFeatures, pigstrip.ExportHTML)
}

// ExportFileToHTML reports that HTML export is stripped.
func ExportFileToHTML(string, string) (string, error) {
	return "", pigstrip.Error("HTML export", pigstrip.ListFeatures, pigstrip.ExportHTML)
}
