package coding

import (
	icodingagent "github.com/tmih06/ompig/internal/codingagent"
)

// The session must satisfy the Agent Hub's control surface, or Alt+A in the
// interactive mode cannot reach the roster.
var _ icodingagent.AgentHubHandle = (*Session)(nil)
