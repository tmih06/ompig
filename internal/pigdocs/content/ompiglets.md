# Ompiglets

A Ompiglet is one named whole-agent declaration. It selects/scopes Resources,
sets model/prompt/discovery defaults, and may require an agent environment.

Read [Concepts](concepts.md) first. The stable ontology is:

> Packages distribute available Resources. Ompiglets select and scope them.
> Ompiglet releases pin exact Resource closure and executable component plans.
> Host Ompiglet and Ompiglet Binary are current carriers for the same Ompiglet. Ompiglet Image is the reserved OCI carrier.

Binary and image are build outputs, not files you author or separate management
namespaces.

## Run a Ompiglet

```bash
ompig --ompiglet research
OMPIG_OMPIGLET_NAME=research ompig
OMPIG_OMPIGLET_PATH=./research.yaml ompig
```

A bare `ompig` uses ordinary builtins/settings/discovery. Ompiglets are the
Ompiglet-specific capability-scoping mechanism.

`ompig --ompiglet` directly selects source semantics. It never searches PATH or
silently selects a built artifact. A source Ompiglet entering required `agentEnv`
remains the Host Ompiglet carrier. Ompiglet Binaries execute directly.

## What a Ompiglet controls

| Facet | Examples |
|---|---|
| Resources | extensions, skills, prompts, hooks, MCP definitions |
| capability scope | exact built-in and extension model-tool allowlists |
| model | provider/name/context/thinking preference |
| prompt | one system prompt |
| discovery | ambient workspace/user extensions and skills |
| environment | optional required image, Dev Container, or typed source |
| release/build | release SemVer and portable target/output defaults |

### Model can be unconstrained

Leave `model` out when any consumer-enabled model is acceptable. Explicit/current
selection wins, then consumer default. Do not write `model: any` or
`model: default`.

A present model is a preference under Ompig's normal CLI-over-Ompiglet precedence.

## Ompiglet schema

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
      - npm:@example/web-access@^1
    tools: [web_search, web_fetch]

skills:
  - name: commit
    origins: [local:./skills/commit]

tools: [read, write, bash, edit, grep, find, ls]

discovery:
  extensions: []
  skills: [workspace]
```

Omitted root `tools` means Ompig's normal built-ins; `tools: []` exposes none.
Omitted extension tools means all tools from that extension; an empty list
exposes none. CLI and platform policy may narrow these lists but never widen
them. Discovery controls ambient additions only. In an active Ompiglet, omitted
or empty discovery lists mean no ambient Resources; bare Ompig keeps ordinary
upstream discovery.

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

A Ompiglet selects ordinary extensions. This includes an extension that calls
`SetLogin`. Packages make Resources available. Installing a Package does not
activate a Ompiglet.

Package dependencies materialize without changing user/project Package settings.
Only members selected by kind/name become active. Direct `ompig install` keeps
upstream Package discovery behavior.

## Origins, Resource closure, and executable plan

Origins are typed strings tried in declaration order:

- `package:<alias>` selects a named member from a declared Package alias;
- `local:<relative-path>` selects a Ompiglet-owned local Resource;
- npm, Git, HTTP, catalog, and contributed schemes materialize a source and
  select the named Resource from it.

Ompiglet build resolution records exact source/version/content identity. The
Ompiglet artifact contract then assigns each executable component:

- realization: fused, subprocess, or external;
- materialization: binary, Ompiglet release, `agentEnv`, or external.

A Ompiglet Binary may use both fused and subprocess extensions because it remains
Ompig and retains Ompig's subprocess host. Subprocess use does not create
a different binary kind.

Set `build.extensionRealization: fused` to require native compilation of every
selected extension. The Binary build then fails if an extension is not a
fuse-compatible Go factory. It does not use a subprocess fallback. Source
invocation still uses the normal extension host.

Set `build.allowOverlays: true` to let the Binary accept a Ompiglet selected at
launch (`--ompiglet`, `OMPIG_OMPIGLET_PATH`, `OMPIG_OMPIGLET_NAME`) as an overlay on its
own Ompiglet. It is off by default, applies to the built Ompiglet only, and the
overlay cannot replace or remove the Binary's extensions, skills, slots or
system prompt, widen its ceilings, or re-enable what it strips.

Direct-binary startup verifies every non-fused component from its registered
release/environment closure before model/session/tools. It never uses an
arbitrary same-named PATH extension or silently pulls/substitutes content.

## Required agent environment

When `agentEnv` is absent, Ompiglet execution stays on the current host. When
present, exactly one source is selected:

```yaml
agentEnv:
  image: registry.example/dev@sha256:...
  ompigRuntime:
    mode: image
  policy:
    preset: standard
