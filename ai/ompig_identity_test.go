package ai

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/tmih06/ompig/internal/coding/pigidentity"
)

// Pi's openai-codex login sends its own client name as the authorize URL's
// originator (packages/ai/src/auth/oauth/openai-codex.ts createAuthorizationFlow
// originator = "pi"). Ompig sends its own (D26): OpenAI must not attribute Ompig's
// login to Pi.
func TestLoginOpenAICodexAuthorizeURLNamesOmpigAsOriginator(t *testing.T) {
	holdCodexCallbackPort(t)
	stubCodexTokenEndpoint(t)
	var authURL string
	if _, err := LoginOpenAICodex(context.Background(), OAuthLoginCallbacks{
		OnAuth: func(info OAuthAuthInfo) {
			authURL = info.URL
			// openai-codex.ts:376-379.
			if want := "A browser window should open. Complete login to finish."; info.Instructions != want {
				t.Errorf("instructions = %q, want %q", info.Instructions, want)
			}
		},
		OnManualCodeInput: func() (string, error) { return "manual-code", nil },
	}); err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Query().Get("originator"); got != pigidentity.CodexOriginator || got == "" || strings.EqualFold(got, "pi") {
		t.Fatalf("authorize originator = %q, want Ompig's %q", got, pigidentity.CodexOriginator)
	}
}

// The Codex websocket handshake sets originator itself (openai-codex-responses.ts
// buildBaseCodexHeaders), separately from the SSE path: both carry Ompig's value.
func TestCodexWebSocketHeadersNameOmpigAsOriginator(t *testing.T) {
	headers := codexWebSocketHeaders(ClientIdentityTarget{}, nil, nil, nil, "token", "acct", "request")
	if got := headers.Get("originator"); got != pigidentity.CodexOriginator || got == "" || strings.EqualFold(got, "pi") {
		t.Fatalf("websocket originator = %q, want Ompig's %q", got, pigidentity.CodexOriginator)
	}
}
