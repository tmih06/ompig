# Ompiglet design: lean composition and direct outputs

This is the normative maintainer contract for Ompiglet source, validation, registration, invocation, build outputs, records, environments, and external product bindings. Read the [ecosystem ontology](ompiglet-resource-ontology.md) first. Package details are in [Package structure](ompig-package-spec.md); extension hosting is in [runtime cells](extension-runtime-cells.md).

## Purpose

A Ompiglet turns explicit Ompig Resources and policy into one named agent without changing upstream Package behavior. It is the only authored whole-agent entity. A Ompiglet must stay understandable enough to write/read directly while retaining advanced derivation, origin, discovery, environment, and secret controls.

## Closed source shape

```yaml
name: reviewer
description: Focused code-review agent

extends:
  source: npm:@acme/base-review@^1.0.0
  version: "^1.0.0"
  allowWiden: false
  remove:
    extensions: [secondary-linter]
    skills: [old-review]

packages:
  review: npm:@acme/review-resources@^1.0.0

extensions:
  - name: review-tools
    origins:
      - package:review
      - local:./extensions/review-tools
    tools: [review_diff, inspect_tests]

skills:
  - name: code-review
    origins: [package:review]

tools: [read, grep, find, ls]

model:
  provider: anthropic
  name: claude-sonnet-4
  thinking: high

systemPrompt:
  file: prompts/review.md

discovery:
  extensions: []
  skills: [workspace]

agentEnv:
  source: package:review
  ompigRuntime:
    mode: inject
    version: latest
  policy:
    preset: standard

secrets:
  - name: source-token
    from:
      ref: vault:source-token

release:
  version: 1.2.0

build:
  targets: [darwin/arm64, linux/amd64]
  outputName: ompig-reviewer
```

Allowed root fields are exactly:

```text
name description extends packages extensions skills tools model
systemPrompt discovery agentEnv secrets release build slots strip
```

The schema is closed: every field outside that root vocabulary fails without an alias. `ompig ompiglet schema` prints it as JSON Schema.

## Packages and origins

`packages` maps an author alias to one typed source. Declaring it materializes no member and mutates no Package settings. Extension/skill `origins` are ordered typed source strings. The first exact successful origin wins and enters resolution identity.

Supported source families use the shared parser:

```text
package:<alias>
local:<relative-path>
npm:<package>
git:<repository>
http(s):<source>
<contributed-scheme>:<locator>
```

A contributed scheme works only when installed product code registers its resolver. Stock Ompig does not provide a `marketplace:` or `catalog:` resolver.

Local Ompiglet-owned origins resolve from the declaring Ompiglet's directory. `agentEnv.devContainer` resolves from invocation `workspace:`. Absolute, escaping, and symlink-escaping paths fail.

## Tools

Root `tools` controls Ompig built-in model tools. Extension `tools` controls only tools registered by that extension. Both are exact lists:

| Shape | Root | Extension |
|---|---|---|
| omitted | normal Ompig built-ins | all extension tools |
| `[]` | no built-in tools | load extension, expose no tools |
| list | exact built-in ceiling | exact extension tool ceiling |

Effective tools are:

```text
runtime registration ∩ Ompiglet allowlists ∩ CLI narrowing ∩ platform authorization
```

CLI/platform inputs never silently widen a Ompiglet. Slash commands and shortcuts are not Ompiglet capability policy. A Ompiglet can only remove built-in slash commands, through `strip.commands`.

## Strip lists

`strip` disables built-ins (D92). It has five lists: `tools`, `commands` (with the leading slash; `/ompiglet` included), `extensions`, `apis` (model API implementations), and `features`. `internal/pigstrip/ids_generated.go` holds the valid IDs, and `make ompiglet-strip-ids` regenerates it and the matching ID enums of `coding/ompiglet/ompiglet.schema.json`. A list is in deny mode (the IDs it names are stripped) or in keep mode through `strip.keep.<list>` (every ID it does not name is stripped, including IDs a later Ompig adds). One file cannot use both modes for one list.

