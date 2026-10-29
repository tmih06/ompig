// Package pigstrip is the one strip state of a Ompig process: the built-ins this
// process runs without. A Ompiglet Binary's OFF registration shims record the
// ones it compiled out from init, and the startup code records every entry of
// the active Ompiglet's strip list (cmd/ompig recordPigletStrip), so a compiled
// out built-in and a runtime stripped one are the same record. Every
// registry that offers a built-in asks Has where the built-in is registered,
// so no surface derived from that registry can show it, and a request for a
// stripped built-in reports Error.
//
// ompig additive (D92): Stock Ompig compiles every shim in and runs without a
// Ompiglet strip list, so Has is always false there.
package pigstrip

import (
	"fmt"
	"slices"
	"sync"
)

// Strip lists, named as in a Ompiglet's strip key, in manifest order.
const (
	ListTools      = "tools"
	ListCommands   = "commands"
	ListExtensions = "extensions"
	ListAPIs       = "apis"
	ListFeatures   = "features"
)

// Lists returns the five strip lists in manifest order.
func Lists() []string {
	return []string{ListTools, ListCommands, ListExtensions, ListAPIs, ListFeatures}
}

// Known returns the stable IDs list may name, sorted, from the generated ID
// table (ids_generated.go). A command ID keeps its leading slash.
func Known(list string) []string {
	return slices.Clone(knownIDs[list])
}

// Feature IDs. The runtime-only features reuse a Pi switch where the feature
// loads; the others have registration shims a Ompiglet Binary compiles out with
// the build tag ompig_strip_<id> (every character other than a letter or digit
// becomes an underscore).
const (
	// Themes disables custom theme discovery and --theme paths (Pi --no-themes).
	Themes = "themes"
	// Skills disables skill discovery and --skill paths (Pi --no-skills).
	Skills = "skills"
	// PromptTemplates disables prompt template discovery and --prompt-template
	// paths (Pi --no-prompt-templates).
	PromptTemplates = "prompt-templates"
	// NodeExtensions is the embedded Node runtime that runs TypeScript and
	// JavaScript extensions.
	NodeExtensions = "node-extensions"
	// ExtensionSDKGo, ExtensionSDKRust and ExtensionSDKPython are the
	// embedded SDK sources Ompig stages to build Go and Rust source extensions
	// and to run Python extensions.
	ExtensionSDKGo     = "extension-sdk-go"
	ExtensionSDKRust   = "extension-sdk-rust"
	ExtensionSDKPython = "extension-sdk-python"
	// SyntaxHighlight is highlight.js grammar highlighting in code blocks and
	// tool output.
	SyntaxHighlight = "syntax-highlight"
	// WordDictionaries are the dictionaries that find word boundaries in CJK,
	// Thai, Lao, Khmer and Burmese text for word navigation.
	WordDictionaries = "word-dictionaries"
	// ExportHTML is the HTML session export (/export to .html, --export,
	// RPC export_html).
	ExportHTML = "export-html"
	// SelfUpdate is `ompig update` of the ompig binary and the startup new
	// version notice.
	SelfUpdate = "self-update"
	// Changelog is the bundled changelog behind /changelog and the startup
	// What's New.
	Changelog = "changelog"
	// Docs is the bundled Ompig documentation (`ompig docs`) and the system
	// prompt section that points the model at it.
	Docs = "docs"
	// Mermaid draws mermaid code blocks as diagrams.
	Mermaid = "mermaid"
	// PigletBuilder is `ompig ompiglet build` and `ompig ompiglet publish`.
	PigletBuilder = "ompiglet-builder"
)

// Features lists every feature ID a Ompiglet can strip: the runtime-only
// features, then the ones a Ompiglet Binary compiles out.
func Features() []string {
	return append([]string{Themes, Skills, PromptTemplates}, BinaryFeatures()...)
}

// BinaryFeatures lists the feature IDs a Ompiglet Binary can compile out with
// a ompig_strip_<id> tag.
func BinaryFeatures() []string {
	return []string{
		NodeExtensions, ExtensionSDKGo, ExtensionSDKRust, ExtensionSDKPython,
		SyntaxHighlight, WordDictionaries, ExportHTML, SelfUpdate, Changelog,
		Docs, Mermaid, PigletBuilder,
	}
}

// Tag is the Go build tag that compiles id out: ompig_strip_ followed by id
// with every character other than an ASCII letter or digit replaced by an
// underscore (llama.cpp becomes ompig_strip_llama_cpp).
func Tag(id string) string {
	b := []byte("ompig_strip_" + id)
	for i := len("ompig_strip_"); i < len(b); i++ {
		if c := b[i]; (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			b[i] = '_'
		}
	}
	return string(b)
}

type entry struct{ list, id string }

var (
	mu       sync.RWMutex
	stripped []entry
	ompiglet string
)

// SetPiglet names the Ompiglet whose strip list this process runs, so a request
// for a stripped built-in can say which Ompiglet strips it and how to keep it.
// The returned function restores the previous name.
func SetPiglet(name string) (undo func()) {
	mu.Lock()
	defer mu.Unlock()
	previous := ompiglet
	ompiglet = name
	return func() {
		mu.Lock()
		defer mu.Unlock()
		ompiglet = previous
	}
}

// Strip records that list/id is absent from this process. A registration
// shim compiled with ompig_strip_<id> calls it from init; the startup code
// calls it for each entry of the active Ompiglet's strip list. The returned
// function undoes the call.
func Strip(list, id string) (undo func()) {
	mu.Lock()
	defer mu.Unlock()
	e := entry{list, id}
	if slices.Contains(stripped, e) {
		return func() {}
	}
	stripped = append(stripped, e)
	return func() {
		mu.Lock()
		defer mu.Unlock()
		if i := slices.Index(stripped, e); i >= 0 {
			stripped = slices.Delete(stripped, i, i+1)
		}
	}
}

// Has reports whether list/id is stripped from this process. A command is
// named with its leading slash, as in the strip list.
func Has(list, id string) bool {
	mu.RLock()
	defer mu.RUnlock()
	return slices.Contains(stripped, entry{list, id})
}

// Active reports whether this process strips anything. It is false in Stock
// Ompig.
func Active() bool {
	mu.RLock()
	defer mu.RUnlock()
	return len(stripped) > 0
}

// IDs lists the stripped IDs of list in the order they were stripped. It is empty in Stock Ompig.
func IDs(list string) []string {
	mu.RLock()
	defer mu.RUnlock()
	var ids []string
	for _, e := range stripped {
		if e.list == list {
			ids = append(ids, e.id)
		}
	}
	return ids
}

// Error is the error a request for a stripped built-in reports. what names
// the built-in for the user, for example "HTML export".
// A process that knows its Ompiglet (SetPiglet) also names it and says how to keep the built-in: a strip is opt-in, so the author who wrote
// the entry can remove it.
func Error(what, list, id string) error {
	mu.RLock()
	name := ompiglet
	mu.RUnlock()
	if name == "" {
		return fmt.Errorf("%s is stripped from this Ompiglet (strip.%s: %s)", what, list, id)
	}
	return fmt.Errorf("%s is stripped from the Ompiglet %q (strip.%s: %s); to keep it, remove %s from strip.%s in the Ompiglet manifest and rebuild", what, name, list, id, id, list)
}
