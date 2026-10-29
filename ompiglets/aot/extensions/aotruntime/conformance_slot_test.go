package aotruntime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestWriteConformanceSlot builds the dynamic host (jiti compiled in, no extension compiled in) into a runtime slot at $OMPIG_AOT_CONFORMANCE_SLOT, so that
// Ompig's extension conformance suite can run on the compiled executable: its extensions are written per test and not known at build time.
// `make test-aot-conformance` runs it, then the suite with OMPIG_TS_RUNTIME=aot and OMPIG_TS_RUNTIMES_DIR set to the slot. It skips unless the compiler, a
// runtime tree and the slot directory are named.
func TestWriteConformanceSlot(t *testing.T) {
	slot := os.Getenv("OMPIG_AOT_CONFORMANCE_SLOT")
	if slot == "" {
		t.Skip("set OMPIG_AOT_CONFORMANCE_SLOT (and OMPIG_AOT_TEST_COMPILER, OMPIG_AOT_TEST_RUNTIME_TREE) to build the conformance slot")
	}
	compiler, tree := realBuildInputs(t)
	work := filepath.Join(slot, ".work")
	if err := os.RemoveAll(slot); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	cpu := "native"
	if os.Getenv("OMPIG_AOT_CONFORMANCE_BASELINE") != "" {
		cpu = "baseline"
	}
	result, err := build(context.Background(), buildOptions{
		Compiler: compiler, Platform: platformName(), CPU: cpu, Version: "ompig-aot-host conformance", Dynamic: true,
		Request: Request{RuntimeTree: tree}, WorkDir: work, Log: os.Stderr,
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(slot, "aot")
	if err := os.Rename(result.Dir, dir); err != nil {
		t.Fatal(err)
	}
	digest, err := fileSHA256(filepath.Join(dir, "ompig-ts-host"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf(`{"id":"aot","kind":"executable","version":"conformance","platform":%q,"provider":"ompig-aot-conformance","image":{"executable":"ompig-ts-host","sha256":%q}}`, platformName(), digest)
	if err := os.WriteFile(filepath.Join(dir, "runtime.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.RemoveAll(work)
}

// TestBuildSlotRequests is the benchmark's build step (bench/bolt-aot): it runs the Ompiglet's worker once over the slot at $OMPIG_AOT_BUILD_SLOT, so that
// every build request Ompig recorded there becomes an image. The compiler downloads from $OMPIG_AOT_RELEASE_URL, or the Pi-Bolt release, and is verified as it
// is for a user.
func TestBuildSlotRequests(t *testing.T) {
	slot := os.Getenv("OMPIG_AOT_BUILD_SLOT")
	if slot == "" {
		t.Skip("set OMPIG_AOT_BUILD_SLOT to build the requests recorded in a runtime slot")
	}
	w := newWorker(slot, platformName(), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	w.tick(ctx)
	requests, err := w.store.readRequests()
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 0 {
		t.Fatalf("%d build requests remain after the build", len(requests))
	}
}