Strip lists inherit through `extends`. `extends.remove.strip` re-enables IDs and requires `extends.allowWiden`. A child can narrow keep mode but cannot leave it. Validation refuses a strip list that removes every built-in tool together with `/quit`.

A Ompiglet Binary leaves the code of a stripped built-in extension, model API, or feature out of the executable. Binary records keep the strip table of the Ompig that built them (`stripTable`) and the keep lists (`stripKeep`). `ompig ompiglet build` and `ompig ompiglet show` print a strip delta report when this Ompig's table has new IDs since that record. The user guide is `site/docs/ompiglets.md`, section "Strip built-ins".

## Slots

`slots` names members that replace a part of Ompig. `slots.frontend` is the only slot: a Go module whose root package exports `func Frontend() frontend.Frontend` (`extensions/sdk/frontend`). `ompig ompiglet build --format binary` fuses it, and the member draws the interactive mode in place of the ANSI renderer (D91). A child inherits the slot, replaces it by naming it, or removes it with `extends.remove.slots`. Source invocation and Stock Ompig fuse no frontend member. The user guide is `site/docs/ompiglet-binaries.md`, section "Frontend members".

## Discovery

Discovery controls ambient additions only:

```yaml
discovery:
  extensions: [workspace, user]
  skills: [workspace]
```

Allowed sources initially are `workspace` and `user`. Empty or omitted lists in an active Ompiglet mean no ambient Resources of that kind. Explicit Ompiglet entries always load; explicit caller additions remain subject to current runtime policy and artifact restrictions. Bare Ompig without a Ompiglet keeps upstream Package/discovery behavior.

External-harness adapters belong to the harness extension and its own namespaced config, not Ompiglet schema.

## Derivation

`extends` owns source, version constraint, widening intent, and removals. Depth is eight with cycle detection and exact source digests. Every base/child resolves its own paths before merge. Whole fields replace; named collections preserve base order, replace by identity, append new identities, and remove only existing identities. Unsafe widening and dangling Package aliases fail. Release/build/ derivation metadata does not inherit into effective composition.

## Model and prompt

Omitted `model` means unconstrained consumer selection. Present model is a simple preference using provider/public name, context window, and thinking level. It is not a hard policy. `any`, `default`, capability, fallback, and policy fields are rejected until a concrete enforced requirement is approved.

`systemPrompt` accepts inline text or one Ompiglet-anchored file. Additional prompt Resources should be selected through Packages/extensions rather than a dormant `agentPrompts` array.

## MCP boundary

Ompiglet source contains no MCP endpoint, command, args, headers, credentials, or config path. MCP clients and portable definitions are Resources selected through extensions and Packages. A deployment can supply MCP bindings, authorization, and credentials at runtime. MCP tools are scoped through their owning adapter extension's `tools` list.

## Product bindings

A product uses the same Ompiglet schema as any other consumer. It can select product extensions, adapters, skills, tools, discovery, and environment requirements. Product URLs, remote identities, model bindings, MCP endpoints, credentials, trace context, and deployment policy stay outside portable Ompiglet YAML.

## Secrets

Secrets are logical declarations with one `env|file|ref` source. Consumers use typed secret bindings. Values never enter Ompiglet source metadata, records, sessions, logs, argv, image layers, or hashes. Raw Ompig has no built-in secret store; contributed resolvers handle `ref` values. Missing or unauthorized required values fail before startup. `--unsafe-host` never bypasses secret requirements.

## Environments

`agentEnv` accepts exactly one of direct image, standard Dev Container path, or typed source resolving one environment. Omission preserves host execution. Presence constrains direct source invocation, generated script, and Ompiglet Binary. A Ompiglet Image or managed deployment must honor the same requirement. The whole Ompig process enters or verifies it before startup; code-execution sandbox is separate.

