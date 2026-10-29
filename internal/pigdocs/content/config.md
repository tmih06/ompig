# Configuration

Ompig reads configuration from two places: the agent directory, which applies to every project you open, and the `.ompig` directory inside a project. Ompig also loads instruction files such as `AGENTS.md` from the directories around your working directory.

Ompig keeps its state under `~/.ompig` by default. It does not read or write Pi's `~/.pi` directory unless you explicitly select shared directories, so Ompig and Pi can run side by side without sharing settings, credentials or sessions. This separation and its opt-in are divergence D2.

Use `/settings` in an interactive session to change common preferences. When you edit a configuration file by hand, run `/reload` so the running session picks up the change. `/reload` rereads settings, keybindings, extensions, skills, prompt templates, themes and context files.

## Configuration root

Ompig chooses one configuration root at startup:

| Order | Source | Root |
|---|---|---|
| 1 | `OMPIG_HOME` is set and not empty | `$OMPIG_HOME` |
| 2 | `XDG_CONFIG_HOME` is set | `$XDG_CONFIG_HOME/ompig` |
| 3 | default | `~/.ompig` |

Pi reads neither `OMPIG_HOME` nor `XDG_CONFIG_HOME`. Its default agent directory is `~/.pi/agent`.

The root holds these directories:

| Path | Contents |
|---|---|
| `<root>/agent/` | The agent directory. See below. |
| `<root>/ompiglets/` | Your [Ompiglet](ompiglets.md) source files. |
| `<root>/artifacts/ompiglets/` | Ompiglet Binaries that Ompig manages. |
| `<root>/receipts/ompiglets/` | Ompiglet resolution and build records. |
| `<root>/state/<namespace>/` | State that one Ompig capability owns. |
| `<root>/docs/` | The offline documentation that `ompig docs` writes. |

## Agent directory

The agent directory is `<root>/agent`, which is `~/.ompig/agent` by default. Set `OMPIG_CODING_AGENT_DIR` to move only this directory. Pi reads `PI_CODING_AGENT_DIR` for the same purpose. Ompig reads that variable only in shared mode.

This page writes the agent directory as `<agent-dir>`.

