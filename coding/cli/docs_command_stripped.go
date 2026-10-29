//go:build ompig_strip_docs

package cli

import (
	"context"
	"encoding/json"
	"io"

	"github.com/tmih06/ompig/agent"
	"github.com/tmih06/ompig/internal/pigstrip"
)

// ompig additive (D92): this Ompiglet Binary compiled out the Ompig documentation bundle.
func init() { pigstrip.Strip(pigstrip.ListFeatures, pigstrip.Docs) }

// runDocsCommand reports that `ompig docs` is unavailable in this build.
func runDocsCommand(args []string, _, _ io.Writer) int {
	if len(args) == 0 || args[0] != "docs" {
		return -1
	}
	return reportStrippedFeature("ompig docs", pigstrip.Docs, "")
}

// prepareDocsBundle does nothing: this build has no documentation bundle.
func prepareDocsBundle() {}

// writeDocsBundle does nothing: this build has no documentation bundle.
func writeDocsBundle() {}

// writeDocsBundleBeforeToolCall lets every tool call proceed: this build has no documentation bundle.
func writeDocsBundleBeforeToolCall(context.Context, string, string, json.RawMessage) agent.ToolCallHookResult {
	return agent.ToolCallHookResult{}
}

// releaseDocsLeases does nothing: this build has no documentation bundle to lease.
func releaseDocsLeases() {}