## Sessions

Ompiglets do not change the upstream Session header or bind a transcript to a Ompiglet, carrier, component plan, or environment. Resume, clone, and fork load the selected conversation under the current process composition; current tool, secret, environment, and platform policy governs every new action. `/ompiglet` inspects the Ompiglet active in this process and writes nothing to the Session; a Ompiglet that strips it (`strip.commands: [/ompiglet]`) runs without it. A different Ompiglet is selected with another `ompig --ompiglet` invocation, which may resume the same Session when the caller chooses it.

## YAML lifecycle

Ompiglet YAML is authored directly by a user, editor, or agent. The closed schema is the power-user surface; Ompig applies no inferred directory, Session, or installed-Resource selection policy.

```text
ompig ompiglet schema
ompig ompiglet validate <name|path>
ompig --ompiglet <name|path>
ompig ompiglet add <path|npm:ref|git:ref>
```

`ompiglet add` validates one Ompiglet and its portable closure, then copies them without replacement into `~/.ompig/ompiglets/`. An npm or Git source is materialized without changing Package settings and retains an adjacent origin record with its exact version or commit. The original source, shell state, Sessions, and artifact inventory remain unchanged. Editing means editing the YAML source directly.

A product may register a contributed Ompiglet source resolver. Catalog refs work only while that product resolver is installed.

### npm and Git Ompiglet registration

```text
ompig ompiglet add npm:<package>
ompig ompiglet add git:<repository>
```

The remote-source path materializes published Ompiglet source, validates its portable closure, and retains its origin in an adjacent `<name>.origin.json` record with the exact npm version/integrity or Git commit. It reads `ompig.ompiglet` from `package.json`, or `ompiglet.yaml` at the source root when that field is absent. It rejects local Resource sources and credential-bearing Git URLs. `OMPIG_OFFLINE` or `PI_OFFLINE` forbids fetching remote sources.

## Invocation and outputs

Source invocation is direct:

```bash
ompig --ompiglet reviewer
ompig --ompiglet ./reviewer.yaml
```

Build outputs:

```text
ompig ompiglet build reviewer --format script --out ~/.local/bin/reviewer
ompig ompiglet build reviewer --format script --out -
ompig ompiglet build reviewer --format binary --out ./ompig-reviewer
```

On Unix a script is an executable POSIX wrapper. On Windows it is a two-line cmd.exe launcher (D69). The Unix form is:

```sh
#!/bin/sh
exec ompig --ompiglet '/canonical/reviewer.yaml' "$@"
```

It is untracked, receives no record, and is never removed by Ompig. A Binary executes directly and uses Ompiglet records. Each current build invocation requires exactly one `script|binary` format.

A Binary build of a Ompiglet that lists no extensions produces Stock Ompig's own parts with that Ompiglet baked in. A Ompiglet that lists extensions that resolve to none fails with `ompiglet resolves to no extensions`. A Binary embeds the sources of each TypeScript or JavaScript extension and runs them with the installed Node (22.13 or newer), as Stock Ompig does.

### Reserved: Image and locked artifact builds

```text
ompig ompiglet build reviewer --format image --out registry/reviewer
ompig ompiglet build reviewer --format binary|image --locked
ompig ompiglet build reviewer --format binary|image --record <path>
```

`--format image`, `--locked`, and `--record` are reserved CLI shapes. They fail without writing output.

Set `build.extensionRealization: fused` when every selected extension must be compiled into the Ompiglet Binary. A Binary build then rejects every extension that is not a fuse-compatible Go factory. It does not fall back to a subprocess component. This requirement does not change source invocation.

