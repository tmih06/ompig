# Ompig ecosystem ontology

This is the normative maintainer reference for Ompig Resources, Packages, Ompiglets, outputs, environments, and external product bindings. Return to the [maintainer router](README.md) for implementation details.

## The shortest model

> Packages distribute and make Resources available. Ompiglets select and scope them. A source Ompiglet runs with `ompig --ompiglet`; an optional generated script is only a thin source entry point. Ompiglet Binaries execute directly and verify exact records. Ompiglet Image is the reserved OCI carrier. Deployment platforms supply their bindings outside portable Ompiglet source.

## Entities and outputs

| Term | Contract | Authored? |
|---|---|---:|
| Resource | one extension, skill, prompt, theme, hook, MCP definition/client, or agent environment | yes |
| Plugin | cross-harness Agent Plugins package plus distinct vendor facets | yes |
| Package | optional Pi/Ompig distribution envelope for Resources; never decides Ompiglet activation | yes |
| Ompiglet | one named whole-agent composition and policy | yes |
| Ompiglet release | versioned distribution joining source identity, exact closure, records, and optional artifacts | generated publication state |
| source entry script | executable POSIX wrapper around `ompig --ompiglet <canonical-source>`; a cmd.exe launcher on Windows (D69) | generated convenience output |
| Ompiglet Binary | target-native executable for one immutable Ompiglet composition | build artifact |
| Ompiglet Image | reserved OCI materialization of Ompiglet/runtime/environment closure; no producer exists | build artifact |

The script is a user-owned source entry point, outside Ompiglet artifact and managed inventory. Ompig writes it only to an explicit `--out` path or stdout and never removes it later.

## Execution carriers

| Carrier | Invocation | Contract |
|---|---|---|
| Host Ompiglet | `ompig --ompiglet <name|path>` or generated source script | current Ompig process resolves source and enters required environment |
| Ompiglet Binary | execute the Binary path | target-native Ompig verifies its baked Ompiglet and registered closure |
| Ompiglet Image | reserved: run the image digest/reference | reserved exact Ompig, Resource, runtime, and environment closure |

No ProfileInstance exists. Ompiglet execution uses exactly the three carriers in the table above. A local run is a Session. Remote-agent identity belongs to the consuming platform.

## Package responsibility

Upstream-compatible `ompig install|list|remove|update|config` controls Package settings and discovery. Installation may materialize and record a Package. It must not:

- copy or activate a Ompiglet;
- copy Agent declarations into another store;
- merge MCP configuration;
- create aliases, scripts, launchers, or shell state;
- activate Ompig-only Resources outside normal Package discovery/filtering.

Ompig may provide side-effect-free `ompig package list --json` and `ompig package validate <dir>`. Ordinary `package.json` is the authoring format; Ompig does not need a second Package membership editor.

A Ompiglet may materialize a named Package source without mutating Package settings. Only Ompiglet entries that explicitly select Package members activate them.

## Lean Ompiglet contract

The closed root vocabulary is:

```text
name description extends packages extensions skills tools model
systemPrompt discovery agentEnv secrets release build slots strip
```

`name` is the only required field. `ompig ompiglet schema` prints the JSON Schema of this vocabulary. `slots` names members that replace a part of Ompig; `slots.frontend` is the only slot (D91). `strip` disables built-ins and compiles them out of a Ompiglet Binary (D92). A Ompiglet's version is `release.version`; a root `version` field fails to parse.

Removed pre-release fields receive no aliases:

```text
builtin agentPrompts mcpServers mcpConfigPaths extendsPolicy top-level remove
harness settings extension commands extension shortcuts
```

### Tools

Root tools are an exact allowlist of Ompig built-in model tools:

```yaml
tools: [read, grep, find, ls]
```

| Shape | Meaning |
|---|---|
| omitted | Ompig's normal built-in tool set |
| `[]` | no built-in model tools |
| list | exact built-in ceiling |

Extension tools use the same list shape:

```yaml
extensions:
  - name: web
    origins: [package:web]
    tools: [web_search, fetch_content]
```

Effective availability is the intersection of runtime registration, Ompiglet allowlists, explicit CLI narrowing, and platform authorization. CLI/platform inputs never silently widen a Ompiglet ceiling.

