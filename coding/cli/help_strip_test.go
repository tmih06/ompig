package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tmih06/ompig/internal/pigstrip"
)

// ompig additive (D92): a Ompiglet that strips self-update lists `ompig update [source]` in --help, without the self targets
// that Pi 1.1.0 args.ts:293 lists as `update [source|self|pi]` and that Stock Ompig prints as `[source|self|ompig]`.
func TestHelpWithoutSelfUpdateListsOnlyTheSourceTarget(t *testing.T) {
	var stock bytes.Buffer
	printHelp(&stock, false)
	if !strings.Contains(stock.String(), "  ompig update [source|self|ompig]   Update ompig, extensions, or model catalogs\n") {
		t.Fatalf("stock help lacks Pi's update line:\n%s", stock.String())
	}
	t.Cleanup(pigstrip.Strip(pigstrip.ListFeatures, pigstrip.SelfUpdate))
	var stripped bytes.Buffer
	printHelp(&stripped, false)
	if !strings.Contains(stripped.String(), "  ompig update [source]           Update extensions or model catalogs\n") || strings.Contains(stripped.String(), "source|self") {
		t.Fatalf("help without self-update:\n%s", stripped.String())
	}
}
