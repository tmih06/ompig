//go:build !ompig_strip_export_html

package cli

import (
	"fmt"

	"github.com/tmih06/ompig/internal/codingagent"
	"github.com/tmih06/ompig/internal/pigstrip"
)

// runExportFlag is `ompig --export <session.jsonl> [output.html]`: it writes
// the session file as HTML and exits. Mirrors upstream main.ts:459-466.
//
// This file and its internal/codingagent/export import are the boundary a
// Ompiglet Binary compiles out with the ompig_strip_export_html tag.
func runExportFlag(sessionFile, outputPath string) {
	// ompig additive (D92): a Ompiglet that strips export-html fails as its Binary does.
	if pigstrip.Has(pigstrip.ListFeatures, pigstrip.ExportHTML) {
		exitProcess(reportStrippedFeature("HTML export", pigstrip.ExportHTML, ""))
	}
	result, err := codingagent.ExportFileToHTML(sessionFile, outputPath)
	if err != nil {
		printCLIError("%v", err)
		exitProcess(1)
	}
	fmt.Printf("Exported to: %s\n", result) // upstream: main.ts console.log(`Exported to: ${result}`)
	exitProcess(0)
}