Slash commands and shortcuts are not Ompiglet capability policy. A Ompiglet cannot add or scope them. It can only remove built-in slash commands with `strip.commands` (D92).

### Discovery

Discovery governs ambient Resources only:

```yaml
discovery:
  extensions: [workspace, user]
  skills: [workspace]
```

Allowed sources initially are `workspace` and `user`. Empty or omitted lists in an active Ompiglet mean no ambient Resources of that kind. Explicit Ompiglet Resources and explicit caller additions are unaffected and remain subject to the current runtime's policy. Ompiglets do not add composition identity to Session files. Bare Ompig retains upstream ambient behavior.

Harness-format discovery belongs to a selected harness extension. That extension may read `.claude`, `.github`, `.pi`, and `.ompig` according to its own namespaced configuration. Raw Ompiglet schema does not encode one extension's options.

### Packages and origins

Package dependencies are aliases:

```yaml
packages:
  review: npm:@acme/review@^1
```

Resource entries carry one ordered typed-string origin vocabulary:

```yaml
extensions:
  - name: review-tools
    origins:
      - package:review
      - local:./extensions/review-tools
```

The shared source parser owns local/Git/npm/HTTP/catalog/contributed identity, anchoring, ordering, and materialization. Package order never implies activation.

### Derivation

All derivation behavior is one object:

```yaml
extends:
  source: npm:@acme/base@^1
  version: "^1"
  allowWiden: false
  remove:
    skills: [inherited]
```

Depth is eight. Each source resolves its own Ompiglet/workspace paths before merge. Whole fields replace; named collections replace/add/remove by identity; order is deterministic; absent removals and unsafe widening fail. Release/build/ extends metadata does not inherit into the effective Ompiglet.

### MCP

Raw Ompiglet has no MCP endpoint, command, args, headers, credentials, or adapter config path. MCP clients and portable definitions are Resources selected through extensions and Packages. A deployment platform can supply MCP endpoints, authorization, and credentials at runtime. Tool names may be scoped through the owning adapter extension.

### Product bindings

A product-specific Ompiglet uses the same schema as every other Ompiglet. It can select product extensions, adapters, skills, tools, discovery, and environment requirements. The product injects remote identity, endpoints, credentials, trace context, and deployment policy. No second Ompiglet schema exists.

Remote-agent creation and update belong to the consuming platform. Ompiglet publication creates reusable release state and no remote-agent state.

## Component realization and materialization

Executable components record independent facts:

| Axis | Values | Meaning |
|---|---|---|
| realization | fused, subprocess, external | how the component executes |
| materialization | binary, release, `agentEnv`, external | where required bytes/runtime come from |

Skills/prompts/themes remain exact Resource graph entries without fake process realization. Subprocess use does not create another Binary kind.

## Records and inventory

Ompiglet records use one strict current unversioned shape with two current kinds and one reserved kind:

```text
ompiglet-resolution
ompiglet-binary
`ompiglet-image` (reserved; no Image producer exists)
```

Records bind exact source/effective/graph, Resource/component plan, target, builder/toolchain, environment/external requirements, artifact digest, and verification. They verify artifacts and inventory; they never choose what a shell executes.

Inventory reports source and Binary facets by target/version, including active, stale, missing, unverified, or tampered state. Image inventory is reserved until an Image producer exists. Inventory never discovers artifacts from PATH.

## Build surface

```text
ompig ompiglet build <name> --format script --out <path|->
ompig ompiglet build <name> --format binary --out <path>
```

Script output is a validated Host Ompiglet wrapper and accepts only source-script options. Binary builds use exact records and execute directly. Each current build emits one explicitly selected format.

## Reserved: Image and locked artifact builds

```text
ompig ompiglet build <name> --format image --out <reference>
ompig ompiglet build <name> --format binary|image --locked
ompig ompiglet build <name> --format binary|image --record <path>
```

These forms are reserved in the CLI and fail without producing an artifact. `--format image` reports `Ompiglet Image build is not implemented`.

## Environment responsibility

`agentEnv` constrains the whole agent independent of carrier and component realization:

