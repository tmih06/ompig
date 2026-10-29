package agenthub

// The agent transcript view: what the hub's Enter opens. It renders one
// subagent's messages as a scrollable text card, which is the read-only
// inspect path for a live, parked or killed agent.
//
// Upstream: omp docs/agent-hub.md "Read and steer a subagent".

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmih06/ompig/agent"
	"github.com/tmih06/ompig/ai"
	"github.com/tmih06/ompig/tui"
)

// Transcript is a read-only message card.
type Transcript struct {
	ID       string
	messages []agent.AgentMessage
	scroll   int
	width    int
	height   int
}

// NewTranscript builds a transcript view.
func NewTranscript(id string, messages []agent.AgentMessage, width, height int) *Transcript {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 20
	}
	copied := make([]agent.AgentMessage, len(messages))
	copy(copied, messages)
	return &Transcript{ID: id, messages: copied, width: width, height: height}
}

var _ tui.Component = (*Transcript)(nil)

// Invalidate satisfies the component contract; the card renders from a snapshot.
func (t *Transcript) Invalidate() {}

// HandleInput scrolls the view.
func (t *Transcript) HandleInput(data string) {
	switch data {
	case "\x1b[A", "k":
		if t.scroll > 0 {
			t.scroll--
		}
	case "\x1b[B", "j":
		t.scroll++
	case "\x1b[5~", "b":
		t.scroll = max(0, t.scroll-10)
	case "\x1b[6~", " ":
		t.scroll += 10
	case "g":
		t.scroll = 0
	case "G":
		t.scroll = 1 << 30
	}
}

// Render draws the visible window of the transcript.
func (t *Transcript) Render(width int) []string {
	lines := t.lines(width)
	visible := max(t.height-2, 1)
	start := t.scroll
	if start > len(lines) {
		start = max(0, len(lines)-visible)
	}
	if start < 0 {
		start = 0
	}
	end := min(len(lines), start+visible)
	out := make([]string, 0, visible+1)
	out = append(out, pad(fmt.Sprintf(" Agent %s · %d messages · j/k scroll · esc close", t.ID, len(t.messages)), width))
	out = append(out, lines[start:end]...)
	return out
}

// lines renders every message into display lines.
func (t *Transcript) lines(width int) []string {
	if len(t.messages) == 0 {
		return []string{pad(" (no messages yet)", width)}
	}
	var out []string
	for _, message := range t.messages {
		out = append(out, t.messageLines(message, width)...)
	}
	return out
}

// messageLines renders one message.
func (t *Transcript) messageLines(message agent.AgentMessage, width int) []string {
	var out []string
	switch {
	case message.User != nil:
		out = append(out, pad("▌ user", width))
		out = append(out, wrap(plainUserText(message.User), width-2, "│ ")...)
	case message.Assistant != nil:
		assistant := message.Assistant
		out = append(out, pad("▌ assistant", width))
		for _, block := range assistant.Content {
			switch typed := block.(type) {
			case ai.TextContent:
				if strings.TrimSpace(typed.Text) != "" {
					out = append(out, wrap(typed.Text, width-2, "│ ")...)
				}
			case ai.ThinkingContent:
				out = append(out, wrap(muted("thinking: "+typed.Thinking), width-2, "│ ")...)
			case ai.ToolCall:
				out = append(out, wrap("→ "+typed.Name+" "+compactJSON(typed.Arguments), width-2, "│ ")...)
			}
		}
		if assistant.ErrorMessage != "" {
			out = append(out, wrap("error: "+assistant.ErrorMessage, width-2, "│ ")...)
		}
	case message.ToolResult != nil:
		result := message.ToolResult
		label := "← " + result.ToolName
		if result.IsError {
			label += " (error)"
		}
		out = append(out, pad("  "+label, width))
		text := toolResultText(result)
		if text != "" {
			out = append(out, wrap(text, width-4, "  │ ")...)
		}
	}
	return out
}

// plainUserText flattens a user message's content.
func plainUserText(message *agent.UserMessage) string {
	if message == nil {
		return ""
	}
	switch content := message.Content.(type) {
	case ai.UserText:
		return string(content)
	case ai.UserContentBlocks:
		return ai.ContentText(content)
	default:
		return ""
	}
}

// toolResultText flattens a tool result's content.
func toolResultText(result *agent.ToolResultMessage) string {
	var b strings.Builder
	for _, block := range result.Content {
		if text, ok := block.(ai.TextContent); ok {
			b.WriteString(text.Text)
		}
	}
	return b.String()
}

// compactJSON renders arguments on one line.
func compactJSON(raw ai.JsonObject) string {
	if raw == nil {
		return ""
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return ""
	}
	text := string(encoded)
	if len(text) > 120 {
		text = text[:120] + "…"
	}
	return text
}

// wrap breaks text into width-sized lines with a prefix on each.
func wrap(text string, width int, prefix string) []string {
	if width < 8 {
		width = 8
	}
	var out []string
	for paragraph := range strings.SplitSeq(text, "\n") {
		if paragraph == "" {
			out = append(out, pad(prefix, width))
			continue
		}
		remaining := []rune(paragraph)
		for len(remaining) > 0 {
			chunk := remaining
			if len(chunk) > width {
				chunk = chunk[:width]
			}
			out = append(out, pad(prefix+string(chunk), width))
			remaining = remaining[len(chunk):]
		}
	}
	// A block never exceeds the card height contribution unbounded; the caller
	// scrolls.
	return out
}

// muted marks secondary text.
func muted(text string) string { return text }
