package cli

import (
	"github.com/tmih06/ompig/ai"
	"github.com/tmih06/ompig/coding"
	"github.com/tmih06/ompig/coding/extension"
)

// extensionScopedModels materializes only an explicit scope. Extension getters read the Session's retained result instead of resolving models again.
func extensionScopedModels(services *coding.AgentSessionServices, patterns []string) []extension.ScopedModel {
	// upstream: packages/coding-agent/src/main.ts:main
	if len(patterns) == 0 {
		return []extension.ScopedModel{}
	}
	modelRuntime := services.ModelRuntime()
	runtime := newLazyStartupModelRuntime(modelRuntime.RuntimeModels, modelRuntime.HasConfiguredAuth)
	scoped, _ := resolveModelScopeFromModels(patterns, runtime.getAvailable())
	result := make([]extension.ScopedModel, 0, len(scoped))
	for _, entry := range scoped {
		model := modelRuntime.GetModel(entry.Model.Provider, entry.Model.ID)
		if model != nil {
			result = append(result, extension.ScopedModel{Model: model, ThinkingLevel: ai.ModelThinkingLevel(entry.ThinkingLevel)})
		}
	}
	return result
}
