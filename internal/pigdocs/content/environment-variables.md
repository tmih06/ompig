# Environment variables

Ompig reads these variables at startup. Each one has a setting or a default that
covers the normal case, so set a variable only when you need to change Ompig from
outside its configuration: in a container, in CI, or while debugging.

Variables named `PI_*` are read for compatibility with upstream Pi. Directory variables use the spellings selected by `OMPIG_USE_PI_DIRS`; other paired variables accept both spellings.

## Location

| Variable | Effect |
|---|---|
| `OMPIG_HOME` | Configuration root. Default `~/.ompig` |
| `XDG_CONFIG_HOME` | When `OMPIG_HOME` is not set, the configuration root is `$XDG_CONFIG_HOME/ompig` |
| `OMPIG_CODING_AGENT_DIR` | Agent directory alone in default mode. Default `$OMPIG_HOME/agent` |
| `OMPIG_USE_PI_DIRS` | Exact value `1` selects Pi's agent directory and project `.pi` resources (D2). Off by default |
| `PI_CODING_AGENT_DIR` | Agent directory in shared mode only. Default `~/.pi/agent` |

In default mode, `OMPIG_HOME` moves settings, keybindings, sessions, trust decisions and installed packages together. In shared mode it moves only Ompig-owned product state. See [Using Pi's directories](config.md#using-pis-directories).

## Network

| Variable | Effect |
|---|---|
| `OMPIG_OFFLINE`, `PI_OFFLINE` | Do not check for package updates or refresh dynamic model catalogs such as Radius. Accepts `1`, `true` or `yes`; any `PI_OFFLINE` value stops the catalog refresh |
| `OMPIG_UPDATE_URL` | Release manifest used by `ompig update` |
| `OMPIG_UPDATE_TRUST_ROOT` | Public key that signs the release manifest |
| `PI_SKIP_VERSION_CHECK` | Do not check for a newer Ompig release at startup. Any non-empty value disables the check, as in Pi. `ompig update` still checks |
| `PI_OAUTH_CALLBACK_HOST` | Host the OAuth callback listens on |
| `PI_SHARE_GATEWAY_URL` | Artifact upload endpoint used only when you run `/share`. Default `https://ompig.dev/v1/artifacts?visibility=unlisted&title=Ompig+session` |
| `PI_CODEMODE_MAX_WORKERS` | Maximum codemode VMs alive at once across all sessions in the process; default `2`. Each VM's script may use up to 256 MB. Raise it when one process runs several sessions, such as in-process subagents. A value that is not a positive integer is ignored; the variable is read on every script. Further scripts wait for a free VM. See [Codemode](codemode.md) |
| `PI_TELEMETRY` | Override the `enableInstallTelemetry` setting: `1`, `true` or `yes` enables it, anything else disables it. Gates the install/update ping to `https://ompig.dev/api/report-install?version=<version>` (sent only after a fresh install or an update with new changelog entries) and the OpenRouter/NVIDIA/Cloudflare attribution headers. `PI_OFFLINE` also stops the ping regardless of this setting. |
| `OMPIG_INSTALL_TELEMETRY_URL` | Overrides the install/update ping's destination URL. No upstream equivalent; tests use it to point at a local server instead of the network. |

Set `OMPIG_OFFLINE` in CI. Without it, every run checks for updates and fails
slowly when the network is blocked.

Offline mode, from `--offline`, `OMPIG_OFFLINE` or `PI_OFFLINE`, sets
`PI_OFFLINE=1` and `PI_SKIP_VERSION_CHECK=1` in the environment extensions
inherit, as Pi does. Pi extensions that check `PI_OFFLINE` see Ompig's offline
mode.

## Diagnostics

| Variable | Effect |
|---|---|
| `OMPIG_DEBUG` | Write a debug log |
| `OMPIG_DEBUG_KEYS` | Log every key as Ompig decodes it |
| `OMPIG_DEBUG_TOOLS` | Print the number of tools sent with each OpenAI Chat Completions request to stderr. Needs the exact value `1` |
| `OMPIG_RENDER_DEBUG` | Write render diagnostics |
| `PI_TUI_DEBUG_REDRAW` | Log why the screen was fully repainted. Needs the exact value `1` |
| `OMPIG_STARTUP_TRACE` | Time each startup step |
| `OMPIG_PROFILE` | Write Go runtime profiles when Ompig exits. Takes a comma-separated list of `cpu`, `heap`, `allocs`, `block`, `mutex`, `goroutine`, `trace`, `metrics` and `flight`. `metrics` writes `ompig-<pid>-metrics.json`: allocation and GC counts for the whole run, live heap, peak RSS, and the goroutines, open files and child processes left at exit. `flight` keeps the last 10 seconds of the execution trace in memory and writes it as `<agent dir>/crash-output/<pid>.flight.trace` (mode 0600) when the session ends without a user request, beside the crash output file; it writes nothing at a normal exit |
| `OMPIG_PROFILE_DIR` | Directory for `OMPIG_PROFILE` output. Default: the working directory |

Use `OMPIG_DEBUG_KEYS` when a keybinding does not respond: it shows whether the key
reached Ompig at all. See [terminal setup](terminal-setup.md).

Use `PI_TUI_DEBUG_REDRAW=1` when the view jumps while output arrives. The log records each full repaint's cause in `pi-tui-debug.log` in the configured TUI log directory, normally the agent directory. Standalone TUI instances without a configured log directory do not write redraw logs. Pi reads the same variable. Ompig ignores the names `OMPIG_DEBUG_REDRAW` and `PI_DEBUG_REDRAW`.

When a session ends without you asking it to, look in the agent directory (`~/.ompig/agent`, or `$OMPIG_HOME/agent`, or the directory `OMPIG_CODING_AGENT_DIR` names). `exit.log` has one line for each signal (`SIGHUP`, `SIGTERM`, `SIGINT`) and each closed terminal, with the time, the pid and the parent pid. If only the input closes while the terminal stays open, Ompig keeps running, as Pi does. `crashes.json` has the panics and fatal errors, and the next start announces the newest one. A session that the operating system killed leaves no line of its own, so the next start adds one to `exit.log`. `ompig diagnose` prints these paths and the newest `exit.log` lines. The `OMPIG_DEBUG` log is `ompig-debug.log` in the temporary directory (`$TMPDIR` on macOS, a per-user folder).

`OMPIG_DEBUG_TOOLS` and `PI_TUI_DEBUG_REDRAW` need the exact value `1`. The other
diagnostic variables accept any non-empty value.

## Sessions

| Variable | Effect |
|---|---|
| `OMPIG_CODING_AGENT_SESSION_DIR` | Session storage and lookup in default mode. Ignored in shared mode. `--session-dir` overrides it |
| `PI_CODING_AGENT_SESSION_DIR` | Session storage and lookup in shared mode only. `--session-dir` overrides it |

## Experimental features

| Variable | Effect |
|---|---|
| `PI_EXPERIMENTAL` | Turn on experimental features when set to `1`. The footer then shows an `xp` badge |
| `OMPIG_EXPERIMENTAL` | Same as `PI_EXPERIMENTAL`. Accepts `1`, `true` or `yes` |

## Updates and deployments

These variables describe how Ompig was installed, so `ompig update` can give the right
instruction. A product deployment sets them. You do not need them for a normal
install.

| Variable | Effect |
|---|---|
| `OMPIG_INSTALL_TIER` | Declare the installation type when Ompig cannot detect it: `container`, `image`, `immutable-binary` or `ompiglet-binary`. Any other value is an error |
| `OMPIG_IMAGE_REF` | Image reference that `ompig update` names for a container or image install |
| `OMPIG_IMAGE_PULL_CMD` | Pull command that `ompig update` prints for a container or image install |
| `OMPIG_REDEPLOY_INSTRUCTION` | Exact redeploy instruction that `ompig update` prints for a container or image install |
| `OMPIG_UPDATE_ALLOW_LOOPBACK_HTTP` | Allow an `http://localhost` or loopback `OMPIG_UPDATE_URL`. Accepts `1`, `true` or `yes`. For testing a release server |

## Display

| Variable | Effect |
|---|---|
| `PI_HARDWARE_CURSOR` | Show the terminal's own cursor |
| `PI_CLEAR_ON_SHRINK` | Repaint when content shrinks |
| `PI_CLIPBOARD_OSC52` | Set to `1` to also send copied text to the terminal with OSC 52 after the local clipboard write, as Ompig does over SSH or Mosh. Use it when a remote client reaches Ompig without SSH variables, such as a multiplexer attached from another machine or a remote IDE daemon |
| `PI_HYPERLINKS` | Override OSC 8 hyperlink detection with `1`, `0`, or `auto` |
| `PI_IMAGE_PROTOCOL` | Override inline image detection with `kitty`, `iterm2`, `none`, or `auto` |
| `PI_TRUE_COLOR` | Override true-color detection with `1`, `0`, or `auto`. With true color off, themes use the 256-color palette |

Each has a setting that does the same thing. A `terminal.hyperlinks`,
`terminal.images` or `terminal.trueColor` setting wins over its variable. See [settings](settings.md).

## Extensions and ompiglets

| Variable | Effect |
|---|---|
| `OMPIG_OMPIGLET_NAME` | Ompiglet to run |
| `OMPIG_OMPIGLET_PATH` | Directories searched for ompiglets |
| `OMPIG_SDK_GO_ROOT` | Go SDK used when building an extension |
| `OMPIG_SDK_PY_ROOT` | Python SDK used when building an extension |
| `OMPIG_SDK_RS_ROOT` | Rust SDK used when building an extension |
| `OMPIG_CELL_BUILD_TIMEOUT` | Time an extension build may take |
| `OMPIG_TS_RUNTIME` | Runtime that runs TypeScript and JavaScript extensions for this process: `node` or an installed runtime's name. Overrides the `extensionsTypescriptRuntime` setting (D19). |
| `OMPIG_TS_RUNTIMES_DIR` | Runtime slot directory. Default `<agent dir>/runtimes/typescript` |
| `OMPIG_BUILDERS_FILE` | Container builder configuration for `ompig ompiglet build`. Default `$OMPIG_HOME/state/pigletbuild/builders.json` |
| `OMPIG_CONTAINER_ENGINE` | Engine of the built-in `container` builder for `ompig ompiglet build`: `docker` or `podman`. Default: Podman when on `PATH`, else Docker |
| `OMPIG_OMPIGLET_PULL_ALLOW_LOOPBACK_HTTP` | Allow `ompig ompiglet pull`, `ompig ompiglet update` and `ompig ompiglet publish --to npm --binaries` to use an `http://localhost` or loopback release URL. Accepts `1`, `true` or `yes`. For testing a release server |
| `OMPIG_OMPIGLET_GITHUB_URL` | Loopback origin, such as `http://127.0.0.1:8080`, that replaces `github.com` and `api.github.com` for `github:` release refs (`ompig ompiglet pull`, `ompig ompiglet publish --to npm --binaries`) and `ompig ompiglet update` discovery. Requires `OMPIG_OMPIGLET_PULL_ALLOW_LOOPBACK_HTTP`; any other host is refused. For testing a release server |
| `OMPIG_LOGO_GLYPHFREE` | `1` renders the logo without special glyphs in `ompig extension preview-login`, and `0` keeps the glyphs. Unset, Ompig drops the glyphs only in Apple Terminal |

The `OMPIG_SDK_*` variables point a build at an SDK checkout instead of the staged
copy. Use them while developing the SDK itself. See
[extensions](extensions.md).

## Process markers

Ompig sets two markers that every child process inherits, as Pi does:

| Variable | Effect |
|---|---|
| `AI_AGENT=pi` | Generic marker that lets tooling identify the launching agent |
| `PI_CODING_AGENT=true` | Lets a child process detect that it runs inside the coding agent |

## Related

- [Providers](providers.md) lists the credential variables for each provider.
- [Settings](settings.md) covers the same behavior in configuration.
- [Config](config.md) lists the files Ompig reads.
- [Terminal setup](terminal-setup.md) covers display and key problems.
