package builtin_test

import (
	"testing"
)

// docs/specs/builtin-codemode-tool-search.md, "Strip": a build with the `ompig_strip_codemode` tag links neither the sandbox,
// its wasm module nor wazero; the untagged build links all of them.
func TestStripCodemodeBuildOmitsCodemodeAndItsEngine(t *testing.T) {
	engine := []string{"github.com/tetratelabs/wazero", "github.com/tmih06/ompig/codemode", "github.com/tmih06/ompig/coding/extension/builtin/codemode"}
	stock := goListDeps(t, "", ompigModule+"/cmd/ompig")
	stripped := goListDeps(t, "ompig_strip_codemode", ompigModule+"/cmd/ompig")
	for _, pkg := range engine {
		if !linksPackage(stock, pkg) {
			t.Errorf("the stock cmd/ompig does not link %s", pkg)
		}
		if linksPackage(stripped, pkg) {
			t.Errorf("a ompig_strip_codemode build of cmd/ompig links %s", pkg)
		}
	}
}