Set `build.allowOverlays: true` and the Ompiglet Binary built from the Ompiglet accepts a Ompiglet selected at launch (`--ompiglet`, `OMPIG_OMPIGLET_PATH`, `OMPIG_OMPIGLET_NAME`) as an overlay on its baked Ompiglet (D202). Otherwise the Binary refuses those selectors. The setting belongs to the built Ompiglet and is not inherited through `extends`; `Bake` keeps it in the baked Ompiglet, so the Binary's Ompiglet digest covers it. The overlay resolves as `extends` resolves a child over its base, with the Binary's name kept, and cannot replace or remove a baked extension or skill, set `slots` or `systemPrompt`, widen `tools`, `discovery`, `agentEnv` or an extension's `tools`, or re-enable a stripped built-in.

## Records and inventory

Ompiglet records use one strict current unversioned shape with current kinds `ompiglet-resolution` and `ompiglet-binary`; `ompiglet-image` is reserved until an Image producer exists. They bind source/effective/graph, Resource/component closure, target, builder/toolchain, environment/external requirements, artifact digest, and verification. Records verify; they do not launch.

```text
ompig ompiglet list [--json]
ompig ompiglet show <name>
ompig ompiglet show --record <path>
ompig ompiglet tree [name] [--depth <n>] [--json]
ompig ompiglet lineage <name|path|binary> [--source <name|path>] [--json]
ompig ompiglet verify <binary> [--source <name|path>] [--workspace <dir>]
ompig ompiglet remove <name> --source|--binary|--all
ompig ompiglet prune [--keep <n>] [--max-size <size>] [--dry-run]
```

`prune` removes old built Binaries and keeps the newest `n` of each Ompiglet and target (default 2). Pulled installs stay. `verify` checks a Binary's signature, and with `--source` its recorded Ompiglet digests against that source, without running it. `--workspace` names the workspace the build used for `workspace:` paths (default: the current directory, as for the build). `ompig ompiglet keygen <key-path>` creates an Ed25519 signing key pair, and `ompig ompiglet trust [list|add|revoke|require]` manages trusted signer keys.

### Pull signed Binary releases

```text
ompig ompiglet pull <release-index-url|github:owner/repo@version> [--target os/arch] [--version version] [--accept-signer key-id]
```

A direct pull downloads one signed release index and the Binary selected for the requested target. The default target is the current operating system and architecture. Distribution uses HTTPS, including GitHub Releases, and does not delegate to Pi services (D18).

The index is an Ed25519 DSSE envelope with payload type `application/vnd.ompig.ompiglet-release+json`. Its unversioned payload contains `ompiglet`, release `version`, `ompigVersion`, `sourceRef`, `signer`, and `binaries`. Each target entry contains `url`, `sha256`, and `size`. The checksum covers the entire downloaded file, including its Binary signature trailer.

Release URL policy failures do not quote rejected references. HTTP transport diagnostics omit private URL text while retaining cancellation and error identity for API callers.

Pull verifies the index signature, local trust policy, signer continuity, complete asset size and checksum, Binary signature, and matching Ompiglet, release, target, and Ompig identities before publishing managed files. A receipt retains the signed index and Binary manifest. Receipt verification binds the artifact checksum and size to that signed index. The current pointer identifies the selected artifact, receipt, and signer. The first successful pull pins the signer. A later signer change requires an exact `--accept-signer` value and must still satisfy local trust policy.

Publication rechecks the signer under an operating-system lock shared by CLI processes. A busy store fails with a retry diagnostic. The lock is released when its handle closes or its process exits; its file remains under `receipts/ompiglets/`. Publication and pulled-release inventory reject symlinked managed path components. File operations use opened directory handles so replacing a parent path cannot redirect them outside the managed store. Host policy compares DNS names after IDNA normalization and removal of terminal DNS dots, before requesting indexes, assets, or redirect destinations.

### Published Ompiglet Binary releases

```text
ompig ompiglet publish <name> --to github --repo <owner/repo> --sign-key <key> [--yes]
ompig ompiglet pull <release-index-url|github:owner/repo@version>
```

`publish --to github` uploads signed per-target Ompiglet Binaries, `SHA256SUMS`, and a signed release index as one GitHub Release. `pull` verifies a signed release index and installs one target's Binary.

