//go:build parity

package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckPigFreshnessDetectsStaleBinary(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd", "ompig"), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "ompig")
	if err := os.WriteFile(bin, []byte("ompig"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "cmd", "ompig", "main.go")
	if err := os.WriteFile(src, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-10 * time.Second)
	newer := time.Now().Add(10 * time.Second)
	if err := os.Chtimes(bin, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(src, newer, newer); err != nil {
		t.Fatal(err)
	}

	report, err := CheckPigFreshness(root, bin)
	if err != nil {
		t.Fatalf("CheckPigFreshness: %v", err)
	}
	if report.Fresh {
		t.Fatalf("CheckPigFreshness Fresh=true, want stale; report=%+v", report)
	}
	if report.NewestPath != src {
		t.Fatalf("NewestPath=%q want %q", report.NewestPath, src)
	}
}

func TestCheckPigFreshnessIgnoresTestsAndTestdata(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd", "ompig", "testdata"), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "ompig")
	if err := os.WriteFile(bin, []byte("ompig"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-10 * time.Second)
	newer := time.Now().Add(10 * time.Second)
	if err := os.Chtimes(bin, old, old); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		filepath.Join(root, "cmd", "ompig", "main_test.go"),
		filepath.Join(root, "cmd", "ompig", "testdata", "fixture.go"),
	} {
		if err := os.WriteFile(p, []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, newer, newer); err != nil {
			t.Fatal(err)
		}
	}

	report, err := CheckPigFreshness(root, bin)
	if err != nil {
		t.Fatalf("CheckPigFreshness: %v", err)
	}
	if !report.Fresh {
		t.Fatalf("CheckPigFreshness Fresh=false, want fresh; report=%+v", report)
	}
}
