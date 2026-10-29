# Ompig docs

This is Ompig's user manual and the canonical local reference for coding agents
helping users manage Ompig. It is embedded in the binary and materialized under
`~/.ompig/docs/bundle-<hash>/` by `ompig docs`, one directory per build, named by
the content hash of its pages. Ompig also checks the bundle marker at startup.
Two Ompig builds that share a config root each keep their own bundle, so neither
rewrites the other's. `ompig docs path` prints this build's directory. Do not edit
the materialized copy; edit the bundled source and sync again.

Ompig is the Go-native port of upstream Pi. Numbered differences are summarized in
[divergences](divergences.md).

## Start by task

Read the one row that matches the task, then follow only its direct links.

| I want to... | Start here | Then read |
|---|---|---|
| start from zero, guided | [Onboarding](onboarding.md) | [Install](install.md) |
| do everyday work for a user who does not program (PDFs, spreadsheets, documents, reports, files) | [Applied use cases](applied-use-cases.md) | [Sessions](sessions.md) |
| install Ompig and run a first session | [Install](install.md) | [Providers](providers.md) |
| understand the small mental model | [Concepts](concepts.md) | [Packages](packages.md) or [Ompiglets](ompiglets.md) |
| install or share capabilities | [Packages](packages.md) | [Extensions](extensions.md) or [Skills](skills.md) |
| publish a Package or Ompiglet to npm | [Publishing](publishing.md) | [Packages](packages.md) or [Ompiglets](ompiglets.md) |
| shape, launch, build, or hand off an agent | [Ompiglets](ompiglets.md) | [Runtime cells](runtime-cells.md) for internals |
| write an extension | [Extensions](extensions.md) | [Extension API](extension-api.md) |
| configure models and credentials | [Providers](providers.md) | [Models](models.md) |
| look up a command or path | [Commands](commands.md) | [Config](config.md) |
| run Ompig in CI or a container | [Environment variables](environment-variables.md) | [Config](config.md) |
| change a setting | [Settings](settings.md) | [Project trust](trust.md) |
| resume, branch, or export work | [Sessions](sessions.md) | [Compaction](compaction.md) |
| change or look up a key | [Keybindings](keybindings.md) | [Terminal setup](terminal-setup.md) |
| understand hooks/events | [Events](events.md) | [Extensions](extensions.md) |
| write reusable prompts | [Prompt templates](prompt-templates.md) | [Skills](skills.md) |
| delegate work to a subagent | [Subagents](subagents.md) | [Agent Hub](subagents.md#agent-hub) |
| connect an MCP server | [MCP servers](mcp.md) | [Codemode](codemode.md) |
| run scripts that call tools | [Codemode](codemode.md) | [MCP servers](mcp.md) |
| route requests to different models | [Virtual models](virtual-models.md) | [Extensions](extensions.md) |
| change colors | [Themes](themes.md) | [Settings](settings.md) |
| build terminal UI for an extension | [TUI components](tui.md) | [Extension API](extension-api.md) |
| embed Ompig in a program | [SDK](sdk.md) | [Extensions](extensions.md) |
| add a provider or model endpoint | [Custom providers](custom-provider.md) | [Models](models.md) |
| understand Stock Ompig and explicit compositions | [Stock Ompig and Ompig Standard](built-in-extensions.md) | [Divergences](divergences.md) |
| map entities, paths, and the command that inspects each | [Knowledge graph](knowledge-graph.md) | [Concepts](concepts.md) |
| run Ompig from a script or another program | [CLI integration](cli-integration.md) | [JSON mode](json.md) or [RPC](rpc.md) |
| use JSON mode or RPC | [JSON mode](json.md), [RPC](rpc.md) | [RPC commands](rpc-commands.md), [RPC extension UI](rpc-extension-ui.md) |
| read the file and message formats | [Session format](session-format.md) | [Message types](message-types.md) |
| run a local llama.cpp model | [llama.cpp](llama-cpp.md) | [Models](models.md) |
| check what Ompig can reach and run | [Security](security.md) | [Project trust](trust.md) |
| see usage and cost | [Usage](usage.md) | [Settings](settings.md) |
| fix an install, trust, toolchain, or display problem | [Troubleshooting](troubleshooting.md) | [Install](install.md) |

The platform pages on the site (tmux, Termux, Windows, containerization, shell aliases) are not in this bundle on purpose. They are setup guides for a particular terminal or host, not behavior of the program, and the site keeps them. `troubleshooting.md` and `terminal-setup.md` link them. This is a documentation choice, not a numbered divergence.

## The model in one table

| Pole | Job | Example |
|---|---|---|
| Resource | add one capability or instruction set | extension, skill, prompt, hook, MCP definition, theme, environment |
| Package | distribute available Resources | install from local/git/npm or a contributed resolver |
| Ompiglet | select/scope Resources and define one named agent | `ompig --ompiglet research` |
| Ompiglet release | pin exact Resource closure and executable component plan plus optional artifacts | source plus Ompiglet Binary/Ompiglet Image facets |
| Ompiglet Binary | target-native Ompig executable for one immutable Ompiglet composition | may use fused, subprocess, and external components |
| Ompiglet Image | OCI materialization of Ompiglet component/environment closure | environment-complete carrier |

The two authored management concepts to remember are **Package** and **Ompiglet**.
Read [Concepts](concepts.md) before operational pages to keep authored entities,
carriers, component realization, materialization, and environments distinct.

## Authoritative summary

Ompig is one Go binary whose config root defaults to `~/.ompig` and can be overridden
with `OMPIG_HOME`. Normal extensions run as subprocesses over the current subprocess wire; compatible
Go factories may also fuse into a Ompiglet Binary. These are per-component
execution choices, not different binary kinds. Ompiglet releases record exact
Resource origins and whether each component is fused, subprocess-hosted, or
external and supplied by the binary, release closure, `agentEnv`, or explicit
mapping.

Extension identity comes from its selected source. Runtime registration declares capabilities.
Packages distribute resources. Ompiglets select and scope them into agents.
Products may contribute authentication targets and install
resolvers, and publishing commands without becoming part of raw Ompig.

## Conventions

- `~` means the user's home directory.
- Shell examples assume bash/zsh and `ompig` on `PATH`.
- "Upstream" means upstream Pi; "Ompig" means this Go binary.
- "Carrier" means Host Ompiglet, Ompiglet Binary, or Ompiglet Image.
- "Realization" means fused, subprocess, or external component execution.
- "Materialization" means binary, release, environment, or external delivery.
- "Cell" is an internal extension-host scheduling/process unit, not a user entity.
- Tables are normative reference; prose explains rationale and boundaries.

## Guidance for coding agents

The system prompt points here instead of carrying this advice in every request.
Read it when a user asks you to configure Ompig or to extend it.

Configuring Ompig: a Package makes Resources available; a Ompiglet selects
Resources, built-in tools, discovery, defaults, and environment; a Ompiglet
Binary is a direct executable build output. Packages never contain or activate
Ompiglets. A Ompiglet without a model uses normal model selection, and a Ompiglet
model is a preference, not a requirement. Ompiglet Binaries prebuild Go and Rust
extension cells, but MCP services, interpreted runtimes, configuration, and
secrets stay external requirements. Recommend `ompig install` for Package
capabilities, `ompig --ompiglet` for source execution, and
`ompig ompiglet build <name> --format binary --out <path>` only for a direct Binary
handoff. Read [Concepts](concepts.md) before advising.

Extending Ompig:

- Read [Extensions](extensions.md) first (factory and standalone forms,
  lifecycle, discovery, validation), then [Extension API](extension-api.md) for
  the host API and the per-language SDK quickstart.
- This bundle is a sufficient reference. Do not look for a Ompig source checkout
  or installed extensions to copy unless the user asks you to inspect Ompig
  internals. Author the extension as a self-contained project in the current
  workspace, then validate it with `ompig install <path> --validate-only --json`.
- Scaffold with `ompig extension init <dir>`. Prefer Go (native, fusible into a
  Ompiglet Binary), then Rust (native subprocess, needs cargo), then Python
  (needs a host runtime). Check toolchains in that order and ask before
  installing one; prefer an existing version manager. See [Install](install.md).
- Pick the simplest dev loop: packaged artifact, subprocess wrapper, or
  linked/channel only when a restart or binary update is acceptable.
- Do not add Wasm, embedded JavaScript, dynamic Go plugins, or in-process
  production runtimes without an approved Ompig spec or divergence.
- Change Ompig core only for upstream parity, provider or session behavior,
  package or runtime infrastructure, or generic extension host APIs.

