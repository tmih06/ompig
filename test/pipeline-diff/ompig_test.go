package pipelinediff

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tmih06/ompig/agent"
	"github.com/tmih06/ompig/ai"
	"github.com/tmih06/ompig/internal/requestvalidity"
)

// ompigRun sends one case through Ompig's production pipeline: the agent loop's ConvertToLLM(NormalizeMessages(...)), then the model's API implementation. Without normalize it skips NormalizeMessages, which Pi has no counterpart of, so only convertToLlm and the ai package's transformMessages and converters remain. It returns the contract violations the request checker recorded.
func ompigRun(c testCase, baseURL string, normalize bool) (violations []string, streamError string, err error) {
	model := ai.GetBuiltinModel(c.Target.Provider, c.Target.Model)
	if model == nil {
		return nil, "", fmt.Errorf("unknown model %s/%s", c.Target.Provider, c.Target.Model)
	}
	model.ProviderMeta.BaseURL = baseURL
	encoded, err := json.Marshal(c.Messages)
	if err != nil {
		return nil, "", err
	}
	var messages []agent.AgentMessage
	if err := json.Unmarshal(encoded, &messages); err != nil {
		return nil, "", fmt.Errorf("decode agent messages: %w", err)
	}
	if normalize {
		messages = agent.NormalizeMessages(messages, model)
	}
	llm := agent.ConvertToLLM(messages)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ctx, collector := requestvalidity.Collect(ctx)
	stream, err := ai.Stream(ctx, model, ai.Context{SystemPrompt: c.SystemPrompt, Messages: llm}, ai.StreamOptions{APIKey: "test-key"})
	if err != nil {
		return collector.Violations(), "", err
	}
	result := stream.Result()
	return collector.Violations(), result.ErrorMessage, nil
}
