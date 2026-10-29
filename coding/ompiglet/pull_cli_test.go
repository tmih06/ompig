package ompiglet

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pigletrelease "github.com/tmih06/ompig/coding/ompiglet/release"
	"github.com/tmih06/ompig/coding/ompiglet/signature"
)

func TestRunCommandPullInstallsAndListsSignedRelease(t *testing.T) {
	t.Setenv("OMPIG_HOME", t.TempDir())
	t.Setenv("OMPIG_OMPIGLET_PULL_ALLOW_LOOPBACK_HTTP", "1")
	t.Chdir(t.TempDir())
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	const target = "testos/testarch"
	binaryPath := filepath.Join(t.TempDir(), "ompig-porter")
	if err := os.WriteFile(binaryPath, []byte("test executable\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest := "sha256:" + strings.Repeat("a", 64)
	if _, err := signature.Sign(binaryPath, signature.Manifest{
		Ompiglet: "porter", ReleaseVersion: "1.2.3", Target: target, OmpigVersion: "ompig-test",
		PigletDigest: digest, SourceDigest: digest, ResolutionDigest: digest, ComponentPlanDigest: digest,
	}, key); err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	var index []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/ompiglet-release.json":
			_, _ = w.Write(index)
		case "/ompig-porter":
			_, _ = w.Write(binary)
		default:
			http.NotFound(w, request)
		}
	}))
	defer server.Close()
	sum := sha256.Sum256(binary)
	index, err = pigletrelease.Sign(pigletrelease.Index{
		Ompiglet: "porter", Version: "1.2.3", OmpigVersion: "ompig-test", SourceRef: "npm:@example/porter@1.2.3",
		Binaries: map[string]pigletrelease.Binary{target: {URL: server.URL + "/ompig-porter", SHA256: hex.EncodeToString(sum[:]), Size: int64(len(binary))}},
	}, key)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr strings.Builder
	code := RunCommand([]string{"ompiglet", "pull", server.URL + "/ompiglet-release.json", "--target", target, "--json", "--no-input"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "Pulled porter 1.2.3") {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	code = RunCommand([]string{"ompiglet", "list", "--json", "--no-input"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"name":"porter"`) || !strings.Contains(stdout.String(), `"target":"testos/testarch"`) {
		t.Fatalf("list code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	code = RunCommand([]string{"ompiglet", "remove", "porter", "--binary"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "removed Ompiglet Binary current pointer") {
		t.Fatalf("remove code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	code = RunCommand([]string{"ompiglet", "list", "--json"}, &stdout, &stderr)
	if code != 0 || stdout.String() != "{\"ompiglets\":[]}\n" {
		t.Fatalf("list after remove code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}
