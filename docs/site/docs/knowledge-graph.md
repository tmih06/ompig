# Knowledge graph

Ompig is built from a small set of entities. This page lists each entity, where it lives, the command that inspects it, and how the entities relate. The same graph is published as JSON-LD at [ompig-knowledge-graph.jsonld](/ompig-knowledge-graph.jsonld) and as Mermaid source in `docs/knowledge-graph/ompig.mmd`, and it ships inside every ompig binary as `knowledge-graph.md` in the local docs (`ompig docs show knowledge-graph`).

![Entity relationship diagram of Ompig](/ompig-knowledge-graph.svg)

| Entity | What it is | Where it lives | Inspect with | Read |
|---|---|---|---|---|
| Pi | Upstream TypeScript coding agent; the behavior oracle. | `https://github.com/earendil-works/pi` | `ompig version` | `divergences.md` |
| Ompig | The Go implementation; executable ompig. | `.upstream/pin pins Pi` | `ompig verify` | `README.md` |
| Stock Ompig | Product-neutral ompig binary with no Ompiglet selected. | `cmd/ompig` | `ompig version` | `built-in-extensions.md` |
| Session | Append-only JSONL tree of entries for one conversation. | `~/.ompig/agent/sessions` | `/tree, /session` | `sessions.md` |
| Model Runtime | Session-owned model lookup, auth, completion, and streaming used by every mode. | `coding/` | `/model, ompig --list-models` | `models.md` |
| Provider | Turns a Transcript into one Event Stream for one API. | `ai/` | `ompig --list-models` | `providers.md` |
| Mode | Interactive TUI, print, JSON event stream, or RPC over stdin/stdout. | `cmd/ompig` | `ompig --mode rpc` | `commands.md` |
| Resource | One capability: extension, skill, prompt, hook, MCP definition, theme, or environment. | `~/.ompig/agent, .ompig/ in a project` | `ompig status --json` | `concepts.md` |
| Extension | Code Resource in Go, Rust, Python, or Node/TypeScript that registers tools, commands, events, and UI. | `extensions/sdk*, ~/.ompig/agent/extensions` | `ompig install --validate-only <dir>` | `extensions.md` |
| Extension SDK | One extension contract per language: Go, Rust, Python, TypeScript declarations. | `extensions/sdk, sdk-rs, sdk-py, sdk-ts` | `ompig extension init` | `extension-api.md` |
| Extension realization | How an extension runs: isolated process, packed runtime cell, or fused into a Ompiglet Binary (Go only). | `coding/extension/host` | `ompig status --json` | `runtime-cells.md` |
| Extension Host | Loads, isolates, reloads, and shuts down extension realizations over one wire contract. | `coding/extension/host/subprocess` | `/reload` | `runtime-cells.md` |
| Package | Versioned collection of Resources from npm, git, or a local path; never owns a Ompiglet. | `~/.ompig/agent/npm, ~/.ompig/agent/git` | `ompig list` | `packages.md` |
| Ompiglet | One named agent: selects and scopes Resources and sets defaults. | `*.yaml Ompiglet file` | `ompig ompiglet validate <file>` | `ompiglets.md` |
| Ompig Standard | The optional curated Ompiglet: login identity, Ompig Runner, Angry Pigs. | `ompiglets/standard/ompig-standard.yaml` | `ompig ompiglet validate ompiglets/standard/ompig-standard.yaml` | `built-in-extensions.md` |
| Ompiglet release | Pins the exact Resource closure and component plan of a Ompiglet. | `release.version and managed records` | `ompig ompiglet show --record <path>` | `ompiglets.md` |
| Ompiglet Binary | Target-native ompig executable for one Ompiglet release; verifies its closure at startup. | `ompig ompiglet build --format binary` | `ompig verify` | `ompiglets.md` |
| Builder | Produces Ompiglet Binaries: native (needs Go and Ompig source) or container (needs Docker or Podman). | `coding/ompigletbuild` | `ompig setup` | `ompiglets.md` |
| Ompig Porter | Parity workbench Ompiglet Resource; proposes ports, never accepts its own work. | `ompiglets/porter` | `make porter-check (source checkout)` | `divergences.md` |
| Divergence | A numbered, approved, user-visible difference from pinned Pi. | `docs/parity/DIVERGENCES.md` | `ompig docs show divergences` | `divergences.md` |

| From | Relation | To | Note |
|---|---|---|---|
| Ompig | implements | Pi | exact pinned release |
| Stock Ompig | builds | Ompig | one binary per target |
| Session | owns | Model Runtime | one per Session |
| Model Runtime | streams through | Provider | one Provider per request |
| Mode | drives | Session | TUI, print, JSON, RPC share one Session path |
| Package | distributes | Resource | one to many |
| Extension | is a | Resource |  |
| Extension | written with | Extension SDK | one language per extension |
| Extension | realized as | Extension realization | isolated, packed, or fused |
| Extension Host | hosts | Extension realization | same contract in every realization |
| Ompiglet | selects | Resource | and scopes discovery |
| Ompig Standard | is a | Ompiglet | explicit, never active in Stock |
| Ompiglet release | pins | Ompiglet | exact closure |
| Ompiglet Binary | realizes | Ompiglet release | per target |
| Builder | produces | Ompiglet Binary | no silent fallback between builders |
| Ompig Porter | is a | Resource | selected by the Porter Ompiglet |
| Divergence | documents difference from | Pi | numbered D<N> |

The graph source is `docs/knowledge-graph/ompig.graph.json`. Run `automation/gen/gen-knowledge-graph.py` after editing it.
