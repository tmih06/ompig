# Commands

Ompig commands include top-level CLI verbs, slash commands inside the TUI, and
`ompig docs` commands for the embedded reference bundle.

## CLI verbs

```
ompig [options] [prompt]
```

| Verb | Purpose |
|---|---|
| `ompig` | Start interactive TUI in current directory. |
| `ompig --print <prompt>`, `ompig -p <prompt>` | One-shot: send prompt, print response, exit. |
| `ompig --mode json <prompt>` | One-shot: write JSONL events to stdout, then exit. |
| `ompig --model <provider/model>` | Override model for this run. |
| `ompig -e <path>` (repeatable) | Load extension at path. |
| `ompig --skill <path>` | Load a Skill file or directory. Repeat the option for several Skills. |
| `ompig --no-extensions`, `ompig -ne` | Skip discovered, configured, and built-in extensions; useful for isolating bugs. Explicit `-e` paths still load, so `ompig -ne -e builtin:mcp` keeps only the built-in MCP support. |
| `ompig --models <patterns>` | Scope Ctrl+P model cycling to comma-separated model patterns. It is not an allow list. Without `--model`, a new session starts on the saved default model when the scope includes it, and otherwise on the first matching model. |
| `ompig --session-id <id>` | Use an exact session ID; may be combined with `--no-session` for provider cache affinity without disk persistence. |
| `ompig --mode rpc` | Start the JSONL RPC command loop on stdin/stdout. |
| `ompig --version` | Print the composite version `<Ompig version>+<pinned Pi version>`. Pi prints only its own version (D63). |
| `ompig version` | Detailed ompig/pi/Go/platform/build banner. |
| `ompig diagnose` | Print resolved config + binary identity. |

### Generic subcommands

Product distributions may contribute additional top-level or nested command paths. Core command paths always win; use `ompig --help` and the product's focused docs for contributed commands.

