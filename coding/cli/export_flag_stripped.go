//go:build ompig_strip_export_html

package cli

import "github.com/tmih06/ompig/internal/pigstrip"

// ompig additive (D92): this Ompiglet Binary compiled out HTML session export.
func init() { pigstrip.Strip(pigstrip.ListFeatures, pigstrip.ExportHTML) }

// runExportFlag reports that `--export` is unavailable in this build and exits 1.
func runExportFlag(string, string) {
	exitProcess(reportStrippedFeature("HTML export", pigstrip.ExportHTML, ""))
}
