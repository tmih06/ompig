package coding

import (
	"context"

	"github.com/tmih06/ompig/agent"
	"github.com/tmih06/ompig/ai"
	"github.com/tmih06/ompig/internal/imageprocessing"
)

// prepareToolResult runs after every extension hook; a hook may change the
// active model or replace images. The profile is the limits model's (agent-session.ts:707). Failed processing preserves the source image.
func (s *Session) prepareToolResult(_ context.Context, result agent.AgentToolResult) agent.AgentToolResult {
	var options *ai.ModelImageResizeOptions
	if model := s.limitsModel(); model != nil && model.InputLimits != nil && model.InputLimits.Images != nil {
		options = model.InputLimits.Images.Resize
	}
	return imageprocessing.NormalizeToolResultImagesWithOptions(result, s.SettingsManager().GetImageAutoResize(), options)
}
