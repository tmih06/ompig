package sdk

import "github.com/tmih06/ompig/extensions/sdk/json"

// MarshalJSON preserves a present empty customPrompt (Pi system-prompt.ts:54-57).
func (o SystemPromptOptions) MarshalJSON() ([]byte, error) {
	type fields SystemPromptOptions
	var custom *string
	if o.CustomPromptSet || o.CustomPrompt != "" {
		custom = &o.CustomPrompt
	}
	return json.MarshalJS(struct {
		fields
		CustomPrompt *string `json:"customPrompt,omitempty"`
	}{fields: fields(o), CustomPrompt: custom})
}

// UnmarshalJSON preserves customPrompt presence in the command result.
func (o *SystemPromptOptions) UnmarshalJSON(data []byte) error {
	type fields SystemPromptOptions
	var wire struct {
		fields
		CustomPrompt *string `json:"customPrompt"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	*o = SystemPromptOptions(wire.fields)
	o.CustomPromptSet = wire.CustomPrompt != nil
	if wire.CustomPrompt != nil {
		o.CustomPrompt = *wire.CustomPrompt
	}
	return nil
}