```

or a workspace-anchored Dev Container:

```yaml
agentEnv:
  devContainer: workspace:.devcontainer/devcontainer.json
```

or typed `source`. Ompiglet-owned files use `ompiglet:`; Dev Container files use
`workspace:`. `--workspace <dir>` selects the machine-local workspace (default:
invocation directory). Absolute, escaping, symlink-escaping, and wrong-anchor
paths fail.

Secrets are declared by logical name and consumed by typed agent-environment
bindings; values never enter Ompiglet source or inspection output:

```yaml
secrets:
  - name: github-token
    from: {env: GITHUB_TOKEN}  # or owner-only absolute/~/ file, or resolver ref
agentEnv:
  image: dev:1
  secrets:
    - secretRef: github-token
      target: {env: GITHUB_TOKEN}
```

Missing environment variables, unsafe files, missing/denied contributed
resolvers, and undeclared references fail before model/session/tool startup.

Raw Ompiglet YAML contains no MCP command, URL, headers, credentials, or adapter
config path. Select an MCP client/definition Package or extension; a deployment platform
supplies platform MCP bindings and authorization externally.

The environment constrains the whole agent across current Host Ompiglet and Ompiglet Binary carriers and the planned Ompiglet Image and managed-deployment carriers. Only explicit local one-run `--unsafe-host` may bypass it.

## Commands

```bash
ompig ompiglet list
ompig ompiglet show <name>
ompig ompiglet show --record <path>
ompig ompiglet show <name> --effective --json
ompig ompiglet validate <name|path>
ompig ompiglet schema
ompig ompiglet add <path|npm:<package>|git:<repository>>
ompig ompiglet pull <release-index-url|github:owner/repo[/ompiglet]@version> [--target os/arch]
ompig ompiglet update <name> [--version version] [--target os/arch] [--accept-signer key-id]
ompig ompiglet publish <name|path> --to github --repo <owner/repo> --sign-key <key> [--targets os/arch,...] [--artifacts <dir>] [--tag-prefix <name>/] [--yes]
ompig ompiglet remove <name> [--source|--binary|--all]
ompig ompiglet build <name> --format script --out <path|->
ompig ompiglet build <name> --format binary --out <path>
```

Owned verbs use full words; `ls`, `rm`, and `check` are not aliases.

Binary builds report real phases and elapsed time on stderr (D18). A terminal shows a spinner and dim step text. CI and pipes receive plain phase lines. Add `--verbose` to stream toolchain output with member labels. Packed members share one compiler invocation; fused Go members compile with the final binary. Failures show the current phase, a diagnostic tail, and a hint. The final summary gives the binary path, size, and time after verification and record publication. `--json` keeps stdout as one JSON result. `ompig build [--verbose]` uses the same display when building Stock Ompig from its source checkout.

The native builder compiles Ompig from source with the host Go toolchain. It uses the Ompig checkout that contains the current directory, or `OMPIG_SOURCE_ROOT`. A release binary without a checkout fetches the source of exactly its own version with `go mod download github.com/tmih06/ompig@v<version>` and prints `fetching Ompig v<version> source (cached after first build)`. `GOPROXY` and `GOSUMDB` verify the download, and later builds reuse the Go module cache and a staged copy under `~/.ompig/cache/ompig-source/`. The fetch needs no Git and no checkout. A development build has no published source for its version, so it reports `source-unavailable` and asks for a checkout or `OMPIG_SOURCE_ROOT` (D18).

Without a ready native builder, auto selection uses the built-in `container` builder; `--builder container` selects it explicitly. It runs the build in Podman or Docker: `OMPIG_CONTAINER_ENGINE=docker|podman` selects the engine, and otherwise Podman wins when both are on `PATH`. It pulls the digest-pinned public Go image of Ompig's ci-go CI image, installs exactly the running Ompig release in it with `go install`, and runs that release's native builder. The host needs no Go and no Ompig source, and the target can be any `linux/<arch>` the engine runs; another architecture needs the engine's emulation. Go modules and build output are cached under `~/.ompig/cache/container-go/`. The image contains only the Go toolchain, so a Ompiglet with packed or isolated Rust extensions needs a configured builder image that also has Cargo. A development build cannot install itself in the container, and the container builder does not sign (D18).

Ompiglet YAML is authored directly with a user editor or coding agent. `schema`,
`validate`, `show`, direct `ompig --ompiglet <path>`, and `build` all use the same
closed source contract.

`ompiglet add` accepts a local Ompiglet path, an `npm:` package, or a `git:` repository. For a local path, it validates the source and copyable Ompiglet-relative prompt files, then copies them without replacement into `~/.ompig/ompiglets/`. It does not infer Resources from a directory or installed state, rewrite source, or change Package settings, shell state, Sessions, or artifact inventory. A contributed catalog ref works only when installed product code registers its resolver; Stock Ompig provides no `marketplace:` or `catalog:` Ompiglet resolver.

Add a Ompiglet from npm or Git:

```bash
ompig ompiglet add npm:@acme/review@^1.0.0
ompig ompiglet add git:https://github.com/acme/review.git@v2.0.0
```

Ompig fetches the package or repository, finds the Ompiglet file, validates it, and copies it into `~/.ompig/ompiglets/`. It writes a `<name>.origin.json` file next to the Ompiglet that records the source, the resolved version, the npm integrity or Git commit, and the Ompiglet digest. `ompig ompiglet list` shows the origin.

Ompig reads the Ompiglet path from the `ompig.ompiglet` field of the package's `package.json`. Without that field, it reads `ompiglet.yaml` at the package or repository root. The path must stay inside the package.

A remote Ompiglet must be portable. Ompig refuses a Git URL that contains credentials. The command fails before it fetches anything when `OMPIG_OFFLINE` or `PI_OFFLINE` is set.

### Add from a monorepo

Select a repository subdirectory at a full lowercase commit SHA (D18). The Ompiglet monorepo is `MichaelKinsy/pigpen`. Replace `<full-commit-sha>` with the reviewed commit:

```bash
ompig ompiglet add 'git:https://github.com/MichaelKinsy/pigpen.git@<full-commit-sha>#subdirectory=ompiglets%2Fpig-with-batteries'
```

Ompig reads `ompig.ompiglet` or `ompiglet.yaml` inside the selected directory. It never selects the repository-root Ompiglet as a fallback. The selected checkout must match the commit and contain no modified or untracked files.

This form copies declared relative local Packages, extensions, skills, and prompt files into `~/.ompig/ompiglets/<name>.source/<commit>/`. It rewrites their paths in the registered YAML without changing explicit empty tool scopes. Each local path must exist beneath the Ompiglet file's directory. Executable permissions are preserved. Absolute paths, parent traversal, symlinks, Git metadata, non-regular files, and closures exceeding 4,096 entries or 32 MiB fail before registration. Keep all required build files inside each declared local Resource directory. Use explicit YAML fields rather than aliases or merge keys for this form. `extends`, local agent environments, Dev Containers, and file-based secrets remain unsupported for remote registration.

The origin records the selected source, commit, original and registered Ompiglet digests, and copied file digests. Inventory verifies the closure. Source removal removes only that Ompiglet's recorded files. Other remote source forms still reject local Resource origins.

### Publish source to npm

```text
ompig ompiglet publish <name|path> --to npm [--yes] [--tag <dist-tag>] [--access public|restricted] [--otp <code>]
```

This publishes a Ompiglet's source to npm with the `ompig-ompiglet` keyword so that `ompig ompiglet add npm:<name>` installs it. It is a dry run until `--yes` is present. Ompig validates the Ompiglet, writes the npm package (`ompiglet.yaml`, `package.json`, README, LICENSE, and the system prompt file), replaces each local Package by `npm:<name>@^<version>`, refuses a `name@version` that npm already has, and runs `npm publish`. npm authenticates you; Ompig never handles an npm token. A signed Binary release that this machine published with `--to github`, or that `--binaries github:<owner/repo>` names, is recorded in `package.json` under `ompig.binaries`. See [Publishing](publishing.md).

### Planned (not in this release): named pulls and Image artifacts

```text
ompig ompiglet pull <name>
ompig ompiglet build <name> --format image --out <reference>
ompig ompiglet build <name> --format binary|image --locked
ompig ompiglet build <name> --format binary|image --record <path>
```

Pull by installed Ompiglet name and the reserved Image, `--locked`, and `--record` build paths are not available in this release.

### In-session

```text
/ompiglet                  # inspect the Ompiglet active in this process; read-only
```

Ompiglets do not change the Session wire or bind a transcript to one composition.
Resume, clone, and fork use the current invocation's tools, environment, secrets,
and policy for new work. Select another Ompiglet in a separate Ompig invocation;
that invocation may resume the same Session when you explicitly choose it.

## Ompiglet Binary

A Ompiglet Binary is a target-native Ompig executable for one immutable Ompiglet composition. Its record captures component realization and materialization.

A Ompiglet Binary file does not automatically promise that every component is
in-process or contained in that one file. Its Ompiglet release says what is
fused, what runs as a subprocess, where required bytes/runtimes come from, and
what remains external.

A Ompiglet Binary is an agent executable, not a Ompiglet YAML editor. It supports
normal execution and its built-in Ompiglet. Use raw Ompig to validate, register,
and build Ompiglet source.

## Sign and verify a Ompiglet Binary

Create an Ed25519 author key and sign during a native Binary build:

```text
ompig ompiglet keygen ./author.key
ompig ompiglet build <name> --format binary --out ./ompig-<name> --sign-key ./author.key
```

The private file is mode 0600 and is never replaced. Publish `author.key.pub`, not the private key. The Binary embeds the public key and verifies the signed DSSE manifest and every executable byte before command dispatch.

Check the file without running it and manage local trust explicitly:

```text
ompig verify ./ompig-<name>
ompig ompiglet verify ./ompig-<name>
ompig ompiglet trust add ./author.key.pub
ompig ompiglet trust revoke ed25519:<key-id>
ompig ompiglet trust require on
```

An unsigned Binary reports unsigned. It runs by default, but `trust require on` makes every Ompiglet Binary require a signature by a trusted, non-revoked key. A valid embedded-key signature proves continuity from that key, not the human identity of its holder.

Pull a published per-target Binary from a signed release index:

```text
ompig ompiglet pull https://github.com/acme/reviewer/releases/download/v1.2.3/ompiglet-release.json
ompig ompiglet pull github:acme/reviewer@1.2.3 --target linux/amd64
```

Pull verifies the index signature, complete asset checksum and size, Binary signature, and release identities before installing any file. The first pull pins the signer for that Ompiglet. A different signer is refused unless `--accept-signer ed25519:<key-id>` names the new index key exactly. Revoked keys and an enabled required-signature policy fail before installation. Pulled Binary facets and their signed receipts appear in `ompig ompiglet list` and `show`.

Publish signed Binaries as one GitHub Release named `v<release.version>`:

```text
ompig ompiglet publish reviewer --to github --repo acme/reviewer --sign-key ./reviewer-signing.key
ompig ompiglet publish reviewer --to github --repo acme/reviewer --sign-key ./reviewer-signing.key --artifacts ./dist --yes
```

Publish is a dry run until `--yes` is present: it validates the Ompiglet, key, and targets, checks that the release does not exist, and prints the assets without building or uploading. With `--yes` it uploads one signed Binary per target named `ompig-<name>-<os>-<arch>` (`.exe` for Windows), `SHA256SUMS`, and the signed `ompiglet-release.json` index. Targets come from `--targets`, else `build.targets`, else the host target. A target without a ready signing builder is listed with each builder's reason, and nothing is uploaded; the native builder builds only the host target, and container builders do not sign. `--artifacts <dir>` publishes prebuilt Binaries instead: each file must be signed by `--sign-key` and built from this Ompiglet source and `release.version`, with exactly one Binary per target. Publish runs `gh release view` and `gh release create` with the user's `gh` login and never handles a GitHub token. It refuses an existing release. `docs/examples/ompiglet-release.yml` in the Ompig repository is a reusable GitHub Actions workflow that builds each target on a native runner, attests provenance, and publishes with `--artifacts`.

### Monorepo publication and named Binary updates

Use `MichaelKinsy/pigpen` for the Ompiglet monorepo. Replace `<version>` with a published release version:

```bash
ompig ompiglet publish ./ompiglets/ompig-with-batteries/ompiglet.yaml --to github --repo MichaelKinsy/pigpen --tag-prefix ompig-with-batteries/ --sign-key ./ompig-with-batteries-signing.key --yes
ompig ompiglet pull 'github:MichaelKinsy/pigpen/ompig-with-batteries@<version>'
ompig ompiglet update ompig-with-batteries
ompig ompiglet update ompig-with-batteries --version 2.0.0-rc.1
```

The explicit `--tag-prefix` must equal the manifest name followed by `/`. It selects `ompig-with-batteries/v1.2.3` without changing release SemVer. The default remains unprefixed `v1.2.3`. Ompig never infers a namespace from repository layout. Dry runs, existence checks, uploads, default Git source refs, and URLs use that namespace. Download URLs escape the tag slash as `%2F`.

The signed index records the repository and tag prefix separately from `sourceRef`. Named pulls bind the requested Ompiglet, version, repository, and prefix before downloading a Binary. Receipts retain this signed identity. To identify independently addable monorepo source, pass `--source-ref 'git:https://github.com/MichaelKinsy/pigpen.git@<full-commit-sha>#subdirectory=ompiglets%2Fpig-with-batteries'`; the default source ref selects only the repository's commit or tag.

Update reads the current signed receipt and enumerates GitHub's public releases API. It selects the highest stable SemVer in that exact namespace, never repository-wide `latest`. Drafts, prereleases, invalid versions, and unrelated prefixes are excluded. Use `--version` to select a prerelease explicitly. Incomplete bounded discovery fails. The installed target stays selected unless `--target` overrides it.

Update refuses rollback, repository or namespace changes, revoked keys, and signer changes without an exact `--accept-signer` key ID. Required-signature policy still applies. It verifies the same index, complete checksum and size, Binary signature, and manifest identities as pull. It revalidates continuity under the release-store lock before publishing. An already-current version is a no-op. Indexes without signed GitHub identity cannot supply automatic updates. Update does not refresh registered source Ompiglets or query a catalog. `OMPIG_OFFLINE` and `PI_OFFLINE` prevent update requests.

Sigstore keyless provenance is separate CI identity evidence. Verify it explicitly with `ompig verify --provenance --repo <owner/name> --signer-workflow <workflow> <binary>`. This invokes `gh attestation verify` and may contact GitHub. Ompiglet Binary startup never performs that network check; its Ed25519 verification is offline.

### Planned (not in this release): Ompiglet Image

A Ompiglet Image will contain exact Ompig, redistributable component/runtime closure, and environment. It will perform no first-run code/runtime installation. Network services, secret values, user configuration, and non-redistributable programs will remain explicit requirements.

## Version and records

- `release.version` is Ompiglet release SemVer;
- Ompiglet records use one strict current unversioned shape with resolution and binary kinds; the image kind is reserved;
- extension wire changes ship atomically with the running Ompig binary, staged SDKs, generated cells, and validation rather than carrying an independent version.

Ompiglet records live outside source discovery and bind source/effective/graph,
Resource closure, executable component plan, target, builder, artifact/environment,
and verification identity. Runtime environment agreement is machine-local state, not publication
provenance.

## Where to go next

- [Concepts](concepts.md) for the complete entity/carrier/component model.
- [Packages](packages.md) for Resource distribution and activation.
- [Extensions](extensions.md) for subprocess and fused-capable components.
- [Commands](commands.md) for the exact shipped CLI.

## Replace a bundled extension

A Ompiglet Binary pins every embedded component: fused Go extensions and prebuilt Go or Rust cells. At startup it verifies the registered closure and exits if a component is missing or different, so a bundled extension cannot be swapped inside a built binary. To ship a fix, change the extension source and rebuild the binary with `ompig ompiglet build`.

When two configured origins provide the same extension name, Ompig loads neither and names both origins in the error, so no copy wins silently. During development, run Stock `ompig` with the Ompiglet file or the extension source instead of the built binary: edit the source and run `/reload`. Compiled cells are cached under `~/.ompig/cache/ext` by source digest, so a changed source is rebuilt on reload.
