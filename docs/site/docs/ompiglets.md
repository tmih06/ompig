# Ompiglets

A Ompiglet defines one named agent application. It selects and scopes Resources, tools, prompts, model preferences, discovery, secrets, and environment requirements.

A Ompiglet is configuration source. It is not a Package and it is not a separate fork of Ompig.

## Run a Ompiglet

Run registered Ompiglet source by name:

```bash
ompig --ompiglet research
```

Run a Ompiglet directly from a file:

```bash
ompig --ompiglet ./agents/research.yaml
```

You can also select source with an environment variable:

```bash
OMPIG_OMPIGLET_PATH=./agents/research.yaml ompig
OMPIG_OMPIGLET_NAME=research ompig
```

A bare `ompig` invocation does not select Ompig Standard or another Ompiglet automatically.

## Ompiglets are agent applications

Treat a Ompiglet as the application boundary for one agent experience. It answers these questions:

- Which capabilities belong to this agent?
- Which tools can the model use?
- Which extensions and skills load?
- Can ambient discovery add more Resources?
- Which system prompt applies?
- Which model does the application prefer?
- Which secrets does it require?
- Which environment must contain the whole process?
- How must its extensions run in a Binary?

This boundary lets a team define several agent applications without creating several Ompig forks.

## Example

```yaml
name: research
description: "Web research agent"

model:
  provider: openai
  name: gpt-5
  thinking: high

systemPrompt:
  file: ./prompts/research.md

packages:
  base: npm:@example/base-coding@1.0.0

extensions:
  - name: web-access
    origins:
      - package:base
      - local:./extensions/web-access
    tools: [web_search, web_fetch]

skills:
  - name: source-review
    origins: [local:./skills/source-review]

tools: [read, write, bash, edit, grep, find, ls]

discovery:
  extensions: []
  skills: [workspace]

build:
  targets: [linux/amd64, darwin/arm64]
  outputName: ompig-research
```

## What a Ompiglet controls

