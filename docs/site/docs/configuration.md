# Configuration

Ompig reads configuration from two places: the agent directory, which applies to every project you open, and the `.ompig` directory inside a project. Ompig also loads instruction files such as `AGENTS.md` from the directories around your working directory. The layout follows Pi 1.4.0 ([Pi's configuration reference](https://github.com/earendil-works/pi/blob/v1.2.0/packages/coding-agent/docs/configuration.md)), with Ompig's directory names.

Ompig keeps its state under `~/.ompig` by default. It does not read or write Pi's `~/.pi` directory unless you explicitly select shared directories, so Ompig and Pi can run side by side without sharing settings, credentials or sessions. This separation and its opt-in are divergence D2.

Use `/settings` in an interactive session to change common preferences. When you edit a configuration file by hand, run `/reload` so the running session picks up the change. `/reload` rereads settings, keybindings, extensions, skills, prompt templates, themes and context files.

`/reload` keeps the editor blocked until extension reload handlers finish. It rebuilds the transcript with the refreshed display settings before `session_start` notifications appear.

## Configuration root

Ompig chooses one configuration root at startup:

| Order | Source | Root |
|---|---|---|
| 1 | `OMPIG_HOME` is set and not empty | `$OMPIG_HOME` |
| 2 | `XDG_CONFIG_HOME` is set and not empty | `$XDG_CONFIG_HOME/ompig` |
| 3 | default | `~/.ompig` |

An empty variable falls through to the next row. A value that starts with `~` or `~/` expands to your home directory. Any other value is used as written, including a relative path. If the root needs your home directory and Ompig cannot find it, Ompig reports an error and exits instead of using a relative path. The host and the extension SDKs for Go, Python, Rust and Node.js (`ctx.ConfigHome()`, `ctx.config_home`, `ctx.config_home()`) use this one rule.

Pi reads neither `OMPIG_HOME` nor `XDG_CONFIG_HOME`. Its default agent directory is `~/.pi/agent`.

The root holds these directories:

| Path | Contents |
|---|---|
| `<root>/agent/` | The agent directory. See below. |
| `<root>/ompiglets/` | Your [Ompiglet](/docs/latest/ompiglets) source files. |
| `<root>/artifacts/ompiglets/` | Ompiglet Binaries that Ompig manages. |
| `<root>/receipts/ompiglets/` | Ompiglet resolution and build records. |
| `<root>/state/<namespace>/` | State that one Ompig capability owns. |
| `<root>/docs/` | The offline documentation that `ompig docs` writes. |

## Agent directory

The agent directory is `<root>/agent`, which is `~/.ompig/agent` by default. Set `OMPIG_CODING_AGENT_DIR` to move only this directory. Pi reads `PI_CODING_AGENT_DIR` for the same purpose. Ompig reads that variable only in shared mode.

This page writes the agent directory as `<agent-dir>`.

| Path | Purpose |
|---|---|
| `<agent-dir>/settings.json` | User [settings](/docs/latest/settings), including Package and resource declarations. |
| `<agent-dir>/keybindings.json` | Custom [keybindings](/docs/latest/keybindings). |
| `<agent-dir>/mcp.json` | [MCP servers](/docs/latest/mcp) available in every project. |
| `<agent-dir>/models.json` | Custom endpoints and models. See [custom providers](/docs/latest/custom-provider). |
| `<agent-dir>/auth.json` | Stored API keys and OAuth tokens. See [providers](/docs/latest/providers). |
| `<agent-dir>/trust.json` | Saved project trust decisions. See [security](/docs/latest/security). |
| `<agent-dir>/AGENTS.md` | Your instructions for every project. See [context files](#context-files). |
| `<agent-dir>/SYSTEM.md` | Replaces Ompig's default system prompt. |
| `<agent-dir>/APPEND_SYSTEM.md` | Adds text to the end of the system prompt. |
| `<agent-dir>/extensions/` | User [extensions](/docs/latest/extensions). |
| `<agent-dir>/skills/` | User [skills](/docs/latest/skills). |
| `<agent-dir>/prompts/` | User [prompt templates](/docs/latest/prompt-templates). |
| `<agent-dir>/themes/` | User [themes](/docs/latest/themes). |
| `<agent-dir>/sessions/` | Saved [sessions](/docs/latest/sessions), one directory per project. |
| `<agent-dir>/bin/` | Helper binaries that Ompig downloads for its tools, such as `fd` and `rg`. |
| `<agent-dir>/npm/`, `<agent-dir>/git/` | Packages installed at user scope. See [packages](/docs/latest/packages). |
| `<agent-dir>/crashes.json` | Crash records. `ompig diagnose` prints its path. |
| `<agent-dir>/exit.log` | One line for each interactive exit that no user action asked for, such as SIGHUP, SIGTERM or a closed terminal (D102). |
| `<agent-dir>/crash-output/` | Output of a goroutine panic or runtime fatal error, turned into a crash record at the next start (D102). |

Ompig creates the agent directory, `sessions/` and each project's session directory with mode 0700 (owner only). Pi 1.4.0 creates them with the process umask (D132, [Pi #10770](https://github.com/earendil-works/pi/issues/10770)).

When the configuration root is read-only, Ompig does not warn that it cannot stage the extension SDKs or write the Ompiglet verification cache (D157). `ompig diagnose` reports a read-only configuration root and what Ompig does not write there.

Ompig reuses helpers already installed in `<agent-dir>/bin/`. Concurrent requests for the same missing helper share one download within a tools manager, including when installation finishes before a waiting request claims the download. Pi starts separate downloads for overlapping same-tool requests; Ompig's coalescing is an additive robustness improvement (D81). Separate managers and processes do not share this coordination.

Ompig also finds skills in `~/.agents/skills/`. See [skills](/docs/latest/skills).

## Using Pi's directories

Use `OMPIG_USE_PI_DIRS=1 ompig` to select Pi's agent directory and project `.pi` resources (D2). Only the exact value `1` enables sharing. The environment selects the directory before settings are read, so the opt-in cannot depend on a setting inside that directory.

Shared mode uses `PI_CODING_AGENT_DIR` or `~/.pi/agent` and `PI_CODING_AGENT_SESSION_DIR`, not the corresponding `OMPIG_CODING_AGENT_*` overrides. `--session-dir` still wins. The configuration root (`OMPIG_HOME`, else `$XDG_CONFIG_HOME/ompig`, else `~/.ompig`) continues to hold Ompig-owned SDK caches, documentation, Ompiglet state and sprite selection. Project trust still applies. Ompig does not copy or merge configuration trees.

Ompig and Pi 1.4.0 use compatible settings, auth and session formats. Auth, settings, trust and dynamic model-catalog cache writes use Pi's lock-directory protocol. Settings writes preserve unknown keys, and Ompig does not write `models.json`. Use Pi-compatible Packages and resources in shared settings. Do not edit one session from two processes at once. Stop older Ompig processes before sharing their files. Ompig automatically reclaims stale empty regular lock files from v0.2.0 after checking that no older writer holds them (D73). A file is stale after 10 seconds for synchronous auth, settings and trust operations, or 30 seconds for asynchronous auth and model-cache operations. A fresh file remains in place and uses the normal contention timeout or caller cancellation. If an older Ompig holds a stale lock, acquisition uses the normal timeout and asks you to stop that process. Only an empty regular file is such a legacy lock file. Anything else at a lock path follows Pi's protocol. A fresh entry is contention. A stale lock directory is removed, and so is a link to a stale directory on Windows, whose target stays. A stale nonempty file, a link to a stale file, a dangling link, a nonempty lock directory, and on Unix any stale link stay in place, and acquisition fails.

Powerline and pi-acp can then find Ompig's sessions. Run the adapter with `OMPIG_USE_PI_DIRS=1 PI_ACP_PI_COMMAND="$(command -v ompig)" pi-acp`. Use one adapter per home: it owns `~/.pi/pi-acp/session-map.json`. Keep the default `~/.pi/agent` location for packages that hard-code it; pi-acp does not relocate its template directories with `PI_CODING_AGENT_DIR`.

Unset `OMPIG_USE_PI_DIRS` to return to separate Ompig directories. The shared files remain where they are.

## Project directory

A project keeps its configuration in `.ompig` inside the working directory. Ompig reads `.ompig` only from the working directory itself. It does not look for `.ompig` in parent directories.

| Path | Purpose |
|---|---|
| `.ompig/settings.json` | Project settings. Keys here override the same keys in `<agent-dir>/settings.json`. |
| `.ompig/mcp.json` | Project [MCP servers](/docs/latest/mcp). |
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

Use `--approve` (`-a`) to trust the project for one run, or `--no-approve` (`-na`) to ignore project files for one run. Use `/trust` to save a decision. See [security](/docs/latest/security).

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

In a Git worktree that sits inside its main checkout, Ompig skips the main checkout's context file when the worktree root has its own file with the same name. This prevents the same repository instructions from loading twice. A worktree file that is a symlink to the main checkout's file still loads once. Pi 1.1.0 loaded no project context file in that case; Pi 1.4.0 fixed it ([Pi #10681](https://github.com/earendil-works/pi/issues/10681)).

Context files do not need project trust. Ompig loads them even when you decline to trust the project. Pi behaves the same way.

The interactive startup banner lists the loaded files on its `[Context]` line. Use `--no-context-files` (`-nc`) to load none of them for one run.

After an in-process switch to a session from another project, Ompig keeps the context files of the project it started in. Pi rebuilds them for the destination project. This difference is D61.

## Related

- [Settings](/docs/latest/settings) lists every settings key.
- [Environment variables](/docs/latest/environment-variables) lists the variables Ompig reads.
- [Security](/docs/latest/security) explains project trust.
- [How Ompig works](/docs/latest/how-ompig-works) shows where configuration enters the agent loop.
