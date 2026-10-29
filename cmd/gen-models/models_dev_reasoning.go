package main

import "github.com/tmih06/ompig/internal/modelgen"

type modelsDevReasoningOption = modelgen.ModelsDevReasoningOption

func getEffortThinkingLevelMap(options []modelsDevReasoningOption) map[string]*string {
	return modelgen.GetEffortThinkingLevelMap(options)
}
