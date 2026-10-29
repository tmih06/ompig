package builtin_test

import (
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/tmih06/ompig/coding/extension/builtin"
	"github.com/tmih06/ompig/internal/pigstrip"
)

const ompigModule = "github.com/tmih06/ompig"

// goListDeps runs `go list [-tags tags] -deps pkg` and returns the listed packages.
func goListDeps(t *testing.T, tags, pkg string) []string {
	t.Helper()
	return goList(t, tags, "-deps", pkg)
}

func goList(t *testing.T, tags string, args ...string) []string {
	t.Helper()
	if testing.Short() {
		t.Skip("go list of cmd/ompig in -short mode")
	}
	cmd := []string{"list"}
	if tags != "" {
		cmd = append(cmd, "-tags", tags)
	}
	out, err := exec.Command("go", append(cmd, args...)...).Output()
	if err != nil {
		t.Fatalf("go %s: %v", strings.Join(append(cmd, args...), " "), err)
	}
	return strings.Fields(string(out))
}

// linksPackage reports whether pkg or one of its subpackages is in the list.
func linksPackage(list []string, pkg string) bool {
	return slices.ContainsFunc(list, func(listed string) bool { return listed == pkg || strings.HasPrefix(listed, pkg+"/") })
}

// docs/design/builtin-mcp.md: a ompig_strip_mcp build of cmd/ompig links neither the MCP client (`mcp`, `mcp/oauth`) nor the
// extension (`coding/mcpext`).
func TestStripMcpBuildOmitsMcpClientAndExtension(t *testing.T) {
	stock := goListDeps(t, "", ompigModule+"/cmd/ompig")
	stripped := goListDeps(t, "ompig_strip_mcp", ompigModule+"/cmd/ompig")
	for _, pkg := range []string{ompigModule + "/mcp", ompigModule + "/mcp/oauth", ompigModule + "/coding/mcpext"} {
		if !slices.Contains(stock, pkg) {
			t.Errorf("the stock cmd/ompig does not link %s", pkg)
		}
		if linksPackage(stripped, pkg) {
			t.Errorf("a ompig_strip_mcp build of cmd/ompig links %s", pkg)
		}
	}
}

// The tool-search registration (tool_search_on.go) is the builtin package's only import of toolsearch, so a
// ompig_strip_tool_search build drops that import. codemode reuses toolsearch's helpers, so the package itself leaves cmd/ompig
// only when codemode is stripped too.
func TestStripToolSearchBuildOmitsToolSearch(t *testing.T) {
	toolsearch := ompigModule + "/coding/extension/builtin/toolsearch"
	importsOf := func(tags string) []string {
		return goList(t, tags, "-f", `{{join .Imports "\n"}}`, ompigModule+"/coding/extension/builtin")
	}
	if !slices.Contains(importsOf(""), toolsearch) {
		t.Errorf("the stock builtin package does not import %s", toolsearch)
	}
	if slices.Contains(importsOf("ompig_strip_tool_search"), toolsearch) {
		t.Errorf("a ompig_strip_tool_search build of the builtin package imports %s", toolsearch)
	}
	if !slices.Contains(goListDeps(t, "", ompigModule+"/cmd/ompig"), toolsearch) {
		t.Errorf("the stock cmd/ompig does not link %s", toolsearch)
	}
	if slices.Contains(goListDeps(t, "ompig_strip_tool_search,ompig_strip_codemode", ompigModule+"/cmd/ompig"), toolsearch) {
		t.Errorf("a ompig_strip_tool_search,ompig_strip_codemode build of cmd/ompig links %s", toolsearch)
	}
}

// ompig-login's `/sprite` registration (ompig_login_on.go) is the builtin package's only import of coding/piglogin. The package
// itself stays in cmd/ompig because internal/codingagent draws the sprite, so this asserts the registration leaves: the
// tagged builtin package compiles ompig_login_off.go instead and no longer imports piglogin.
func TestStripPigLoginBuildOmitsSpriteRegistration(t *testing.T) {
	piglogin := ompigModule + "/coding/piglogin"
	listOf := func(tags string) []string {
		return goList(t, tags, "-f", `{{join .GoFiles "\n"}}{{"\n"}}{{join .Imports "\n"}}`, ompigModule+"/coding/extension/builtin")
	}
	stock, stripped := listOf(""), listOf("ompig_strip_ompig_login")
	if !slices.Contains(stock, "ompig_login_on.go") || slices.Contains(stock, "ompig_login_off.go") || !slices.Contains(stock, piglogin) {
		t.Errorf("stock builtin package = %v, want ompig_login_on.go importing %s", stock, piglogin)
	}
	if slices.Contains(stripped, "ompig_login_on.go") || !slices.Contains(stripped, "ompig_login_off.go") || slices.Contains(stripped, piglogin) {
		t.Errorf("ompig_strip_ompig_login builtin package = %v, want ompig_login_off.go without %s", stripped, piglogin)
	}
}

// A built-in a Ompiglet strips at runtime has no registry row, as in a Binary that compiled it out; the others keep their
// order. Unstripped, the registry is the stock list.
func TestRuntimeStripDropsBuiltinRows(t *testing.T) {
	stock := entryNamesOf(builtin.All(builtin.Options{}))
	for _, name := range []string{"codemode", "tool-search", "mcp", "ompig-login"} {
		if pigstrip.Has(pigstrip.ListExtensions, name) {
			continue // compiled out of this test binary: its OFF shim recorded the strip
		}
		if !slices.Contains(stock, name) {
			t.Fatalf("stock registry %v lacks %s", stock, name)
		}
		t.Run(name, func(t *testing.T) {
			t.Cleanup(pigstrip.Strip(pigstrip.ListExtensions, name))
			got := entryNamesOf(builtin.All(builtin.Options{}))
			if want := slices.DeleteFunc(slices.Clone(stock), func(n string) bool { return n == name }); !slices.Equal(got, want) {
				t.Errorf("registry = %v, want %v", got, want)
			}
		})
	}
	if got := entryNamesOf(builtin.All(builtin.Options{})); !slices.Equal(got, stock) {
		t.Errorf("registry after undo = %v, want %v", got, stock)
	}
}

func entryNamesOf(entries []builtin.Extension) []string {
	var out []string
	for _, entry := range entries {
		out = append(out, entry.Name)
	}
	return out
}
