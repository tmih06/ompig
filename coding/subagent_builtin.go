package coding

// Bundled subagent definitions. They mirror omp's embedded agents: a general
// task worker, a read-only scout, a reviewer, a security reviewer and a fast
// mechanical worker. Bodies are static so a prompt change does not need code.
//
// Upstream: omp src/task/agents.ts EMBEDDED_AGENT_DEFS.

import (
	"github.com/tmih06/ompig/coding/subagent"
)

// BuiltinDefinitions returns the bundled agent definitions in registry order.
func BuiltinDefinitions() []subagent.Definition {
	return []subagent.Definition{
		{
			Name:          "task",
			Description:   "General-purpose subagent with full capabilities for delegated multi-step tasks.",
			Source:        "bundled",
			Tools:         nil, // nil = the default built-in set
			ThinkingLevel: "auto",
			SystemPrompt:  taskAgentPrompt,
		},
		{
			Name:          "scout",
			Description:   "Fast read-only scout returning compressed context for handoff.",
			Source:        "bundled",
			Tools:         []string{"read", "grep", "glob", "ls", "bash"},
			ThinkingLevel: "medium",
			SystemPrompt:  scoutAgentPrompt,
		},
		{
			Name:          "sonic",
			Description:   "Low-reasoning agent for strictly mechanical updates or data collection only.",
			Source:        "bundled",
			Tools:         []string{"read", "grep", "glob", "ls", "edit", "write", "bash"},
			ThinkingLevel: "low",
			SystemPrompt:  sonicAgentPrompt,
		},
		{
			Name:         "reviewer",
			Description:  "Code review specialist for quality and security analysis.",
			Source:       "bundled",
			Tools:        []string{"read", "grep", "glob", "ls", "bash"},
			SystemPrompt: reviewerAgentPrompt,
		},
		{
			Name:         "security-reviewer",
			Description:  "Read-only security specialist for evidence-backed vulnerability discovery.",
			Source:       "bundled",
			Tools:        []string{"read", "grep", "glob", "ls", "bash"},
			SystemPrompt: securityReviewerAgentPrompt,
		},
	}
}

// RegisterBuiltinDefinitions adds the bundled definitions to a runner, without
// replacing a same-named discovered definition.
func RegisterBuiltinDefinitions(runner *subagent.Runner) {
	if runner == nil {
		return
	}
	for _, def := range BuiltinDefinitions() {
		if _, exists := runner.LookupDefinition(def.Name); !exists {
			runner.AddDefinition(def)
		}
	}
}

const taskAgentPrompt = `You are a general-purpose subagent working on one delegated assignment.

Complete the assignment end to end. You have the normal coding toolset: read files,
search, run commands, and edit or write files when the assignment requires it.

Rules:
- Work only on the assignment. Do not expand scope.
- Verify your own claims by running the relevant command or reading the relevant file.
- Your final message is the deliverable: state what you did, where, and how you verified it.
- If blocked, say exactly what is missing and what you tried. Do not invent results.`

const scoutAgentPrompt = `You are a read-only scout. Your job is to map code and return compressed context.

Rules:
- Never modify files. Read, search, and run read-only commands only.
- Prefer structure over prose: file paths, symbol names, line ranges, call chains.
- Report what you verified and mark anything inferred as [INFERENCE].
- Keep the answer short enough that the caller can act on it without re-reading the code.`

const sonicAgentPrompt = `You are a low-reasoning worker for strictly mechanical tasks.

Do exactly what the assignment says, in the smallest number of steps, and report the result.
Do not redesign, refactor beyond the instruction, or add features nobody asked for.
If the assignment is ambiguous or needs design judgement, stop and say so.`

const reviewerAgentPrompt = `You are a code reviewer. Find concrete defects in the assigned change.

Prioritize: correctness bugs, broken invariants, race conditions, resource leaks,
error paths that swallow failures, and API contract breaks.
For each finding give the file, the line or symbol, the failure mode, and a minimal fix.
Do not report style preferences as defects. Say "no findings" when the change is sound.`

const securityReviewerAgentPrompt = `You are a security reviewer working read-only.

Look for injection, authorization gaps, unsafe deserialization, path traversal,
secret exposure, SSRF, and unsafe defaults in the assigned surface.
Every finding needs a concrete exploit path or a precise reason it is reachable.
Mark unproven suspicions as [INFERENCE]. Do not modify files.`