| Facet | Meaning |
|---|---|
| `packages` | Name Package sources that selected Resources can use. |
| `extensions` | Select executable capabilities and limit their model tools. |
| `skills` | Select reusable task instructions. |
| `tools` | Set the built-in model-tool ceiling. |
| `model` | Set a provider, model, context, or thinking preference. |
| `systemPrompt` | Select one inline or file-based system prompt. |
| `discovery` | Permit or reject ambient workspace and user Resources. |
| `secrets` | Declare logical secret requirements without storing values. |
| `agentEnv` | Require an image, Dev Container, or typed environment source. |
| `slots` | Replace a part of Ompig with a fused Go member. `frontend` is the only slot (D91); see [Ompiglet Binaries](ompiglet-binaries.md#frontend-members). |
| `strip` | Strip built-in tools, slash commands, built-in extensions, model APIs, and features; a Ompiglet Binary compiles most of them out (D92). |
| `release` | Set release identity. |
| `build` | Set portable Binary target, output, and realization requirements, and whether the Binary [accepts overlay Ompiglets](ompiglet-binaries.md#accept-overlay-ompiglets) (`allowOverlays`, D202). |

## Model selection

Leave `model` out when any enabled model is acceptable. Ompig then uses its normal explicit, current, and default model selection.

A present model is a preference. An explicit command-line model can override it under Ompig's normal precedence rules.

Do not write `model: any` or `model: default`.

## Tool scope

Omit root `tools` to use Ompig's normal built-in tools. Use an empty list to expose no built-in model tools:

```yaml
tools: []
```

Limit built-in tools with an exact list:

```yaml
tools: [read, grep, find]
```

Omit an extension's `tools` field to expose all tools registered by that extension. Use an empty list to load the extension but expose none of its model tools:

```yaml
extensions:
  - name: review-ui
    origins: [local:./extensions/review-ui]
    tools: []
```

Read-only role with extension tools:

```yaml
name: researcher
tools: [read]
extensions:
  - name: search
    origins: [local:./extensions/search]
    tools: [web_search]
```

Root `tools` restricts built-in tools only, so this role cannot write, edit, or run bash. Extension tools are listed per extension under `extensions[].tools`; here the active tools are exactly `read` and `web_search`.

Command-line and platform policy can narrow these lists. They do not silently widen a Ompiglet.

## Strip built-ins

A Ompiglet can disable Ompig built-ins with `strip` (D92). Pi has no equivalent; a Stock Ompig run without a Ompiglet strip list is unchanged.

`slots` and `strip` are one model. Every part of Ompig a Ompiglet can change is a slot, and a Ompiglet does one of three things with it: it keeps Ompig's own part (the default), replaces it with a fused member under `slots` (today only `frontend`, D91), or strips it under `strip`. Both inherit down the `extends` chain and both come off with `extends.remove`: `remove.slots` restores Ompig's part, and `remove.strip` re-enables a stripped built-in.

```yaml
name: lean-reviewer
tools: [read, grep, find, ls, bash]
strip:
  tools: [bash]
  commands: [/share, /export]
  extensions: [mcp, codemode]
  features: [themes]
```

Each entry is a stable ID. The IDs come from a generated table (`internal/pigstrip/ids_generated.go`) that `make generate` rebuilds from the registrations, so an unknown or renamed ID fails when the Ompiglet resolves, and the error names the ID.

| List | IDs | Effect |
|---|---|---|
| `tools` | built-in tool names: `bash`, `edit`, `find`, `grep`, `ls`, `powershell`, `read`, `write` | The tool joins the `--exclude-tools` denylist. It leaves the tool registry, the active tools, and the system prompt. The effective built-in set is root `tools` minus `strip.tools`. An extension tool with the same name is excluded too. |
| `commands` | built-in slash commands with the leading slash, for example `/share`, and the active Ompiglet's `/ompiglet` | The command leaves the command registry and autocomplete; a stripped `/ompiglet` also leaves RPC `get_commands` and an extension's `getCommands`. Typed text such as `/share` goes to the model like any unknown slash command, so an extension can register the name. |
| `extensions` | built-in extensions: `codemode`, `llama.cpp`, `mcp`, `ompig-login`, `tool-search` | The extension does not load: a settings entry or `-e builtin:<name>` is dropped silently, as Pi's `--no-mcp` drops MCP. A stripped `llama.cpp` starts no llama host. A stripped `mcp` makes no Radius MCP offer after a Radius `/login`. A Binary compiles each one out; `ompig mcp` in a Binary without MCP reports the strip. |
| `apis` | `bedrock-converse-stream`, `google-vertex`, `mistral-conversations` | The API's models are offered nowhere: not by `--list-models`, `/model` and the model selector, model cycling, the `--models` scope, the startup default model, RPC `get_available_models`, or an extension's model registry. `--model` or `--provider` naming one, and building one, fail with the strip message: `Provider amazon-bedrock: API bedrock-converse-stream is stripped from this Ompiglet (strip.apis: bedrock-converse-stream)`. A Binary compiles the implementation out (for Bedrock, the whole AWS SDK). |
| `features` | `themes`, `skills`, `prompt-templates`, `node-extensions`, `extension-sdk-go`, `extension-sdk-rust`, `extension-sdk-python`, `syntax-highlight`, `word-dictionaries`, `mermaid`, `export-html`, `self-update`, `changelog`, `docs`, `ompiglet-builder` | `themes`, `skills`, and `prompt-templates` turn on Pi's `--no-themes`, `--no-skills`, and `--no-prompt-templates` and drop `--theme`, `--skill`, and `--prompt-template` paths; a Ompiglet cannot strip `skills` and also declare `skills` entries. `node-extensions` and the `extension-sdk-*` IDs remove the runtime for TypeScript and JavaScript extensions and the SDKs for Go, Rust and Python source extensions; such an extension then fails to load with a strip message, and a Ompiglet cannot strip a runtime its own extensions need. `syntax-highlight` renders code blocks without token colors, `word-dictionaries` treats a run of CJK, Thai, Lao, Khmer or Burmese text as one word, `mermaid` leaves mermaid fences raw. `export-html`, `self-update`, `docs` and `ompiglet-builder` make HTML export, `ompig update self`, `ompig docs` and `ompig ompiglet build`/`publish` report the strip; `docs` also drops the docs section of the system prompt. `changelog` removes `/changelog` and the startup What's New notice. |

A Ompiglet can't strip every tool and `/quit`. Stripping `/quit` alone is allowed, since Ctrl+D and a double Ctrl+C still exit, and so is stripping every built-in tool alone, which makes a chat-only Ompiglet. A strip list that names every built-in tool and `/quit` together, on its own or through its `extends` chain, fails when the Ompiglet resolves, so `ompig ompiglet validate`, `ompig ompiglet build` and `ompig --ompiglet` refuse it:

```text
strip.tools and strip.commands: a Ompiglet can't strip every tool and /quit (strip.tools: bash, edit, find, grep, ls, powershell, read, write; strip.commands: /quit); keep at least one tool or /quit
```

### Keep mode

A list under `strip.keep` turns that list around: it names the built-ins that stay, and every other ID of that list is stripped, including the IDs a later Ompig adds. Use keep mode for a minimal base that should stay minimal as Ompig and Pi gain built-ins; use the deny lists above for a Ompiglet that should take up new built-ins.

```yaml
name: ompig-core
strip:
  commands: [/share]              # deny mode: strip these
  keep:                           # keep mode: strip everything else in the list
    tools: [read, bash, edit, write]
    extensions: []                # no built-in extension, now or later
```

- One list is in deny mode or in keep mode in one Ompiglet file, never both: `strip.tools` and `strip.keep.tools` together fail.
- `strip.keep` has no `keep` of its own, and keep IDs come from the same table as deny IDs, so an unknown or renamed keep ID fails and the error names it (`strip.keep.extensions[0]: unknown extension ID ...`).
- `keep.<list>: []` keeps nothing from that list. A keep list with no value (`extensions:` alone) fails instead of keeping everything: write `[]`.
- Ompig expands keep mode when the Ompiglet resolves, against the strip table of the Ompig that runs it: the stripped IDs are the list's IDs minus the kept ones. Everything after that (the runtime strip, a Binary's build tags, the record, `ompig ompiglet show`) works on the expanded list, and the floor below holds for it: `keep.tools: []` with `commands: [/quit]` fails.
- `strip.keep.tools` removes the other tools from the registry and the prompt. The root `tools:` allow-list only narrows the active set.

Strip lists merge down the `extends` chain list by list:

| Base | Child | Result | Widening? |
|---|---|---|---|
| deny `D` | `strip.<list>: [x]` | deny `D` and `x` | no |
| keep `K` | `strip.<list>: [x]` | keep `K` without `x` | no |
| keep `K` | `keep.<list>: K2` | keep `K2`; naming an ID outside `K` fails and points to `extends.remove.strip` | no |
| deny `D` | `keep.<list>: K2` | keep `K2`; naming an ID in `D` fails the same way | no |
| keep `K` | `extends.remove.strip.<list>: [x]` | keep `K` and `x` | yes: needs `extends.allowWiden: true` |
| keep `K` | nothing for the list | keep `K`; IDs a later Ompig adds stay stripped down the whole chain | no |

A child can't return a keep-mode list to deny mode, because that would let in every built-in a later Ompig adds. `extends.remove.strip.keep.<list>` fails and names the list; re-enable single IDs with `extends.remove.strip.<list>` instead.

### Inheritance and `ompig ompiglet show`

Deny lists union down the `extends` chain. A child adds entries with its own `strip`. To re-enable an inherited entry, in either mode, name it under `extends.remove.strip` and set `extends.allowWiden: true`; without `allowWiden`, resolution fails with a capability widening error.

```yaml
name: reviewer-with-shell
extends:
  source: local:./lean-reviewer.yaml
  allowWiden: true
  remove:
    strip:
      tools: [bash]
```

`ompig ompiglet show` lists the changed slots in one `Slots` section, each named by its manifest path. A keep-mode list shows its kept IDs above the expanded rows:

```text
Slots:
  frontend  replaced(/work/ompig-tern/frontend)
  extensions  keep([])
  tools.bash  stripped(runtime)
  commands./share  stripped(runtime)
  extensions.codemode  stripped(binary)
  extensions.llama.cpp  stripped(binary)
  extensions.mcp  stripped(binary)
  extensions.ompig-login  stripped(binary)
  extensions.tool-search  stripped(binary)
```

Each stripped ID shows the disposition its Ompiglet Binary gives it: `stripped(binary)` when the Binary does not link the built-in at all, `stripped(runtime)` when it stays compiled in and is disabled at startup. Tools, commands, `themes`, `skills` and `prompt-templates` are runtime; every other ID is binary. When Stock Ompig runs the Ompiglet, it disables every entry at runtime with the same message or fallback, except that the CLI subcommands (`ompig docs`, `ompig ompiglet build`, `ompig update`, `--export`) are only removed in the Binary.

`strip.tools` names exact built-in tool IDs. It takes no `*` patterns: MCP tool names depend on the servers a user configures and cannot be checked when the Ompiglet resolves. To drop MCP tools, strip `extensions: [mcp]`.

### Strip delta report

When Ompig has built a Binary of the Ompiglet before, `ompig ompiglet build` and `ompig ompiglet show` compare the strip table of the running Ompig with the one that Binary was built against (its record keeps it) and print what changed:

```text
Strip table: Ompig 0.4.2 → 0.4.3 adds 3 built-in IDs
  left out (keep mode):     extensions.memory, features.voice
  now included (deny mode): commands./plan
```

"Left out" lists the new IDs a keep-mode list strips; "now included" lists the new IDs a deny-mode list leaves enabled. With no new IDs the report is one line, `Strip table: Ompig 0.4.2 → 0.4.3 adds no built-in IDs`. It prints on every build, before the build runs, and `--json` output carries it as `stripDelta`. `ompig ompiglet show` without `--effective` reads the list modes of a child Ompiglet from the record, since the child's file alone does not state its base's keep mode.

## Discovery

Discovery controls ambient additions only:

```yaml
discovery:
  extensions: [workspace, user]
  skills: [workspace]
```

Explicit entries always load when their origins resolve. An entry whose origins do not resolve is an extension load error: Ompig reports each one and exits with status 1, as it does for any extension that fails to load. Like an explicit `-e` path, a Ompiglet entry still loads with `-ne`, so fix or remove the entry to start. In an active Ompiglet, omitted or empty discovery lists mean no ambient Resources of that type. Bare Stock Ompig keeps its normal discovery behavior.

Use explicit discovery policy to prevent installed or workspace Resources from changing an application unexpectedly.

## Origins

Origins are typed strings. Ompig tries them in declaration order and records the first exact successful source.

```text
package:<alias>
local:<relative-path>
npm:<package>
git:<repository>
http:<source>
https:<source>
<contributed-scheme>:<locator>
```

A local origin resolves from the Ompiglet that declares it. Ompig rejects path escape and unsafe symlink resolution.

A Package dependency does not modify user or project Package settings. The Ompiglet activates only the selected member.

## Required secrets

Declare logical secret names and machine-local sources:

```yaml
secrets:
  - name: github-token
    from:
      env: GITHUB_TOKEN
```

A secret value does not enter portable Ompiglet source, records, sessions, logs, command arguments, or image layers.

Missing or unauthorized values fail before startup.

## Required agent environment

When `agentEnv` is absent, the Ompiglet runs on the compatible current host.

A Ompiglet can require an image:

```yaml
agentEnv:
  image: registry.example/dev@sha256:...
  ompigRuntime:
    mode: image
  policy:
    preset: standard
```

It can instead select a workspace Dev Container or another typed source. The environment constrains the whole agent. It is separate from a code-execution sandbox.

Ompig must enter or verify a required environment before startup. It must not silently fall back to the host.

## Validate and register

Validate source:

```bash
ompig ompiglet validate ./agents/research.yaml
```

Register portable source by name:

```bash
ompig ompiglet add ./agents/research.yaml
ompig ompiglet validate research
ompig --ompiglet research
```

Add a Ompiglet from npm or Git:

```bash
ompig ompiglet add npm:@acme/review@^1.0.0
ompig ompiglet add git:https://github.com/acme/review.git@v2.0.0
```

Ompig fetches the package or repository, finds the Ompiglet file, validates it, and copies it into `~/.ompig/ompiglets/`. It writes a `<name>.origin.json` file next to the Ompiglet that records the source, the resolved version, the npm integrity or Git commit, and the Ompiglet digest. `ompig ompiglet list` shows the origin.

Ompig reads the Ompiglet path from the `ompig.ompiglet` field of the package's `package.json`. Without that field, it reads `ompiglet.yaml` at the package or repository root. The path must stay inside the package.

A remote Ompiglet must be portable. Ompig refuses a Git URL that contains credentials. The command fails before it fetches anything when `OMPIG_OFFLINE` or `PI_OFFLINE` is set.

### Package install scripts

An npm or Git Package that a Ompiglet names can carry lifecycle scripts (`preinstall`, `install`, `postinstall`, `prepare`), and so can the packages it depends on. Ompig installs those Packages with scripts disabled whenever it resolves a Ompiglet: `ompig ompiglet validate`, `ompig ompiglet add`, `ompig ompiglet build` and `ompig --ompiglet` run the package manager with its own `--ignore-scripts` option (npm, pnpm and bun all read it; it is not a `ompig` flag). Validating or adding a Ompiglet therefore does not run a publisher's code. The same applies to the npm or Git package of a Ompiglet source that `ompig ompiglet add npm:<name>` or `git:<repo>` fetches.

A Package whose files exist only after a script ran needs the opt-in:

```bash
ompig ompiglet validate research --allow-install-scripts
ompig ompiglet add npm:@acme/review@1.0.0 --allow-install-scripts
```

`--allow-install-scripts` applies to that one command. It runs the scripts of every npm and Git Package the Ompiglet names and of their dependencies, with your permissions, so review the Ompiglet and its Packages first. Ompig does not remember the choice: a Ompiglet that needs the scripts at every resolution fails to resolve without them, and `ompig ompiglet build` and `ompig --ompiglet` have no opt-in.

This applies to Ompiglet resolution only. `ompig install npm:<name>`, `ompig update` and the other Package commands install as Pi does, and run lifecycle scripts, because Pi passes no `--ignore-scripts`. The opt-in leaves npm's own policy in place: it stops passing `--ignore-scripts`, and nothing more. npm 12 also blocks the scripts of a Package that its `allowScripts` setting does not cover, for Pi and for Ompig alike, so a Package may need approval with `npm install-scripts approve <package>` before the flag has any effect. A Package can still run code when the Ompiglet starts, since a Package is an extension; disabled install scripts do not make a Package safe.

### Add from a monorepo

Select a repository subdirectory with the existing Git source selector and a full lowercase commit SHA (D18). Replace `<full-commit-sha>` with the reviewed commit:

```bash
ompig ompiglet add 'git:https://github.com/acme/agents.git@<full-commit-sha>#subdirectory=ompiglets%2Freviewer'
```

Ompig reads `ompig.ompiglet` or `ompiglet.yaml` inside that selected directory. It never falls back to the repository-root Ompiglet. The checkout must match the pinned commit and contain no modified or untracked files in the selected directory.

The Ompiglets in the Pigpen monorepo (`MichaelKinsy/pigpen`) name their Packages with paths such as `local:../../components/herdr`, outside the Ompiglet's directory. This form rejects them with `error: Package herdr origin "local:../../components/herdr": Ompiglet closure path ../../components/herdr leaves the added subdirectory; bundled local paths must stay inside it`. Pull a Pigpen Ompiglet as a signed Binary instead ([Add a Ompiglet from Pigpen](#add-a-ompiglet-from-pigpen)).

For this pinned subdirectory form, Ompig copies declared relative local Packages, extensions, skills, and prompt files into `~/.ompig/ompiglets/<name>.source/<commit>/`. It rewrites only their paths in the registered YAML. It preserves explicit empty tool scopes and executable permissions. Each local path must exist beneath the Ompiglet file's directory. Absolute paths, parent traversal, symlinks, Git metadata, non-regular files, and closures exceeding 4,096 entries or 32 MiB fail before registration. Keep a Resource's required build files inside its declared local directory. Use explicit YAML fields rather than aliases or merge keys for this form. `extends`, local agent environments, Dev Containers, and file-based secrets remain unsupported for remote registration.

The origin record includes the selected source, commit, original and registered Ompiglet digests, and each copied file's digest. Inventory checks those digests. Removing the source removes its recorded closure without touching sibling Ompiglets. Other remote source forms still reject local Resource origins.

A Ompiglet with relative local Package sources or Resource origins remains source-bound, and `ompiglet add` rejects it. Run that source directly unless all required relative content is registered with it.

### Add a Ompiglet from Pigpen

[Pigpen](https://github.com/MichaelKinsy/pigpen) is a repository of Ompiglets and Packages for Ompig. Each Pigpen Ompiglet has signed Binary releases on GitHub, tagged `<name>/v<version>`. Pull one for your computer:

```bash
ompig ompiglet pull 'github:MichaelKinsy/pigpen/ompig-with-batteries@0.1.0'
```

```text
Pulled ompig-with-batteries 0.1.0 for linux/amd64, signed by ed25519:fc4fbe0a1ba0f640a360bfd5a1c98874
  binary: /home/you/.ompig/artifacts/ompiglets/ompig-with-batteries/0.1.0/linux/amd64/ompig-ompig-with-batteries
  receipt: /home/you/.ompig/receipts/ompiglets/ompig-with-batteries/0.1.0/linux/amd64.pull
```

Run the file on the `binary:` line. It is a complete Ompig with the Ompiglet built in, so it needs no `--ompiglet` option. It carries the Ompig version it was built with, which `--version` prints and which can be older than your `ompig`. `ompig ompiglet list` shows it with the scope `binary`. `ompig --ompiglet ompig-with-batteries` does not run it, because no source is registered. [Ompiglet Binaries](/docs/latest/ompiglet-binaries#pull-a-published-binary) explains the signature checks.

Pigpen's Packages are on npm as `@ompig/pigpen-<name>`, for example `ompig install npm:@ompig/pigpen-dirty-repo-guard` ([Packages](/docs/latest/packages)). A Go extension built for an older extension SDK stops Ompig at startup with `Extension "<name>" was written for an older SDK` and names the changed call, for example `Context.ConfigHome`, which returns an error since Ompig 0.5.0. Remove such a Package with `ompig remove npm:<name>` until its author publishes an update.

## Inspect the active Ompiglet

An active Ompiglet adds a read-only command:

```text
/ompiglet
```

Bare Stock Ompig does not register this command because no Ompiglet is active. A Ompiglet that strips it (`strip.commands: [/ompiglet]`) runs without it, so typed `/ompiglet` goes to the model like any unknown slash command, as in Pi.

Select a different Ompiglet in a separate `ompig` invocation. Ompiglets do not bind existing session history to one composition. New work in a resumed session uses the current invocation's tools, policy, secrets, and environment.

## Build outputs

Create a source-bound script:

```bash
ompig ompiglet build research --format script --out ~/.local/bin/research
```

On Windows the script is a cmd.exe batch file, so give it a `.cmd` name, for example `--out research.cmd` (D69).

Create a native Ompiglet Binary:

```bash
ompig ompiglet build research --format binary --out ./ompig-research
```

A script is a thin launcher. A Ompiglet Binary contains Ompig and a fixed Ompiglet composition. See [Ompiglet Binaries](/docs/latest/ompiglet-binaries).

The native builder compiles Ompig from source with the host Go toolchain. It uses the Ompig checkout that contains the current directory, or `OMPIG_SOURCE_ROOT`. The checkout must be a Git working tree: the builder records its commit, and a plain copy of the source files fails with `resolve Ompig source revision: exit status 128`. A release binary without a checkout fetches the source of exactly its own version with `go mod download github.com/tmih06/ompig@v<version>` and prints `fetching Ompig v<version> source (cached after first build)`. `GOPROXY` and `GOSUMDB` verify the download, and later builds reuse the Go module cache and a staged copy under `~/.ompig/cache/ompig-source/`. The fetch needs no Git and no checkout. A development build has no published source for its version, so it reports `source-unavailable` and asks for a checkout or `OMPIG_SOURCE_ROOT` (D18).

Without a ready native builder, auto selection uses the built-in `container` builder; `--builder container` selects it explicitly. It runs the build in Podman or Docker: `OMPIG_CONTAINER_ENGINE=docker|podman` selects the engine, and otherwise Podman wins when both are on `PATH`. It pulls the digest-pinned public Go image of Ompig's ci-go CI image, installs exactly the running Ompig release in it with `go install`, and runs that release's native builder. The host needs no Go and no Ompig source, and the target can be any `linux/<arch>` the engine runs; another architecture needs the engine's emulation. Go modules and build output are cached under `~/.ompig/cache/container-go/`. The image contains only the Go toolchain, so a Ompiglet with packed or isolated Rust extensions needs a configured builder image that also has Cargo. A development build cannot install itself in the container, and the container builder does not sign (D18).

Publish signed per-target Binaries, `SHA256SUMS`, and a signed release index as one GitHub Release:

```bash
ompig ompiglet publish research --to github --repo acme/research --sign-key ./research-signing.key
```

Publish is a dry run until you add `--yes`. For `ompig-with-batteries` in `MichaelKinsy/pigpen`, pass `--tag-prefix ompig-with-batteries/` and pull a published version with `github:MichaelKinsy/pigpen/ompig-with-batteries@<version>`. The prefix must match the Ompiglet name. `ompig ompiglet update ompig-with-batteries` updates an installed GitHub Binary within its signed namespace, not the repository-wide latest release. See [Publish to GitHub Releases](/docs/latest/ompiglet-binaries#publish-to-github-releases).

## Publish source to npm

Publish a Ompiglet's source to npm with the `ompig-ompiglet` keyword so that anyone can add it with `ompig ompiglet add npm:<name>`:

```bash
ompig ompiglet publish ./agents/reviewer.yaml --to npm
ompig ompiglet publish ./agents/reviewer.yaml --to npm --yes --access public
```

Publish is a dry run until you add `--yes`. Ompig validates the Ompiglet, writes the npm package (`ompiglet.yaml`, `package.json`, README, LICENSE, and the system prompt file), replaces each local Package by `npm:<name>@^<version>`, refuses a `name@version` that npm already has, and runs the `npm` on your `PATH`. npm authenticates you; Ompig never handles an npm token. When this machine published the version's signed Binaries with `--to github`, or `--binaries github:<owner/repo>` names the release, `package.json` records it in `ompig.binaries` so a catalog can show `ompig ompiglet pull` beside `ompig ompiglet add`. See [Publish a Package or Ompiglet](/docs/latest/publishing) for the options, signed Binaries, how ompig.dev finds the package, and trusted publishing in CI. The catalog lists npm packages without review and does not accept uploads.

## Planned (not in this release): named pulls and Image artifacts

```bash
ompig ompiglet pull <name>
ompig ompiglet build <name> --format image --out <reference>
ompig ompiglet build <name> --format binary|image --locked
ompig ompiglet build <name> --format binary|image --record <path>
```

Pull by installed Ompiglet name and the reserved artifact flags above are not available in this release.

## Ompig Standard

Ompig Standard is an explicit Ompiglet in the Ompig source tree:

```bash
ompig --ompiglet ompiglets/standard/ompig-standard.yaml
```

It selects the `piglogin` and `pigrunner` extension Resources through this same contract. `piglogin` owns identity and `/sprite`. `pigrunner` owns `/runner`, `/ompig-runner`, and high-score state. Stock Ompig has no private activation path for Standard.

Ompig Standard also sets:

```yaml
build:
  extensionRealization: fused
```

This setting requires every selected extension to be compiled into its Ompiglet Binary. A non-fusible extension stops the build.

## Worked example: a read-only reviewer

This example builds the smallest useful Ompiglet, runs it, and checks what the model receives. The outputs were captured from Ompig with a test model.

Save this as `agents/reviewer.yaml`:

```yaml
name: reviewer
description: "Reviews a change and lists risks."

systemPrompt:
  text: "You review changes. List the risks first, then the questions."

tools: [read, grep, find, ls]
```

Check it and look at it:

```bash
ompig ompiglet validate ./agents/reviewer.yaml
ompig ompiglet show ./agents/reviewer.yaml
```

```text
OK: reviewer (./agents/reviewer.yaml): 0 extension(s), 0 skill(s)
```

```text
Ompiglet: reviewer
  Reviews a change and lists risks.
Source: ./agents/reviewer.yaml
Built-in tools: read, grep, find, ls

System prompt: You review changes. List the risks first, then the questions. (61 bytes)
```

Run it, in print mode so that the result is one reply:

```bash
ompig --ompiglet ./agents/reviewer.yaml -p "Review the last commit"
```

In the interactive screen, `/ompiglet` prints the active Ompiglet and its source:

```text
 Ompiglet: reviewer
 Source: /home/you/project/agents/reviewer.yaml
```

### What the model receives

Checked in the provider request.

**The system prompt is replaced.** Without a Ompiglet, the system prompt starts `You are an expert coding assistant operating inside ompig, a coding agent harness.` With this Ompiglet it starts with your text, followed by the working directory (`<cwd>/home/you/project</cwd>`). The Ompiglet's text is the prompt: it does not add to Ompig's. Leave `systemPrompt` out to keep Ompig's.

**`tools` is a ceiling, not a list to turn on.** A Ompiglet's `tools` limits which built-in tools can be active. It does not activate any. The active set still starts from the default (`read`, `bash`, `edit`, `write`) or from your `defaultTools` setting or `--tools` option, and the Ompiglet removes everything outside its list. With `tools: [read, grep, find, ls]` the model received only `read`, because `bash`, `edit` and `write` are outside the ceiling and `grep`, `find` and `ls` are not in the default set:

| How Ompig is started | Tools the model received |
|---|---|
| `ompig --ompiglet ./agents/reviewer.yaml` | `read` |
| `ompig --ompiglet ./agents/reviewer.yaml --tools read,grep,ls,bash` | `read`, `grep`, `ls` (`bash` is cut) |
| `defaultTools: ["read","grep","find","ls","bash"]` in settings | `read`, `grep`, `find`, `ls` |
| a Ompiglet with `tools: []` | none |

To give the reviewer `grep`, `find` and `ls`, pass `--tools read,grep,find,ls`, set `defaultTools`, or build the Ompiglet into a script that passes the option:

```bash
ompig ompiglet build reviewer --format script --out ./run-reviewer
```

```text
Ompiglet script written: /home/you/project/run-reviewer
```

The script is two lines. `"$@"` passes along whatever you type after its name:

```sh
#!/bin/sh
exec ompig --ompiglet '/home/you/.ompig/ompiglets/reviewer.yaml' "$@"
```

For this reviewer, call it as `./run-reviewer --tools read,grep,find,ls`. The script file names the registered copy, so keep that copy ([Command line by example](/docs/latest/cli-examples#ompiglets) shows `ompig ompiglet add`).

### Mistakes Ompig reports

The Ompiglet format is closed. An unknown field is an error, with the line:

```text
FAIL: Ompiglet /home/you/project/agents/reviewer.yaml: parse ompiglet YAML: yaml: unmarshal errors:
  line 3: field system not found in type ompiglet.Ompiglet
```

`systemPrompt` takes `text` or `file`. A skill needs an origin: `FAIL: skill "commit": no origin specified`. `ompig ompiglet schema` prints every accepted field. It lists the same values the parser accepts for the built-in tools (including `powershell`), the strip IDs, the discovery scopes, the `agentEnv` runtime modes and policy presets, and `build.extensionRealization`, and a test fails when the two differ.

## Related documentation

- [Ompig concepts](/docs/latest/concepts)
- [Packages](/docs/latest/packages)
- [Extensions](/docs/latest/extensions)
- [Ompiglet Binaries](/docs/latest/ompiglet-binaries)
- [Derivative harnesses](/docs/latest/derivative-harnesses)
