//go:build !ompig_strip_ompig_login

package builtin_test

import (
	"slices"
	"testing"

	"github.com/tmih06/ompig/coding/extension"
	"github.com/tmih06/ompig/coding/extension/factoryload"

	"github.com/tmih06/ompig/coding/extension/builtin"
)

// ompig-login is Ompig's own built-in extension (D2, owner decision 2026-10-01): the `/sprite` command of the sprite login.
// Upstream's index.ts lists no such entry, so it comes after upstream's and another extension's `/sprite` replaces it.
func entryNames(entries []builtin.Extension) []string {
	var out []string
	for _, entry := range entries {
		out = append(out, entry.Name)
	}
	return out
}

func TestAllListsPigLoginLastAsReplaceable(t *testing.T) {
	all := builtin.All(builtin.Options{})
	if len(all) == 0 || all[len(all)-1].Name != "ompig-login" {
		t.Fatalf("registry %v, want ompig-login last", entryNames(all))
	}
	if !all[len(all)-1].Replaceable {
		t.Error("ompig-login is not replaceable: an extension that registers /sprite could not take the command")
	}
	if names := entryNames(all); slices.Contains(names[:len(names)-1], "ompig-login") {
		t.Errorf("ompig-login listed twice in %v", names)
	}
}

func TestPigLoginResolvesAndRegistersTheSpriteCommand(t *testing.T) {
	entry, err := builtin.Resolve("builtin:ompig-login", builtin.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Path() != "builtin:ompig-login" || entry.Factory == nil {
		t.Fatalf("entry = %+v", entry)
	}
	ext, err := factoryload.LoadExtensionFromFactory(entry.Factory, ".", extension.CreateEventBus(), extension.CreateExtensionRuntime(), "builtin:ompig-login")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ext.Commands["sprite"]; !ok || !slices.Equal(ext.CommandOrder, []string{"sprite"}) {
		t.Fatalf("commands = %v (order %v), want /sprite", ext.Commands, ext.CommandOrder)
	}
}
