package turn

import (
	"testing"

	"github.com/tmih06/ompig/durable/core/jv"
)

func BenchmarkClassifyFailedMessage(b *testing.B) {
	message := jv.NewObject()
	message.Set("stopReason", "error")
	message.Set("errorMessage", "503 service unavailable: upstream connect error")
	message.Set("provider", "openai")
	for b.Loop() {
		IsContextOverflow(message)
		IsRetryableAssistantError(message)
	}
}
