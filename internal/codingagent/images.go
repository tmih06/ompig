package codingagent

import (
	"github.com/tmih06/ompig/agent"
	"github.com/tmih06/ompig/internal/imageprocessing"
)

func NormalizeToolResultImages(result agent.AgentToolResult, autoResize bool) agent.AgentToolResult {
	return imageprocessing.NormalizeToolResultImages(result, autoResize)
}

func DetectSupportedImageMimeType(data []byte) string {
	return imageprocessing.DetectSupportedImageMimeType(data)
}
