# Packages

A package is a versioned Ompig/Pi distribution envelope for resources: extensions,
skills, prompts, themes, hooks, MCP definitions, agent environments, and
compatible cross-tool agents. Install packages to add capabilities to Ompig. Use
an independent [ompiglet](ompiglets.md) to select and scope capabilities into one
named agent.

Return to the [docs router](README.md) or read the short
[concept model](concepts.md).

## What a package may carry

| Resource | Purpose |
|---|---|
| Extension | executable capability: tools, commands, hooks, providers, UI |
| Skill | task instructions |
| Prompt | reusable prompt/template |
| Theme | terminal appearance |
| Hook | event command metadata |
| MCP definition | external tool-server configuration |
| Agent environment | standard Dev Container definition and portable local closure |
| Cross-tool agent | compatible plugin roots may expose agent declarations |

A Package does not author Ompiglet, Ompiglet Binary, or Ompiglet Image manifests.

## Package source layout

Put owned resources directly under the package root:

```text
base-coding/
├── package.json
├── extensions/
├── skills/
├── prompts/
├── themes/
├── hooks/
├── mcp/
└── .devcontainer/
```

Do not add `resources/` between the package root and those directories. The
upstream-compatible `package.json` `pi` block may select explicit members:

```json
{
  "name": "@example/base-coding",
  "version": "1.0.0",
  "pi": {
    "extensions": ["extensions/*"],
    "skills": ["skills/*"],
    "prompts": ["prompts/*.md"],
    "themes": ["themes/*.json"]
  }
}
```

Ompig-specific membership uses the additive `ompig` block without changing Pi's
fields. As in Pi, a package with a `pi` block loads only what it declares: its
hooks, MCP servers, and agent environments come from the `ompig` block, not from
conventional directories.

```json
{
  "ompig": {
    "hooks": ["hooks/*.json"],
    "mcpServers": ["mcp/*.json"],
    "agentEnvironments": [".devcontainer/go/devcontainer.json"]
  }
}
```

## Author and validate

Author Package membership directly in ordinary `package.json`, using the standard
`pi` block and optional Ompig Resource fields above. Validate without installing:

```bash
ompig package validate ./base-coding
ompig package validate ./base-coding --json
```

Validation rejects malformed manifests and JSON Resources, missing members, duplicate non-skill public names, lexical or symlink escape, missing Compose services, and absolute or escaping bind mounts.

At startup, duplicate skill names keep the first valid definition and produce a `[Skill conflicts]` warning with the selected and skipped paths, as in Pi. Resolved project, user and Package skills precede additional `--skill` paths. Use `--no-skills` to load only explicitly requested skills. Project skills, including ancestor `.agents/skills`, require project trust. A skill entry may name a Markdown file or a directory. Manifest globstars such as `prompts/**/*.md` match files at the root of that directory and at any depth below it. A manifest entry with `*` or `?` is expanded as Node's `fs.globSync` expands it in Pi, so brace sets, extglobs such as `*.@(md|txt)` and POSIX classes work, and on Windows and macOS a wildcard segment ignores case. A manifest glob, override or Package filter that Pi cannot compile drops that resource kind and the later ones from a local Package, and stops startup for an npm or git Package, as in Pi.

Direct Package installation preserves upstream behavior: it records the Package
in settings and normal discovery exposes enabled members, defaulting to all
unless Package filters narrow them. Installation does not copy Ompiglets or
Agents, merge MCP configuration, create diagnostics, or generate entry points.
A Ompiglet Package dependency resolves through the same materializer without
mutating global/project settings; only explicit member selections activate.

## Install

```bash
ompig install ./local-package
ompig install git:https://github.com/example/package.git
ompig install 'git:https://github.com/example/monorepo.git@v2#subdirectory=packages%2Freview'
ompig install npm:@example/package
ompig install 'npm:@example/private@1.2.3?registry=https%3A%2F%2Fnpm.example.com%2Fteam'
```

Use `npm:` for registry sources and `git:` for Git shorthand such as `github.com/user/repo` or `git@github.com:user/repo`. Without a prefix, those spellings are local paths, even when the path does not exist. Explicit HTTPS and SSH URLs do not require `git:`.

A Git subdirectory selector is `#subdirectory=<URL-escaped-relative-path>`. Ompig
clones one repository checkout under the canonical Git root, resolves the
selected Package root inside it, rejects missing or symlink-escaping selections,
and keeps the checkout while another configured Package still uses a sibling
subdirectory.

