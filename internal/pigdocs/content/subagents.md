# Subagents

A subagent is a second agent with its own context, tools and model, working on one
assignment. The main agent spawns it with the `task` tool, reads its result from
`agent://<id>`, and can message it later with `agents` instead of spawning a
replacement. `Alt+A` opens the **Agent Hub**, which shows every subagent of the
session and lets you steer, revive or kill it.

## Spawning from the model

The `task` tool takes a `context` shared by the call and a `tasks` array:

```json
{
  "context": "Review the change on this branch.",
  "tasks": [
    {"name": "Reviewer", "agent": "reviewer",
     "task": "Report correctness findings for the diff.",
     "solutionSpace": "defect cause open; no known bug"}
  ]
}
```

Every item runs in a separate session. A subagent does not see the parent's
conversation, so give it complete instructions; `context` is rendered into each
child's system prompt, and `solutionSpace` describes how open-ended the work is
(which causes or designs remain open), which tunes the child's reasoning effort.

One item in a `tasks` array is equivalent to a single `task` call; a batch is
usually better because the spawns run concurrently.

## Tools

| Tool | Purpose |
| --- | --- |
| `task` | Spawn one subagent, or a batch with `tasks[]`. |
| `wait` | Block until one background subagent settles. Use it when blocked; do not poll. |
| `agents` | `list`, `read`, `message`, `revive`, `kill`, `jobs` for existing subagents. |

`task`, `wait` and `agents` are part of the default tool set. A settings layer
that names its own `defaultTools` replaces that set, so list them explicitly if
you keep the subagent surface while narrowing the built-ins.

## Agents

A subagent runs an *agent definition*: a Markdown file with YAML frontmatter and
the system prompt as its body.

```markdown
---
name: reviewer
description: Report concrete defects in a change.
tools: read, grep, glob, ls, bash
model: "@slow"
thinking-level: high
---

Find correctness bugs, broken invariants, races and swallowed errors. ...
```

| Field | Effect |
| --- | --- |
| `name` | The id the `task` tool's `agent` field selects. `main` and `sub` are reserved. |
| `description` | Shown in the tool description; required. |
| `tools` | Built-in tool names or a CSV list. Omitted means the default set; an empty value means none. `glob` is accepted as Pi's `find`. |
| `spawns` | `*` or a list of agent names this agent may spawn. Omitted inherits the parent's policy; `[]` denies delegation, which also removes `task`, `wait` and `agents`. |
| `model` | Model selectors (`provider/id`, a bare id, or a fuzzy substring) tried in order; an unresolvable selector keeps the parent's model. |
| `thinking-level` | `off`, `minimal`, `low`, `medium`, `high`, `xhigh`, `max` or `auto`. |
| `blocking` | Run inline even when background execution is available. |

Definitions are discovered first-wins by name: project `.ompig/agents/*.md`,
then user `<agent dir>/agents/*.md`, then plugin `agents/` roots, then the
bundled agents. A project file with the same name as a bundled agent replaces it.

Bundled agents: `task` (general-purpose, full tools), `scout` (read-only
exploration), `sonic` (mechanical edits), `reviewer` and `security-reviewer`
(read-only analysis).

## Results and artifacts

Each subagent writes:

| Path | Contents |
| --- | --- |
| `agent://<id>` | Its final output, saved as `<id>.md`. |
| `<id>.json` | Its structured output, when the spawn asked for `outputSchema` and the result satisfied it. |
| `<id>.jsonl` | Its transcript, which the hub and `agents read` open even after it parks. |

Ids are name-based: the first `Reviewer` keeps that id and later ones become
`Reviewer-2`, `Reviewer-3`. A nested spawn is dot-qualified, `Parent.Child`, and
`agent://<parent>.<child>` reads the nested output. A slash after the id is JSON
extraction, so `agent://Worker/reports/0/data` reads one field of the structured
output.

A long output is truncated inline and the full text stays at `agent://<id>`.

`outputSchema` makes a spawn structured: the child's final message is parsed as
JSON (a fenced block with a JSON answer is accepted) and validated against the
schema. A conforming result is written to `<id>.json`; a result that is not JSON
or that the schema rejects is kept as prose, reported as the agent's structured
failure, and leaves no sidecar, so a stale payload never outlives a run.

## Background execution and `wait`

Spawns run in the background by default: `task` returns an acknowledgement that
names each agent, and the result is delivered when the job settles. A blocked
caller calls `wait`; results are delivered exactly once, so a job consumed by
`wait` is not replayed. Do not poll. A definition with `blocking: true` runs
inline instead and its result comes back in the `task` call itself.

Items in one batch run concurrently. The bound is the runtime's semaphore (32 by
default), not the call, so a batch of more than 32 queues the remainder.

The concurrency bound is the runner's semaphore (32 by default); a queued spawn
starts as capacity frees. Nesting is bounded: an agent at the depth limit cannot spawn further, because it
does not receive the `task` tool. The default cap is two levels. A spawn that the
parent's `spawns` list does not allow is rejected before a child session exists.

## Nested delegation

A subagent may spawn its own subagents, up to the depth cap (two levels by
default). Every agent in the tree shares one roster, so the Agent Hub shows the
whole tree and a nested spawn's id is dot-qualified: `Parent.Child`, readable at
`agent://Parent.Child`. An agent at the cap does not receive the `task` tool.

## Agent Hub

`Alt+A` or `/agents` opens the roster. Rows are ordered running, idle, parked, then aborted,
and each shows status, id, agent type, current tool or task, tokens and elapsed
time.

| Key | Action |
| --- | --- |
| `j` / `k`, arrows | Select a row. |
| `enter` | Open the selected agent's transcript. |
| `m` | Type a message; `enter` delivers it to the selected agent. |
| `r` | Revive a parked or killed agent. |
| `x` | Kill the selected agent (aborts its turn, keeps its transcript). |
| `t` | Toggle the parent/child tree view. |
| `1` / `2` | Switch between the roster and the combined activity log. |
| `/` | Filter rows by id or agent type. |
| `esc` | Clear the filter, then close the hub. |

On a terminal at least 100 cells wide the inspector pane sits beside the roster
and adds model, thinking level, request and tool-call counts, cost, context
window use, the current intent, lineage and artifact paths.

Steering a running agent queues the message as steering; steering an idle agent
starts a new turn on the same transcript. A killed agent keeps its transcript, so
reviving it reopens the same conversation instead of starting over. Killing and
reviving never delete output.

Advisor transcripts appear as read-only rows: they can be read but not messaged,
revived or killed.

A resumed session's hub repopulates from the artifact directory: agents that
session already spawned come back as parked rows, so their transcripts and
outputs stay readable and a message revives them.

## Resource use

A subagent is a full session: its own model context, its own tool loop and its
own token spend. An idle agent keeps its live session for seven minutes and is
then parked, which releases the session while keeping the transcript, output and
a revive path; `SetIdleTTL` changes the window and a non-positive value disables
parking. The hub's token and cost columns are per agent, so the roster is
the place to see where a delegation is spending. Prefer messaging an existing
agent over spawning a replacement: the existing one already holds the context,
and a new spawn pays for it again.