- omission preserves host execution;
- presence requires Host Ompiglet, script, Binary, Image, and managed deployments to enter or verify the same environment before startup;
- only explicit local one-run `--unsafe-host` may bypass where permitted;
- code-execution sandbox remains separate.

Secrets are logical external requirements and never enter portable source values, records, sessions, argv, logs, or image layers.

## State layout

Ompig writes only Ompig roots and never `.pi`/`~/.pi`:

```text
~/.ompig/
  agent/                  # settings, sessions, npm, git, catalog
  ompiglets/               # editable Ompiglet source
  artifacts/ompiglets/     # managed Binary artifacts; Image artifacts are reserved
  receipts/ompiglets/      # Ompiglet records
  receipts/ompiglet-publications/  # npm source publications made from this machine
  docs/
  state/<extension-id>/   # contributed extension/product state
  state/ompiglet-verify/    # Ompiglet Binary startup verification cache
```

Workspace state stays under `.ompig/`. Directories are created only when needed. Branch-only extra roots and Ompig-managed bin launchers are absent. Explicitly selected import extensions may read Pi state but never write it.

## Command grammar

| Verb | One meaning |
|---|---|
| `add` | validate and register one independent Ompiglet YAML source |
| `remove` | delete explicitly selected local source or Binary facets |
| `pull` | acquire one signed Binary from an explicit release index or GitHub release reference |
| `update` | update an installed signed GitHub Binary release within its signed namespace |
| `build` | emit a source script or produce a Binary artifact |
| `prune` | remove old built Binaries; pulled installs stay |
| `publish` | publish Ompiglet source to npm, or signed Ompiglet Binaries as one GitHub Release; never Agent state |
| `verify` | check a Binary's signature, and with `--source` its recorded Ompiglet digests, without running it |
| `keygen`, `trust` | create a signing key pair; manage trusted signer keys |
| `list`, `show`, `tree`, `lineage`, `validate`, `schema` | read-only inspection |

```text
ompig --ompiglet <name|path>

ompig ompiglet schema
ompig ompiglet validate <name|path>
ompig ompiglet list [--json]
ompig ompiglet show <name>
ompig ompiglet show --record <path>
ompig ompiglet tree [name] [--depth <n>] [--json]
ompig ompiglet lineage <name|path|binary> [--source <name|path>] [--json]
ompig ompiglet add <path|npm:ref|git:ref>
ompig ompiglet pull <release-index-url|github:owner/repo@version>
ompig ompiglet update <name>
ompig ompiglet remove <name> --source|--binary|--all
ompig ompiglet prune [--keep <n>] [--max-size <size>] [--dry-run]
ompig ompiglet build ...
ompig ompiglet publish <name|path> --to github --repo <owner/repo> --sign-key <key>
ompig ompiglet publish <name|path> --to npm
ompig ompiglet keygen <key-path>
ompig ompiglet verify <binary> [--source <name|path>] [--workspace <dir>]
ompig ompiglet trust [list|add|revoke|require]
```

A contributed catalog ref requires an installed product resolver. Stock Ompig has no `marketplace:` or `catalog:` Ompiglet source resolver.

### Source publication

```text
ompig ompiglet publish <name|path> --to npm
```

`publish --to npm` distributes reusable Ompiglet source through npm; it never distributes Agent state. Source registration accepts npm or Git. Signed per-target Binaries publish to and pull from GitHub Releases. The workflow does not use pi.dev or a ompig.dev upload API.

### Reserved: named pulls

`ompig ompiglet pull <name>`, which would acquire a registered release and artifact closure by installed Ompiglet name, does not exist. `ompig ompiglet pull` accepts only an explicit release index URL or GitHub release reference.

Remote runtime management belongs to the consuming platform. Stock Ompig has no product deployment command.

## Vocabulary rules

- Package makes Resources available; Ompiglet selects them.
- Script is a source entry point; Binary/Image are artifacts.
- Record verifies; direct path/reference executes.
- Release distributes; carrier executes.
- Environment constrains the whole agent.
- Status describes observed inventory and creates no entity.
- Products supply bindings without changing the portable Ompiglet schema.

Avoid “self-contained” unless the record proves no release/environment/external requirements. Keep code, JSON, records, API, UI, and docs on this closed vocabulary.
