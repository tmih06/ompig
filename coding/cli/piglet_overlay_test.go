package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmih06/ompig/coding/ompiglet"
)

// bakedPigletBinary is a Ompiglet Binary named name whose baked Ompiglet is bakedYAML; it accepts overlays when the baked
// Ompiglet's build block says so (D202).
func bakedPigletBinary(t *testing.T, name, bakedYAML string) pigletBinary {
	t.Helper()
	baked, err := ompiglet.ParseBytes([]byte(bakedYAML))
	if err != nil {
		t.Fatal(err)
	}
	return pigletBinary{
		Is:   true,
		Name: func() string { return name },
		OverlayBase: func() *ompiglet.OverlayBase {
			if baked.Build == nil || !baked.Build.AllowOverlays {
				return nil
			}
			return &ompiglet.OverlayBase{Ompiglet: baked, Digest: "sha256:" + strings.Repeat("ab", 32)}
		},
	}
}

func clearPigletEnv(t *testing.T) {
	t.Helper()
	t.Setenv("OMPIG_OMPIGLET_PATH", "")
	t.Setenv("OMPIG_OMPIGLET_NAME", "")
	t.Setenv("OMPIG_HOME", t.TempDir())
}

func TestPigletBinaryWithoutAllowOverlaysRefusesEverySelector(t *testing.T) {
	clearPigletEnv(t)
	binary := bakedPigletBinary(t, "ompig-tern", "name: ompig-tern\nstrip:\n  tools: [edit]\n")
	if got := pigletBinaryOverlayRefusal(map[string]any{}, binary); got != "" {
		t.Fatalf("no selector refused: %q", got)
	}
	selectors := map[string]func(){
		"--ompiglet":          func() {},
		"OMPIG_OMPIGLET_PATH": func() { t.Setenv("OMPIG_OMPIGLET_PATH", "extra.yaml") },
		"OMPIG_OMPIGLET_NAME": func() { t.Setenv("OMPIG_OMPIGLET_NAME", "extra") },
	}
	for name, selected := range selectors {
		t.Run(name, func(t *testing.T) {
			selected()
			flags := map[string]any{}
			if name == "--ompiglet" {
				flags["ompiglet"] = "extra.yaml"
			}
			got := pigletBinaryOverlayRefusal(flags, binary)
			for _, want := range []string{"Ompiglet Binary (ompig-tern)", "does not accept --ompiglet overlays", "build.allowOverlays: true", "stock ompig"} {
				if !strings.Contains(got, want) {
					t.Fatalf("refusal %q lacks %q", got, want)
				}
			}
		})
	}
}

func TestPigletBinaryWithAllowOverlaysAcceptsEverySelector(t *testing.T) {
	clearPigletEnv(t)
	binary := bakedPigletBinary(t, "ompig-tern", "name: ompig-tern\nstrip:\n  tools: [edit]\nbuild:\n  allowOverlays: true\n")
	for _, flags := range []map[string]any{{"ompiglet": "extra.yaml"}, {}} {
		t.Setenv("OMPIG_OMPIGLET_NAME", "extra")
		if got := pigletBinaryOverlayRefusal(flags, binary); got != "" {
			t.Fatalf("an accepting Binary refused %v: %q", flags, got)
		}
	}
}

func TestResolvePigletLaysAnOverlayOnAnAcceptingBinary(t *testing.T) {
	clearPigletEnv(t)
	binary := bakedPigletBinary(t, "ompig-tern", "name: ompig-tern\nextensions:\n  - name: core\nstrip:\n  tools: [edit]\nbuild:\n  allowOverlays: true\n")
	dir := t.TempDir()
	overlay := filepath.Join(dir, "extra.yaml")
	if err := os.WriteFile(overlay, []byte("name: extra\nextensions:\n  - name: extra\n    origins: [local:./extra]\nstrip:\n  tools: [grep]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, run := range map[string]func() (*ompiglet.Ompiglet, bool){
		"flag": func() (*ompiglet.Ompiglet, bool) {
			return resolvePigletOn(map[string]any{"ompiglet": overlay}, nil, binary)
		},
		"path": func() (*ompiglet.Ompiglet, bool) {
			t.Setenv("OMPIG_OMPIGLET_PATH", overlay)
			return resolvePigletOn(map[string]any{}, nil, binary)
		},
	} {
		t.Run(name, func(t *testing.T) {
			p, baked := run()
			if p == nil || !baked || p.Name != "ompig-tern" {
				t.Fatalf("overlay = %#v baked=%t, want the Binary's Ompiglet with the overlay laid over it", p, baked)
			}
			if len(p.Extensions) != 2 || p.Extensions[0].Name != "core" || p.Extensions[1].Name != "extra" || len(p.Strip.Tools) != 2 {
				t.Fatalf("effective Ompiglet = %#v", p)
			}
		})
	}
}

func TestResolvePigletNeverFallsBackToTheBinaryWhenAnOverlayDoesNotResolve(t *testing.T) {
	clearPigletEnv(t)
	binary := bakedPigletBinary(t, "ompig-tern", "name: ompig-tern\nextensions:\n  - name: core\nbuild:\n  allowOverlays: true\n")
	dir := t.TempDir()
	conflict := filepath.Join(dir, "conflict.yaml")
	if err := os.WriteFile(conflict, []byte("name: conflict\nextensions:\n  - name: core\n    origins: [local:./other]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if p, _ := resolvePigletOn(map[string]any{"ompiglet": conflict}, nil, binary); p != nil {
		t.Fatalf("a refused overlay resolved to %#v", p)
	}
	if p, _ := resolvePigletOn(map[string]any{"ompiglet": filepath.Join(dir, "missing.yaml")}, nil, binary); p != nil {
		t.Fatalf("a missing overlay resolved to %#v", p)
	}
	t.Setenv("OMPIG_OMPIGLET_NAME", "no-such-ompiglet")
	if p, _ := resolvePigletOn(map[string]any{}, nil, binary); p != nil {
		t.Fatalf("an unresolvable OMPIG_OMPIGLET_NAME fell back to %#v", p)
	}
}