| Subcommand | Purpose |
|---|---|
| `ompig login [provider]` | Authenticate a built-in or contributed target. Built-ins use `auth.json`; contributed providers may own their credential store. |
| `ompig logout [provider]` | Remove credentials through the same provider/store used by login and TUI `/logout`. |
| `ompig auth check --provider <provider> [--model <model>] [--json] [--credentials] [--no-refresh]` | Print `ready`, `not_ready`, or `invalid` and exit 0, 1, or 2. `--json` writes the structured result; `--credentials` emits the resolved credential when ready. Expired OAuth credentials are refreshed unless `--no-refresh` is given, which also leaves `auth.json` and its directory untouched. |
| `ompig auth print-api-key --provider <provider> [--model <model>]` | Print the resolved API key for an external client. Refuses a provider configured with OAuth. |
| `ompig auth print-bearer-token --provider <provider> [--model <model>] [--min-expiry <duration>]` | Print an OAuth bearer token, refreshing it when less than `--min-expiry` (default `30m`; units `ms`, `s`, `m`, `h`) remains. Refuses a provider configured with an API key. |
| `ompig install <source> [-l]` | Install a package through a core or contributed resolver. `-l` installs into the project. |
| `ompig remove <source> [-l]` | Remove a package from settings. `ompig uninstall` is an alias. |
| `ompig update [self\|ompig\|<source>] [--self\|--extensions\|--models\|--all] [--extension <source>] [--force]` | Bare `ompig update` updates the ompig binary, as `pi update` does. `<source>` (or `--extension <source>`) updates one package, `--extensions` updates every package, `--models` refreshes the model catalogs, and `--all` updates packages and then ompig. `--force` reinstalls ompig even when it is the latest version. |
| `ompig install <path> --validate-only [--json]` | Validate without installing; emits structured diagnostics. |
| `ompig install --validate-only --set "<p1>,<p2>"` | Validate a ompiglet-style extension set. |
| `ompig extension init <path> [--name <matching-name>] [--lang go\|python\|rust] [--login] [--isolated] [--force] [--json]` | Scaffold an extension that resolves the staged SDK offline (Go default). `--login` scaffolds a Go login factory with standard Ompig art. |
| `ompig extension upgrade [<name\|path>...] [--all] [--dry-run] [--summary]` | Rewrite Go extensions written for an older SDK with the SDK's tested rules: back up, show a diff, rebuild, report per extension (D109). See [Upgrade a Go extension](extensions.md). |
| `ompig extension preview-login <path>` | Start exactly one extension and render the login set during `session_start`; no model session starts. |
| `ompig extensions cache stats [--json]` | Inspect extension and runtime-cell cache classifications without changing use metadata. |
| `ompig extensions cache prune [--retention <duration>] [--max-size <bytes>] [--failures] [--dry-run] [--json]` | Remove eligible inactive cache entries. Hard roots always remain. `--failures` also removes recorded build failures, so the next start compiles those extensions again. |
| `ompig reload` | Stage the embedded extension SDKs and drop extension builds an older SDK produced. |
| `ompig list` | List installed Packages with Pi-compatible output. |
| `ompig package list [--json]` | Inspect configured Package state without starting runtimes. |
| `ompig package validate <dir> [--json]` | Validate ordinary Package source and Resource membership without installing. |
| `ompig package publish [<dir>] --to npm [--yes] [--tag <dist-tag>] [--access public\|restricted] [--otp <code>]` | Optional sugar over `npm publish` for a Package: validate it, check the catalog fields and that `name@version` is free, show npm's dry run, and publish only with `--yes`. Ompig never handles an npm token. |
| `ompig status [--json]` | Side-effect-free Package/Resource/Ompiglet health and canonical path overview; invalid state exits non-zero. |
| `ompig login --list [--json]` | List generic built-in and contributed authentication targets without reading credentials. |
| `ompig ompiglet list\|show\|validate\|schema\|add\|pull\|update\|publish\|remove\|prune\|build\|keygen\|verify\|tree\|lineage\|trust` | Current Ompiglet source, signed release distribution, inspection, build, and Binary-signing surface (D18). Owned verbs use full words. |
| `ompig ompiglet prune [--keep <n>] [--max-size <size>] [--dry-run]` | Remove old built Ompiglet Binaries, keeping the newest `n` of each Ompiglet and target (default 2). Pulled installs always remain. |
| `ompig ompiglet build <name> --format script --out <path\|->` | Write an explicit source-bound entry script: a POSIX shell script, or a cmd.exe batch file on Windows (D69). Creates no Ompig state or records. |
| `ompig ompiglet build <name> --format binary --out <path> [--sign-key <private-key>]` | Build a Ompiglet Binary and managed v1 resolution/Binary records. The optional Ed25519 signature is checked before command dispatch. |
| `ompig ompiglet keygen <private-key>` | Create an Ed25519 private key and `<private-key>.pub` without replacing existing files. |
| `ompig ompiglet publish <name\|path> --to github --repo <owner/repo> --sign-key <key> [--targets os/arch,...] [--artifacts <dir>] [--tag-prefix <name>/] [--dry-run\|--yes]` | Publish signed per-target Ompiglet Binaries, `SHA256SUMS`, and a signed release index as one GitHub Release. Dry run until `--yes`. See `ompig docs show ompiglets`. |
| `ompig ompiglet publish <name\|path> --to npm [--yes] [--tag <dist-tag>] [--access public\|restricted] [--otp <code>] [--npm-name <name>] [--package-map <alias>=<ref>] [--binaries github:<owner/repo>\|--no-binaries]` | Publish Ompiglet source to npm so `ompig ompiglet add npm:<name>` installs it. Local Packages become `npm:` references. Dry run until `--yes`; refuses an existing `name@version`; Ompig never handles an npm token. See `ompig docs show publishing`. |
| `ompig ompiglet verify <binary> [--source <name\|path>] [--workspace <dir>]` | Verify a Ompiglet Binary signature offline without running it. An unsigned Binary reports unsigned and exits non-zero. With `--source`, also bake that source as the build does and fail when the Binary records a different source or Ompiglet digest. `--workspace` names the workspace the build used (default: the current directory). |
| `ompig ompiglet tree [name] [--depth <n>] [--json]` | Show the installed Ompiglets arranged by `extends`; `*` marks the selected one and `[installed]` one with a built Binary. |
| `ompig ompiglet lineage <name\|path\|binary> [--source <name\|path>] [--json]` | Show an `extends` chain root first with what each link adds, removes or alters, or a signed Binary's recorded identity. Reads local files only. |
| `ompig ompiglet trust [list\|add\|revoke\|require]` | Manage trusted and revoked signer keys and the required-signature policy. |
| `ompig ompiglet build <name> --format image ...` | Reserved Ompiglet Image shape; currently fails clearly because the Image producer is not implemented. |
| `ompig mcp add\|remove\|list\|login\|logout` | Configure and check MCP servers without starting a session. See [MCP servers](mcp.md#mcp-commands). |
| `ompig config [--local]` | Open the Resource filter TUI. Press Tab to switch global and project scope. |
| `ompig setup [status\|go\|container]` | Show the toolchains that extensions and Ompiglet builds use (default), install a verified Go toolchain under `$OMPIG_HOME/toolchains/go`, or show how to install a container runtime. |
| `ompig verify [--json] [--checksums <file>] [--provenance] [--packages] [path...]` | Verify this binary, downloaded files, Ompiglet files, Packages, and extension directories by SHA-256 digest. Ompiglet Binary signatures are checked offline against the local trust policy. |
| `ompig docs [sync\|path\|list\|show <name>]` | Materialize and read the documentation bundled with Stock Ompig. |

### Options

These options apply to `ompig [options] [prompt]`. Run `ompig --help` for the full list.

| Option | Purpose |
|---|---|
| `--provider <name>`, `--api-key <key>` | Select the provider to search for `--model` and pass its API key for this run. `--provider` requires `--model`. |
| `--thinking <level>` | Set the thinking level: `off`, `minimal`, `low`, `medium`, `high`, `xhigh`, or `max`. |
| `--system-prompt <text>` | Replace the system prompt. |
| `--append-system-prompt <text>` | Append text or a file's contents to the system prompt. Repeat it to append more. |
| `--continue`, `-c` | Continue the previous session. |
| `--resume`, `-r` | Select a session to resume. |
| `--session <path\|id>` | Use a session file or a partial session UUID. |
| `--fork <path\|id>` | Fork a session file or partial UUID into a new session. |
| `--session-dir <dir>` | Directory for session storage and lookup. |
| `--no-session` | Do not save the session. |
| `--name`, `-n <name>` | Set the session display name. |
| `--no-mcp` | Turn off the built-in MCP support for this run: no servers connect, and there are no MCP tools or `/mcp`. It does not affect an extension that replaces the built-in MCP support. |
| `--no-stdin` | Ignore piped stdin. Use it when the process that starts Ompig leaves the stdin pipe open and never closes it. Without it, Ompig waits for end of input, as Pi does. |
| `--no-skills`, `-ns` | Turn off skill discovery and loading. |
| `--prompt-template <path>` | Load a prompt template file or directory. Repeat it for more. |
| `--no-prompt-templates`, `-np` | Turn off prompt template discovery and loading. |
| `--theme <path>` | Load a theme file or directory. Repeat it for more. |
| `--use-theme <name>` | Set the initial interactive theme for this run. |
| `--no-themes` | Turn off theme discovery and loading. |
| `--no-context-files`, `-nc` | Do not load `AGENTS.md` or `CLAUDE.md` files. |
| `--export <file> [output]` | Export a session file to HTML and exit. |
| `--list-models [search]` | List available models, with optional fuzzy search. |
| `--tui-mode <mode>` | Set the TUI mode for this run: `fullscreen` (the default) or `regular`. |
| `--verbose` | Force verbose startup, overriding `quietStartup`. |
| `--approve`, `-a` | Trust project-local files for this run. |
| `--no-approve`, `-na` | Ignore project-local files for this run. |
| `--offline` | Turn off startup network operations, as `PI_OFFLINE=1` does. |
| `--ompiglet <name\|path>` | Run a named Ompiglet or the Ompiglet file at a path. A Ompiglet Binary refuses it unless its Ompiglet set `build.allowOverlays`, which lays the selected Ompiglet over the Binary's own. See [Ompiglets](ompiglets.md). |

Extensions can register their own options, such as `--plan` from a plan-mode extension.

### Tools

```bash
ompig --tools read,grep,find,ls -p "Review the code in src/"
```

The tool options select the tools the model can call for one run. See [Settings](settings.md#tools) to change the default selection.

| Option | Effect |
|---|---|
| `--tools`, `-t <list>` | Replace the default selection with a comma-separated allowlist. It applies to built-in, extension, and custom tools. Entries are tool names or patterns where `*` matches any characters. MCP tools are kept unless an entry starts with `mcp__` (see [MCP tools](#mcp-tools)). A list of only `+name` and `-name` entries is not an allowlist: it adds tools to or removes them from the default selection, for example `ompig --tools +codemode,-write`. These entries take exact tool names, not patterns, and cannot be mixed with plain names. A tool removed with `-name` stays removed after `/reload`. |
| `--exclude-tools`, `-xt <list>` | Turn off the named tools or patterns after all other selection options, MCP tools included. It applies to built-in, extension, and custom tools. An excluded tool cannot be called. |
| `--no-builtin-tools`, `-nbt` | Turn off the default built-in tools and keep extension and custom tools. |
| `--no-tools`, `-nt` | Start with every built-in, extension, custom, and MCP tool turned off. |

Without these options, Ompig enables `read`, `bash`, `edit`, and `write`, unless the `defaultTools` setting changes the set. Extension tools stay enabled.

| Built-in tool | Purpose | On by default |
|---|---|---|
| `read` | Read text files and supported images | yes |
| `bash` | Run shell commands | yes |
| `powershell` | Run PowerShell commands on Windows | no |
| `edit` | Replace exact text in an existing file | yes |
| `write` | Create or overwrite a file | yes |
| `grep` | Search file contents | no |
| `find` | Find paths by glob pattern | no |
| `ls` | List directory contents | no |

<a id="mcp-tools"></a>

`--tools` selects the tools declared to the model. It does not remove MCP tools, whose reach is set by their [exposure](mcp.md#control-tool-exposure): `ompig --tools read,codemode` keeps every MCP tool callable from codemode scripts. An MCP tool that no entry names or matches is never declared directly, whatever its exposure; only `tool_search`, if listed, can load it. Once an entry starts with `mcp__`, `--tools` filters MCP tools too, so this keeps only the tools of the `radius` server:

```sh
ompig --tools read,bash,codemode,'mcp__radius__*'
```

The MCP resource tools (`list_mcp_resources`, `list_mcp_resource_templates`, `read_mcp_resource`) count as MCP tools. To remove MCP tools, use `--exclude-tools 'mcp__*'` or `--no-mcp`.

For a read-only session, allow only the tools that cannot change files:

```bash
ompig --tools read,grep,find,ls
```

To keep the default set without one tool, exclude it:

```bash
ompig --exclude-tools bash
```

### `ompig docs`

Stock Ompig materializes its reference docs under `~/.ompig/docs/` so the coding
agent can read the API implemented by the running binary.

```
ompig docs              # sync + summary (default)
ompig docs sync         # force re-sync
ompig docs path         # print this build's docs directory
ompig docs list         # list available files
ompig docs show <name>  # print one doc to stdout
```

Each build writes its bundle to its own directory, `~/.ompig/docs/bundle-<hash>/`,
named by the content hash of its pages. Two Ompig builds that share a config root
(two installed versions, or stock Ompig beside a Ompiglet Binary) each keep their
own bundle, so neither rewrites the other's docs and a running session's docs
never change. A sync keeps the four most recently used bundles and removes
older ones. Pages an older Ompig wrote directly in `~/.ompig/docs/` stay for that
version. `EnsureSynced` runs at startup; a warm start reads one marker file.

When the config root cannot be written and holds no bundle of the running
build, startup writes nothing and prints nothing. The system prompt then has no
docs section, and no other text names a docs page, because Pi's docs always
exist and Ompig does not name files that are not there (D22). `ompig docs show`
still prints the embedded pages. To give a read-only image its docs, run
`ompig docs sync` while the config root is still writable.

## Slash commands (TUI)

The dispatcher is exhaustive - every command here is a parity-mirrored upstream command except where noted as `[ompig]`. Extensions may register additional slash commands via `register.commands[]`.

### Core

| Command | Description |
|---|---|
| `/settings` | Open the settings menu. |
| `/model` | Open the model selector. |
| `/thinking [level]` | Set the thinking level, or open the selector without a level. |
| `/scoped-models` | Enable/disable models for Ctrl+P cycling. |
| `/login` | Configure provider authentication. |
| `/logout` | Remove provider authentication. |
| `/new` | Start a new session in the same cwd. |
| `/resume` | Resume a different session. |
| `/fork` | Fork from a previous user message. |
| `/clone` | Duplicate session at the current position. |
| `/tree` | Navigate session tree. |
| `/compact` | Manually compact the context. |
| `/reload` | Reload keybindings, extensions, skills, prompts, themes, and context files. |
| `/reload --explain` | Same plus a placement/cell report. [ompig] |
| `/export [path]` | Export session (default HTML; specify `.jsonl`). |
| `/import <path>` | Import and resume a JSONL session. |
| `/share` | Upload an unlisted Session share that expires after 30 days. |
| `/bug [description]` | Write a bug report archive to the current directory and print a prefilled Ompig issue link to attach it to. Nothing is uploaded (D62). |
| `/copy` | Copy the last agent message to clipboard. |
| `/name <text>` | Set the session display name. |
| `/session` | Show session info and stats. |
| `/changelog` | Show changelog entries. |
| `/hotkeys` | List keyboard shortcuts. |
| `/quit` | Exit ompig. |
| `/trust` | Set the trust decision for the current project. |
| `/sprite [list\|set <id>\|preview [id]\|create]` | Choose the ompig the startup header shows, preview a sprite's full art, or design your own with the model (`create`; the picker's last item, "Create your own...", shows how). Registered by the built-in `ompig-login` extension, so it is absent with `--no-extensions` or `-builtin:ompig-login`. [ompig] |
| `/llama` | Manage a local llama.cpp server. |

There is no `/exit` or `/clear` command. Pi has neither, and Ompig matches Pi. To
leave, use `/quit`. To clear the editor, press the `app.clear` key (`ctrl+c` by
default). `/hotkeys` lists the current bindings.

### Ompiglet composition

| Command | Description |
|---|---|
| `/ompiglet` | Inspect the Ompiglet active in this process. |

Ompig Standard and other Ompiglets can add commands through selected extension
Resources. Stock Ompig does not register `/runner`, `/ompig-runner`, or other
product commands.

### Parity harness (only when `OMPIG_PARITY_HARNESS=1`)

`/probe-*` family - internal scenarios used by `test/parity/scenarios/`. Not user-facing.

## Keyboard shortcuts

| Key | Action |
|---|---|
| `Enter` | Submit message. |
| `Shift+Enter` / `Ctrl+J` | Newline in editor. |
| `Esc` | Cancel the active dialog, compaction, Bash command, or model turn. |
| `Ctrl+C` | Clear the editor; press it again within 500 ms to exit. |
| `Ctrl+D` | Exit on empty editor. |
| `Ctrl+P` / `Shift+Ctrl+P` | Cycle to next/previous scoped model (provider-qualified). |
| `Shift+Tab` | Cycle thinking level on reasoning-capable models. |
| `Tab` | Autocomplete (slash commands, paths, mentions). |
| `Ctrl+L` | Open the model selector. |
| `Ctrl+O` | Expand or collapse tool output. |
| `Ctrl+T` | Show or hide thinking blocks. |
| `Ctrl+G` | Open the current editor buffer in `externalEditor`, `$VISUAL`, `$EDITOR`, Notepad on Windows, or `nano` elsewhere. |

Extensions may install additional shortcuts via `register.shortcuts[]`. The dispatcher rejects duplicates across the host and all loaded extensions.

## Core-vs-extension precedence

Core commands always win. If an extension registers a slash command whose name collides with a built-in, the built-in handler runs and the extension command is suppressed with a diagnostic. The same rule applies to CLI command paths: core paths are dispatched before contributed top-level or nested paths.
