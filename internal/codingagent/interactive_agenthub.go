package codingagent

// The Agent Hub overlay: a live roster of the session's subagents with
// per-agent activity, usage and controls (focus, message, revive, kill).
//
// Upstream: omp docs/agent-hub.md (Alt+A). The roster is the subagent
// registry; the overlay renders it and applies control actions through the
// session handle's AgentHubHandle surface.

import (
	"context"
	"fmt"

	"github.com/tmih06/ompig/agent"
	"github.com/tmih06/ompig/coding/subagent"
	"github.com/tmih06/ompig/internal/codingagent/agenthub"
	"github.com/tmih06/ompig/tui"
)

// AgentHubHandle is the part of the session the Agent Hub drives. A session
// that can spawn subagents implements it; one that cannot leaves the Hub
// unavailable.
//
// The interface lives here, not in coding/, so the TUI package does not depend
// on the coding package (which imports this one).
type AgentHubHandle interface {
	// SubagentRegistry is the live agent roster.
	SubagentRegistry() *subagent.Registry
	// SubagentJobs is the background job manager, or nil.
	SubagentJobs() *subagent.Jobs
	// SubagentSteer delivers a message to an agent, queueing it when a turn is
	// running and reviving a parked agent first.
	SubagentSteer(ctx context.Context, id, message string) error
	// SubagentRevive restores a parked or killed agent so it can run again.
	SubagentRevive(ctx context.Context, id string) error
	// SubagentKill stops an agent and disposes its live session.
	SubagentKill(ctx context.Context, id string) error
	// SubagentTranscript returns an agent's messages for the transcript view.
	SubagentTranscript(id string) ([]agent.AgentMessage, bool)
}

// openAgentHub shows the roster overlay. It is a no-op when the session has no
// subagent surface, so the key is harmless in a session that cannot spawn.
func (m *InteractiveMode) openAgentHub(ctx context.Context) {
	session := m.currentSession()
	if session == nil {
		m.showError("No active session")
		return
	}
	controls, ok := m.opts.SessionHandle.(AgentHubHandle)
	if !ok || controls.SubagentRegistry() == nil {
		m.showError("Subagents are unavailable in this session")
		return
	}
	if m.tuiInst == nil {
		return
	}
	width, height := m.hubSize()
	model := agenthub.New(agenthub.Options{
		Registry: controls.SubagentRegistry(),
		Jobs:     controls.SubagentJobs(),
		Width:    width,
		Height:   height,
		// A running agent's tool, tokens and elapsed time must move without a
		// keypress, so every registry change repaints the overlay.
		OnChange: func() { m.requestRender() },
	})
	model.SetWide(width >= 100)
	overlayHandle := m.tuiInst.ShowOverlay(model, tui.OverlayOptions{
		Title:          "Agent Hub",
		WidthFraction:  0.95,
		HeightFraction: 0.85,
	})
	listener := tui.TuiInputListener(func(data string) *tui.TuiInputResult {
		return &tui.TuiInputResult{Consume: m.handleAgentHubKey(data)}
	})
	controller := &agentHubController{
		mode:     m,
		hub:      model,
		overlay:  overlayHandle,
		controls: controls,
		listener: listener,
		ctx:      ctx,
	}
	remove := m.tuiInst.AddInputListener(&listener)
	controller.removeInput = remove
	m.agentHub = controller
	m.requestRender()
}

// hubSize reports the roster budget for the current terminal.
func (m *InteractiveMode) hubSize() (int, int) {
	width, height := 100, 20
	if m.tuiInst != nil {
		if w := m.tuiInst.Width(); w > 0 {
			width = w
		}
		if h := m.tuiInst.Height(); h > 0 {
			height = h
		}
	}
	return width, max(8, height-6)
}

// handleAgentHubKey routes one raw key to the open hub. It reports whether the
// hub consumed the input.
func (m *InteractiveMode) handleAgentHubKey(data string) bool {
	controller := m.agentHub
	if controller == nil {
		return false
	}
	key := decodeHubKey(data)
	if key == "" {
		return true
	}
	controller.handleKey(key)
	return true
}