A custom npm registry selector is
`?registry=<URL-escaped-absolute-HTTPS-URL>`. Ompig gives each registry a distinct
managed root and passes the non-secret URL to npm, pnpm, or Bun. Registry URLs
with embedded credentials, query strings, or fragments are rejected. Configure
tokens and custom CAs through that package manager's normal `.npmrc`, environment,
or certificate mechanism; Ompig never accepts or writes a registry token.

For user-scoped packages, version metadata lookups run from managed package storage and do not consult the invoking project's `.npmrc` (security divergence D79). Your user configuration and explicit `npmCommand` arguments still apply. Trusted project-scoped packages keep Pi's project `.npmrc` behavior.

Ompig chooses npm, pnpm, or Bun flags from the `npmCommand` executable name, or the name after its last `--` separator. It removes only a `.cmd` or `.exe` suffix and preserves case. It does not run a version probe to identify the package manager.

Products may add source schemes through Ompig's install-resolver contract. For
example, a product-enabled Ompig may accept a `marketplace:` source. Raw Ompig does
not need to know that product's HTTP or authentication protocol.

## Validate without installing

```bash
ompig install ./extension --validate-only --json
ompig install --validate-only --set ./ext-a,./ext-b --json
```

Validation resolves, builds, starts, and registers the selected extension set
without changing package settings. Use it before publishing or before claiming
that a ompiglet's extensions work together.

## Scope

Installs are user-scoped by default. `--local` records a workspace dependency in
`.ompig/settings.json`:

```bash
ompig install ./workspace-extension --local
```

Ompig materializes managed sources once:

| Scope/source | Root |
|---|---|
| user npm | `$OMPIG_CODING_AGENT_DIR/npm` (default `~/.ompig/agent/npm`) |
| user Git | `$OMPIG_CODING_AGENT_DIR/git` |
| downloaded catalog bundle | `$OMPIG_CODING_AGENT_DIR/catalog` |
| project npm | `<workspace>/.ompig/npm` |
| project Git | `<workspace>/.ompig/git` |
| local path | remains at its authored path |

Plugin and Package records may refer to the same materialized source root; Ompig
does not duplicate it into separate caches.

## Listing a Ompig package

### Publish to npm

Add the keyword `ompig-package` and a description to `package.json`, then run `npm publish`. The ompig.dev catalog finds packages by npm keyword when its site builds each day, labels community listings as unreviewed, and accepts no uploads.

`ompig package publish [<dir>] --to npm` is optional sugar over `npm publish`. It validates the Package, checks the fields the catalog reads and that `name@version` is free on npm, shows npm's dry run, and publishes only with `--yes`. Ompig never edits `package.json` and never handles an npm token. See [Publishing](publishing.md).

## Editors and ACP: ompig-acp

`ompig install npm:@ompig/pigpen-acp` installs the Package, whose `/acp` command prints the editor settings. It does not install the `ompig-acp` executable. An editor that speaks the Agent Client Protocol (Zed, JetBrains) starts `ompig-acp`, and `ompig-acp` starts `ompig --mode rpc` for each session. No prebuilt `ompig-acp` download exists yet, so one command builds it from the source the Package ships. It has no dependencies and needs Go 1.26 or later (`ompig setup go` installs one).

```sh
ompig install npm:@ompig/pigpen-acp
ompig acp install
ompig-acp --version        # ompig-acp 0.1.0 (ACP protocol version 1)
```

`ompig acp install` works offline from the installed Package's files, builds with the local Go toolchain, and writes `ompig-acp` (`ompig-acp.exe` on Windows) next to `ompig`: in the directory of `$OMPIG_BIN`, else `~/.local/bin`. It prints the path, and says so when that directory is not on your `PATH`; use the full path as the editor command then.

Then point the editor at it. In Zed `settings.json`:

```json
{
  "agent_servers": {
    "Ompig": {
      "type": "custom",
      "command": "ompig-acp",
      "args": ["--ompig", "/path/to/ompig"],
      "env": {}
    }
  }
}
```

`--ompig` names the `ompig` executable (or a built Ompiglet Binary) to drive; without it `ompig-acp` uses its own override variable (see `ompig-acp --help`), then `ompig` on `PATH`. If ompig has no model or key yet, the editor shows an authentication error with `ompig-acp --terminal-login`, which starts ompig in the editor's terminal so you can sign in. Installing the Package or building the `acp` Ompiglet does not turn a Ompiglet Binary into an ACP server; the editor always runs `ompig-acp`.