### Published Ompiglet source on npm

```text
ompig ompiglet publish <name|path> --to npm [--yes] [--tag <dist-tag>] [--access public|restricted] [--otp <code>]
        [--npm-name <name>] [--package-map <alias>=npm:<name>@<range>]... [--binaries github:<owner/repo>|--no-binaries]
```

`publish --to npm` writes one npm package from an authored Ompiglet and runs `npm publish`. It is a dry run unless `--yes` is present, and it refuses an existing `name@version` (`npm view`). npm authenticates the author, including a one-time password, a passkey, and trusted publishing from CI (`--provenance` is added when `ACTIONS_ID_TOKEN_REQUEST_URL` is set). Ompig never reads or stores an npm token.

Publication validates the Ompiglet as `ompiglet validate` does and refuses what `ompiglet add` refuses: `extends`, `agentEnv.devContainer`, file-based secrets, and local Resource origins. The generated package contains `ompiglet.yaml`, `package.json`, `README.md`, `LICENSE`, and the `systemPrompt.file` file. Its `ompiglet.yaml` is the authored YAML with every `local:` Package replaced by `npm:<name>@^<version>`, taken from that Package's `package.json` (or `--package-map <alias>=<ref>`), with `?registry=<url>` when that `package.json` sets an HTTPS `publishConfig.registry`. A local Package with no public name and version, or with a registry an `npm:` source cannot name, fails the publication. An `npm:` Package without its own registry that no published version satisfies is a warning in the dry run and fails `--yes`; a registry-qualified Package is not prechecked.

The generated `package.json` takes `name` from `--npm-name`, a `package.json` beside the Ompiglet, or the Ompiglet name, and `version` from `release.version`. It adds the keyword `ompig-ompiglet`, so catalogs find the package by npm keyword search, and the block `"ompig": {"ompiglet": "ompiglet.yaml", "binaries": {"ref": "github:<owner>/<repo>[/<name>]@<version>", "signer": "<key id>"}}`. `ompig.ompiglet` is the path `ompiglet add` reads. `ompig.binaries` is present when the version has a signed GitHub Binary release: `--binaries github:<owner/repo>` reads the signed release index, and otherwise a release this machine published with `publish --to github --yes` is recorded in `receipts/ompiglet-publications/`. `license`, `author`, `repository`, `homepage`, `bugs`, `funding`, and `publishConfig` come from a `package.json` beside the Ompiglet; scripts and dependencies do not. `ompig ompiglet add npm:<name>` accepts exactly what publication produces.

### Reserved: pull by name

`ompig ompiglet pull <name>` does not exist. `ompig ompiglet pull` takes an explicit release index URL or GitHub release reference. `ompig ompiglet update <name>` updates an installed signed GitHub Binary release within its signed namespace.

Remote agent creation and update belong to the consuming platform.

## State

Editable source lives in `~/.ompig/ompiglets/`. Managed Binary outputs and records live under `~/.ompig/artifacts/ompiglets/` and `~/.ompig/receipts/ompiglets/`. npm source publications are recorded under `~/.ompig/receipts/ompiglet-publications/`. A signed Binary records a passing startup check in `~/.ompig/state/ompiglet-verify/verified.json`, keyed by its resolved path, file identity, signature block, and trust policy, so a later start with no change skips the whole-file hash. Image artifacts are reserved. Explicit scripts live only at user-selected paths. Ompig writes no `.pi` state; managed Ompiglet state stays within those Ompiglet roots.

## Current implementation boundary

The lean closed source schema, typed Package/origin vocabulary, derivation object, strip lists, the frontend slot, direct source invocation, Ompiglet Binary producer, remote npm/Git registration, GitHub Binary publication, npm source publication, signed-index pull, and `ompig ompiglet update <name>` are implemented. Pull by installed Ompiglet name, locked records, and Ompiglet Image production do not exist.
