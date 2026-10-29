//go:build ompig_strip_mistral_conversations

package ai

import "github.com/tmih06/ompig/internal/pigstrip"

// This build has no mistral-conversations provider.
func init() { pigstrip.Strip(pigstrip.ListAPIs, string(APIMistralConversations)) }

// NewMistralAPIProvider reports that this Ompiglet strips mistral-conversations.
func NewMistralAPIProvider(cfg MistralConfig) (Provider, error) {
	return nil, strippedMistralError(&cfg)
}