## Manage installed Packages

Ompig retains upstream Pi's top-level Package verbs:

```bash
ompig package list
ompig update
ompig update --self
ompig update <source>
ompig update --extension <source>
ompig update --extensions
ompig update --all
ompig update --force
ompig remove <source>
ompig uninstall <source>
ompig uninstall <source> --local --approve
```

`uninstall` is an alias for `remove`, including `--local` and help. Both commands remove the Package from the selected settings scope. Removing the last Package retains `"packages": []` in that settings file.

Package commands show `Installing`, `Removing`, or `Updating` before work starts. npm and Git write their own output to the terminal. A failed child command prints its diagnostics before the CLI reports `Error: <command> failed with code <n>` and exits with status 1.

npm updates batch eligible Packages by scope. Git updates reconcile the configured ref and reinstall dependencies when the checkout changes or dependencies are missing. Installing another ref of an already configured Package replaces its source without discarding resource filters. A failed new Git installation removes its incomplete checkout.

Bare update, `--self`, and the `self`/`ompig` positional targets update Ompig.
`--extensions` updates installed Packages only; `--all` updates Packages before
Ompig. `--extension` names one Package and `--force` reinstalls the current Ompig
release. Conflicting targets fail before either update path starts.

Ompig's owned management surface uses full-word noun/verb commands such as
`ompig package list`, while retaining upstream-compatible install, remove, and
update behavior. Bare `ompig list` retains upstream-compatible Package listing.
Use `ompig --help` as the authority for the binary you are running.

### Enable or disable one installed Resource

Use Package filters when you no longer want one Resource without removing the whole Package:

```bash
ompig config
ompig config --local
```

Use `ompig config --local` to edit project-scoped overrides. The TUI updates
settings. It does not update the Package's `package.json`. At startup a
declared extension, skill, prompt, or theme that matches nothing is skipped, as
Pi skips it, and the rest of the Package loads; `ompig config` lists it as
missing so you can disable it. `ompig package validate` remains strict for
publication.

Startup ignores malformed `package.json` JSON and discovers conventional directories instead. A `pi` object loads only its valid resource arrays. A wrong-typed field declares no resources and does not discard valid sibling fields. These manifest cases produce no diagnostic, as in Pi. Explicit installation and publication validation remain strict.

## Toolchains and first use

In a conventional `extensions/` directory, each top-level `.ts` or `.js` file is an independent extension unless a `pi.extensions` manifest or `index.ts`/`index.js` selects directory entries. A `main.ts`, `main.js`, or `extension.ts` filename does not hide sibling extensions. A `package.json` without extension entries does not select the directory by itself.

Installing materializes source. It does not necessarily build every extension
immediately. A source extension builds into a content-addressed runtime cell on
first use:

| Extension language | Host requirement on first source build |
|---|---|
| Go | Go toolchain |
| Rust | Cargo/Rust toolchain |
| Python | Python and dependencies |
| Node/TypeScript | Node runtime and dependencies |

The cache makes later matching loads fast, but the cache path is not a package
version or reproducibility identity.

## Published Ompiglet releases

A Ompiglet's source publishes to npm with `ompig ompiglet publish <name> --to npm`, and its signed per-target Binaries publish to GitHub Releases with `--to github`. Neither uses a ompig.dev upload API. See [Publishing](publishing.md). Pull by installed Ompiglet name is not available in this release; see [Ompiglets](ompiglets.md#planned-not-in-this-release-named-pulls-and-image-artifacts).

## Packages and ompiglets

| Package | Ompiglet |
|---|---|
| distributes source/Resources | selects/scopes one independent named agent and may build a Binary artifact; Image is reserved |
| installed and updated as a unit | runs through Host or Binary carrier; Image is planned |
| never owns a Ompiglet | may reference zero or more Packages plus direct origins |
| preserves source provenance | records agent defaults and exact component realization/materialization |

Ompiglets remain independent from Packages. A Ompiglet release publishes signed Binary facets to GitHub Releases with `ompig ompiglet publish --to github` and its source to npm with `--to npm`; Ompiglet Image publication remains reserved until an Image producer exists.

## Authoring safety

Do not copy every resource from a running session into a new package implicitly.
A session may contain third-party resources whose licenses and provenance do not
permit rebundling. Package authoring should add explicit resources and preserve
their source/version information. Ompiglet creation from a session is safer
because it writes references rather than copying source.