| Path | Purpose |
|---|---|
| `<agent-dir>/settings.json` | User [settings](settings.md), including Package and resource declarations. |
| `<agent-dir>/keybindings.json` | Custom [keybindings](keybindings.md). |
| `<agent-dir>/models.json` | Custom endpoints and models. See [custom providers](custom-provider.md). |
| `<agent-dir>/auth.json` | Stored API keys and OAuth tokens. See [providers](providers.md). |
| `<agent-dir>/trust.json` | Saved project trust decisions. See [security](trust.md). |
| `<agent-dir>/AGENTS.md` | Your instructions for every project. See [context files](#context-files). |
| `<agent-dir>/SYSTEM.md` | Replaces Ompig's default system prompt. |
| `<agent-dir>/APPEND_SYSTEM.md` | Adds text to the end of the system prompt. |
| `<agent-dir>/extensions/` | User [extensions](extensions.md). |
| `<agent-dir>/skills/` | User [skills](skills.md). |
| `<agent-dir>/prompts/` | User [prompt templates](prompt-templates.md). |
| `<agent-dir>/themes/` | User [themes](themes.md). |
| `<agent-dir>/sessions/` | Saved [sessions](sessions.md), one directory per project. |
| `<agent-dir>/bin/` | Helper binaries that Ompig downloads for its tools, such as `fd` and `rg`. |
| `<agent-dir>/npm/`, `<agent-dir>/git/` | Packages installed at user scope. See [packages](packages.md). |

Ompig reuses helpers already installed in `<agent-dir>/bin/`. Concurrent requests for the same missing helper share one download within a tools manager, including when installation finishes before a waiting request claims the download. Pi starts separate downloads for overlapping same-tool requests; Ompig's coalescing is an additive robustness improvement (D81). Separate managers and processes do not share this coordination.

Ompig also finds skills in `~/.agents/skills/`. See [skills](skills.md).

## Using Pi's directories

Set one environment variable before starting Ompig:

```bash
OMPIG_USE_PI_DIRS=1 ompig
```

Only the exact value `1` enables sharing. This selects `~/.pi/agent` and `<cwd>/.pi` instead of Ompig's agent and project directories (D2). It does not merge, copy or migrate the two trees. Project trust still applies. The setting lives in the environment because a setting inside the relocated directory cannot select that directory.

In shared mode, `PI_CODING_AGENT_DIR` overrides the agent directory and `PI_CODING_AGENT_SESSION_DIR` overrides session storage. `--session-dir` still wins over the environment, followed by the `sessionDir` setting. Ompig ignores `OMPIG_CODING_AGENT_DIR` and `OMPIG_CODING_AGENT_SESSION_DIR` in this mode. `OMPIG_HOME` and `XDG_CONFIG_HOME` still control Ompig-owned SDK caches, documentation, Ompiglets and other product state, not the shared agent directory.

Sharing includes settings, credentials, custom models, keybindings, trust decisions, sessions and discovered resources. Ompig and Pi 1.4.0 read the same current JSON and JSONL shapes. Settings writes retain unknown keys. Ompig does not write `models.json`. Use Pi-compatible Package declarations and resources in shared settings; Ompig-only extensions and Ompiglets do not become Pi-compatible by changing their directory.

Ompig coordinates auth, settings, trust and dynamic model-catalog cache writes with Pi's directory-lock protocol. Independent sessions can use the same agent directory. Do not edit one session concurrently from two processes. Stop older Ompig processes before sharing their files. Ompig automatically reclaims empty regular lock files from v0.2.0 after checking that no older writer holds them. If an older Ompig holds a lock, acquisition uses the normal timeout and asks you to stop that process. Ompig does not reclaim nonempty files, symlinks or active lock directories. Keep a backup before selecting an existing configuration tree.

Packages that hard-code Pi paths, such as Powerline and `pi-acp`, need the default `~/.pi/agent` location. For `pi-acp`, pass the opt-in to the adapter so its Ompig child inherits it:

```bash
OMPIG_USE_PI_DIRS=1 PI_ACP_PI_COMMAND="$(command -v ompig)" pi-acp
```

The adapter keeps its own `~/.pi/pi-acp/session-map.json`; Ompig does not copy or rewrite it. Use one adapter process per home to avoid competing map writes. A custom `PI_CODING_AGENT_DIR` does not relocate paths a package hard-codes, including pi-acp's prompt-template directories.

Unset `OMPIG_USE_PI_DIRS` to return to separate Ompig directories. The shared files remain where they are.

## Project directory

A project keeps its configuration in `.ompig` inside the working directory. Ompig reads `.ompig` only from the working directory itself. It does not look for `.ompig` in parent directories.

| Path | Purpose |
|---|---|
| `.ompig/settings.json` | Project settings. Keys here override the same keys in `<agent-dir>/settings.json`. |
| `.ompig/SYSTEM.md` | Replaces the system prompt for this project. |
| `.ompig/APPEND_SYSTEM.md` | Adds project text to the end of the system prompt. |
| `.ompig/extensions/` | Project extensions. |
| `.ompig/skills/` | Project skills. |
| `.ompig/prompts/` | Project prompt templates. |
| `.ompig/themes/` | Project themes. |
| `.ompig/npm/`, `.ompig/git/` | Packages installed with `ompig install --local`. |
| `.ompig/ompiglets/` | Project Ompiglets. |

### Project trust

A project directory can contain code that runs on your machine. Ompig therefore asks whether you trust a project before it reads `.ompig/settings.json`, `.ompig/mcp.json`, `SYSTEM.md`, `APPEND_SYSTEM.md`, extensions, skills, prompt templates or themes from it. If you do not trust the project, Ompig ignores all of them. A Ompiglet that `--ompiglet <name>` finds in `.ompig/ompiglets/` is used only when the project has a saved trust decision, `defaultProjectTrust` is `"always"`, or the run passes `--approve`. Ompig selects a Ompiglet before it can ask, so it uses your own Ompiglet of that name for the run, and the trust prompt, when one appears, names the project's Ompiglet.

One setting is an exception. Ompig reads `sessionDir` from the project settings before it asks, because it must locate the project's sessions first.

Use `--approve` (`-a`) to trust the project for one run, or `--no-approve` (`-na`) to ignore project files for one run. Use `/trust` to save a decision. See [security](trust.md).

### System prompt files

Ompig looks for `SYSTEM.md` and for `APPEND_SYSTEM.md` separately. For each name, a file in a trusted project wins over the file in the agent directory. Ompig uses one file per name and does not combine the two.

The command line overrides both files. `--system-prompt` replaces the discovered `SYSTEM.md`. `--append-system-prompt` replaces the discovered `APPEND_SYSTEM.md`, and you can give it more than once. Each value can be text or a path to a file.

## Context files

Context files hold instructions for the model, such as build commands, code conventions and project rules. Ompig adds their content to the system prompt.

Ompig looks for context files in the agent directory, in the working directory, and in every parent directory up to the file system root. In each directory it takes the first file that exists from this list:

1. `AGENTS.override.md`
2. `AGENTS.md`
3. `AGENTS.MD`
4. `CLAUDE.md`
5. `CLAUDE.MD`

A directory contributes at most one file. `AGENTS.override.md` therefore replaces `AGENTS.md` or `CLAUDE.md` in its own directory only. It does not hide files in the agent directory or in other directories.

Ompig loads the files in this order:

1. the file in the agent directory;
2. the files from the file system root down to the working directory.

The file closest to your working directory comes last, so its instructions appear after the general ones.

In a Git worktree that sits inside its main checkout, Ompig skips the main checkout's context file when the worktree root has its own file with the same name. This prevents the same repository instructions from loading twice.

Context files do not need project trust. Ompig loads them even when you decline to trust the project. Pi behaves the same way.

The interactive startup banner lists the loaded files on its `[Context]` line. Use `--no-context-files` (`-nc`) to load none of them for one run.

After an in-process switch to a session from another project, Ompig keeps the context files of the project it started in. Pi rebuilds them for the destination project. This difference is D61.

## Startup migrations

On startup, Ompig runs the one-time migrations that Pi runs:

1. When `<agent-dir>/auth.json` does not exist, Ompig moves credentials from a legacy `<agent-dir>/oauth.json` and from `apiKeys` in `<agent-dir>/settings.json` into `auth.json`.
2. Ompig moves session files from the top of the agent directory into `<agent-dir>/sessions/`.
3. Ompig moves the `fd` and `rg` binaries from `<agent-dir>/tools/` into `<agent-dir>/bin/`.
4. Ompig renames a `commands/` directory to `prompts/` and warns about deprecated extension directories.

Ompig applies these migrations only to the selected directories. It does not import files from the unselected tree.

## Diagnostics

`ompig diagnose` prints the Ompig and pinned Pi versions, the build, the Go runtime, the platform, the binary path, the staged extension SDKs, the resolved configuration paths, the providers with stored credentials, and the built-in tools.

Set `OMPIG_STARTUP_TRACE=1` to print the time of each startup step. See [environment variables](environment-variables.md) for the other diagnostic variables.

## Extension runtime variables

Ompig sets these variables for the extension processes it starts. Do not set them yourself.

| Variable | Effect |
|---|---|
| `OMPIG_EXT_SOCKET` | Socket the extension connects to. |
| `OMPIG_EXT_SOCKET_<NAME>` | Socket for one extension inside a packed runtime cell. |
| `OMPIG_EXT_PACKED_CELL` | Key of the packed runtime cell. |
| `OMPIG_EXT_PACKED_CELL_HASH` | Cache key of the packed runtime cell. |

`OMPIG_PARITY_HARNESS=1` turns on the hidden `/probe-*` slash commands that the parity harness uses.

## Related

- [Settings](settings.md) lists every settings key.
- [Environment variables](environment-variables.md) lists the variables Ompig reads.
- [Project trust](trust.md) explains trust decisions.
