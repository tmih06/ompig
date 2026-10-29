package release

import (
	"context"
	"crypto/ed25519"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/tmih06/ompig/coding/ompiglet/signature"
)

func TestFetchIndexVerifiesTheSignedIndexAndInstallsNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("OMPIG_HOME", home)
	t.Setenv("OMPIG_OMPIGLET_PULL_ALLOW_LOOPBACK_HTTP", "1")
	key := newKey(t)
	server, indexURL := releaseServer(t, key, releaseSpec{ompiglet: "porter", version: "1.2.3", target: testTarget, ompigVersion: "ompig-test", binary: []byte("binary")})
	defer server.Close()

	verified, err := FetchIndex(context.Background(), indexURL, Options{Version: "1.2.3"})
	if err != nil {
		t.Fatal(err)
	}
	if verified.Index.Ompiglet != "porter" || verified.Index.Version != "1.2.3" || verified.Index.Signer.KeyID != signature.KeyID(key.Public().(ed25519.PublicKey)) {
		t.Fatalf("index = %+v", verified.Index)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Fatalf("fetching an index wrote managed state: %v", entries)
	}
}

func TestFetchIndexRefusesWhatPullWouldRefuse(t *testing.T) {
	t.Setenv("OMPIG_HOME", t.TempDir())
	t.Setenv("OMPIG_OMPIGLET_PULL_ALLOW_LOOPBACK_HTTP", "1")
	key := newKey(t)
	for name, tc := range map[string]struct {
		spec    releaseSpec
		version string
		want    string
	}{
		"bad signature":    {releaseSpec{ompiglet: "porter", version: "1.2.3", target: testTarget, ompigVersion: "p", binary: []byte("b"), badIndexSignature: true}, "1.2.3", "Ompiglet release index"},
		"version mismatch": {releaseSpec{ompiglet: "porter", version: "1.2.3", target: testTarget, ompigVersion: "p", binary: []byte("b")}, "2.0.0", "version 1.2.3, requested 2.0.0"},
	} {
		t.Run(name, func(t *testing.T) {
			server, indexURL := releaseServer(t, key, tc.spec)
			defer server.Close()
			if _, err := FetchIndex(context.Background(), indexURL, Options{Version: tc.version}); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestFetchIndexReportsAMissingReleaseAndBadReferences(t *testing.T) {
	t.Setenv("OMPIG_HOME", t.TempDir())
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	t.Setenv("OMPIG_OMPIGLET_PULL_ALLOW_LOOPBACK_HTTP", "1")
	if _, err := FetchIndex(context.Background(), server.URL+"/ompiglet-release.json", Options{}); err == nil || !strings.Contains(err.Error(), "fetch Ompiglet release index: 404 Not Found") {
		t.Fatalf("missing release: %v", err)
	}
	if _, err := FetchIndex(context.Background(), "ftp://example.test/x", Options{}); err == nil {
		t.Fatal("accepted a reference that is neither a GitHub ref nor an HTTPS URL")
	}
	if _, err := FetchIndex(context.Background(), "github:acme/porter", Options{}); err == nil {
		t.Fatal("accepted a GitHub reference without a version")
	}
}
