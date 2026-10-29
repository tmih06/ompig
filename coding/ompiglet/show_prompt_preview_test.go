package ompiglet

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// The system-prompt preview of `ompig ompiglet show` is cut at 77 UTF-16 units. A character outside the BMP at the cut leaves half
// a surrogate pair, which the preview writes as U+FFFD, as Node writes a lone surrogate, never as invalid UTF-8.
func TestShowSystemPromptPreviewIsValidUTF8WhenTheCutSplitsASurrogatePair(t *testing.T) {
	t.Setenv("OMPIG_HOME", t.TempDir())
	t.Chdir(t.TempDir())
	text := strings.Repeat("a", 76) + "😀" + strings.Repeat("b", 10)
	path := writePigletSource(t, t.TempDir(), "preview", "name: preview\nsystemPrompt:\n  text: \""+text+"\"\n")
	var out, errOut strings.Builder
	if code := cmdShow([]string{filepath.Clean(path)}, &out, &errOut); code != 0 {
		t.Fatalf("show code=%d err=%s", code, errOut.String())
	}
	want := "System prompt: " + strings.Repeat("a", 76) + "\uFFFD... ("
	if !utf8.ValidString(out.String()) || !strings.Contains(out.String(), want) {
		t.Fatalf("show output = %q, want valid UTF-8 containing %q", out.String(), want)
	}
}