// agentHubController adapts key input to hub requests and applies them to the
// live runtime.
type agentHubController struct {
	mode     *InteractiveMode
	hub      *agenthub.Model
	overlay  *tui.OverlayHandle
	controls AgentHubHandle
	ctx      context.Context
	listener tui.TuiInputListener
	// removeInput detaches the raw-input listener.
	removeInput func()
	closed      bool
}

// handleKey processes one normalized key.
func (c *agentHubController) handleKey(key string) {
	if c.closed {
		return
	}
	request := c.hub.HandleInput(key)
	switch request.Action {
	case agenthub.ActionClose:
		c.close()
		return
	case agenthub.ActionFocus:
		c.focus(request.AgentID)
	case agenthub.ActionRevive:
		c.apply(request, func(ctx context.Context) error {
			return c.controls.SubagentRevive(ctx, request.AgentID)
		})
	case agenthub.ActionKill:
		c.apply(request, func(ctx context.Context) error {
			return c.controls.SubagentKill(ctx, request.AgentID)
		})
	case agenthub.ActionMessage:
		if request.Message == "" {
			// The first press only opens the editor.
			c.mode.requestRender()
			return
		}
		c.apply(request, func(ctx context.Context) error {
			return c.controls.SubagentSteer(ctx, request.AgentID, request.Message)
		})
	}
	c.mode.requestRender()
}

// decodeHubKey normalizes a terminal key sequence to the hub's key names.
func decodeHubKey(data string) string {
	switch data {
	case "\x1b":
		return "esc"
	case "\r", "\n":
		return "enter"
	case "\x7f", "\b":
		return "backspace"
	case "\x03":
		return "ctrl+c"
	case "\x1b[A":
		return "up"
	case "\x1b[B":
		return "down"
	case "\x1b[C":
		return "right"
	case "\x1b[D":
		return "left"
	}
	if len(data) == 1 && data >= " " {
		return data
	}
	return ""
}

// apply runs a control action off the input loop and reports its outcome.
func (c *agentHubController) apply(request agenthub.Request, fn func(context.Context) error) {
	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		err := fn(ctx)
		c.mode.runOnMain(ctx, func() {
			if err != nil {
				c.mode.showError(err.Error())
				return
			}
			c.mode.showStatus(fmt.Sprintf("Agent %s: %s", request.AgentID, string(request.Action)))
			c.mode.requestRender()
		})
	}()
}

// focus shows the focused agent's transcript.
//
// Divergence D-SUBAGENT-3: focusing opens the agent's transcript in an overlay
// instead of re-parenting the whole TUI to the child session, so Esc always
// returns to the main session and no session replacement happens.
// ompig divergence (D208): focus opens a transcript overlay instead of replacing
// the interactive session with the child's.
func (c *agentHubController) focus(id string) {
	messages, ok := c.controls.SubagentTranscript(id)
	if !ok {
		c.apply(agenthub.Request{Action: agenthub.ActionFocus, AgentID: id}, func(ctx context.Context) error {
			return c.controls.SubagentRevive(ctx, id)
		})
		return
	}
	width, height := c.mode.hubSize()
	transcript := agenthub.NewTranscript(id, messages, width, height)
	c.mode.tuiInst.ShowOverlay(transcript, tui.OverlayOptions{
		Title:          "Agent " + id,
		WidthFraction:  0.9,
		HeightFraction: 0.8,
	})
	c.mode.requestRender()
}

// close dismisses the overlay and clears the interactive-mode handle.
func (c *agentHubController) close() {
	c.closed = true
	if c.removeInput != nil {
		c.removeInput()
	}
	if c.overlay != nil {
		c.overlay.Close()
	}
	if c.mode.agentHub == c {
		c.mode.agentHub = nil
	}
	c.mode.requestRender()
}

// AgentHubOpen reports whether the hub overlay is showing.
func (m *InteractiveMode) AgentHubOpen() bool { return m.agentHub != nil }
