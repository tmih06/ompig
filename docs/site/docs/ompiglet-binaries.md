# Ompiglet Binaries

A Ompiglet Binary is a target-native Ompig executable for one fixed Ompiglet composition. It delivers an agent application directly without creating another harness implementation.

A Ompiglet Binary is a carrier for a Ompiglet. It does not introduce another configuration schema.

## Build a Binary

Validate the Ompiglet first:

```bash
ompig ompiglet validate ./agents/reviewer.yaml
```

Build the executable:

```bash
ompig ompiglet build ./agents/reviewer.yaml \
  --format binary \
  --out ./ompig-reviewer
```

Run it directly:

```bash
./ompig-reviewer
```

The build records the Ompiglet, its Resource closure, its component plan, the target, and verification information.

A Ompiglet needs no extension to build (D18). A base that only strips built-ins and sets defaults, such as a `model`, builds a Binary of Ompig's own parts with that Ompiglet baked in:

```yaml
name: ompig-core
model:
  provider: openai
  name: gpt-5
strip:
  extensions: [mcp, codemode, tool-search]
  commands: [/share]
```

Whether a Ompiglet lists extensions is read from its effective form, after `extends` and `extends.remove`. A Ompiglet that lists extensions must still resolve at least one of them: an extension whose origins do not resolve fails the build as `unresolved extension`, and when none resolves the build also reports `ompiglet resolves to no extensions`. A child that extends an empty base and adds extensions builds as any other Ompiglet.

### Build progress

Ompig reports build progress on stderr (D18). A terminal shows a spinner, the current phase, elapsed time, and dim step text. CI and pipes receive one plain line per phase. The final summary gives the binary path, byte size, and total time after verification and record publication.

Add `--verbose` to stream toolchain output with member labels:

```bash
ompig ompiglet build ./agents/reviewer.yaml --format binary --out ./ompig-reviewer --verbose
```

Packed members share one compiler invocation and label. Fused Go members compile with the final binary. Module resolution, compilation, and linking can occur inside one compiler invocation, so Ompig reports that operation as one phase rather than estimating progress. A failed build names its phase, shows the diagnostic tail, and prints a hint. With `--json`, progress stays on stderr and stdout remains one JSON result.

The source-checkout command `ompig build [--verbose]` uses the same progress display. It builds Stock Ompig, not a Ompiglet Binary.

### Worked example

This is the `reviewer` Ompiglet of [Ompiglets](/docs/latest/ompiglets#worked-example-a-read-only-reviewer), built, checked and run. The output was captured from Ompig built from a source checkout (`OMPIG_SOURCE_ROOT` named it), with the Go toolchain's cache already warm. A first build compiles Ompig and takes longer. A released `ompig` fetches its own source instead (see [Ompiglets](/docs/latest/ompiglets#build-outputs)).

```bash
ompig ompiglet add ./agents/reviewer.yaml
ompig ompiglet build reviewer --format binary --out ./ompig-reviewer
```

Ompig prints one line per phase to the standard error, then the result:

```text
Resolving manifest (0s) — reviewer
Resolving extensions (0s) — reviewer
Packing resources (0s) — Inlining prompts and skills
Selecting builder (0s) — auto
Locking build inputs (0s) — Hashing source, extensions, and toolchain identities
Packing build inputs (3s) — Embedding cells, resources, and component records
Compiling and linking Go binary (3s) — reviewer → /home/you/project/ompig-reviewer
Checksumming and verifying binary (9s) — /home/you/project/ompig-reviewer
Writing binary and records (9s) — Publishing verified artifact to the managed store
Built /home/you/project/ompig-reviewer (69042439 bytes) in 10s
```

The file is a complete Ompig with the Ompiglet inside, 69 MB here. It runs on its own:

```bash
./ompig-reviewer --version
ompig ompiglet verify ./ompig-reviewer
ompig ompiglet list
```

```text
0.5.0+1.2.0
```

```text
Ompiglet Binary: ./ompig-reviewer
Signature: unsigned (no signature: anyone who can rewrite this file can rewrite its build record)
```

`verify` exits 1 for an unsigned Binary. Sign one with `--sign-key` ([Sign a Binary](#sign-a-binary)). `ompig ompiglet list` now fills the `BINARY` column for the Ompiglet:

```text
NAME      SCOPE  SOURCE                                BINARY       IMAGE  DESCRIPTION
reviewer  user   /home/you/.ompig/ompiglets/reviewer.yaml  linux/amd64  -      Reviews a change and lists risks.
```

Run it like `ompig`, with no `--ompiglet` option, because the Ompiglet is built in (a Binary refuses `--ompiglet` unless its Ompiglet [accepts overlays](#accept-overlay-ompiglets)). Checked with a test model, the provider request carried the Ompiglet's own system prompt text and only the tools the Ompiglet allows (`read`, and `codemode` because an MCP server was configured on that machine):

```bash
./ompig-reviewer -p "Review the last commit"
```

Copy the file to another computer of the same operating system and architecture and it runs there with no Ompig installed. It still needs a model and credentials on that computer.

## Sign a Binary

Create an Ed25519 author key once. Keep the private file secret and publish the `.pub` file:

```bash
ompig ompiglet keygen ./reviewer-signing.key
```

Sign during the native build:

```bash
ompig ompiglet build ./agents/reviewer.yaml \
  --format binary \
  --out ./ompig-reviewer \
  --sign-key ./reviewer-signing.key
```

The signature covers every executable byte and a DSSE manifest that names the Ompiglet, target, Ompig version, resolution record, component plan, components, and embedded files. The Binary embeds the public half of its author key and checks the signature before command dispatch. An altered signature, manifest, Ompiglet, record, component, or executable byte makes the Binary refuse to run.

Check a file without running it:

```bash
ompig verify ./ompig-reviewer
ompig ompiglet verify ./ompig-reviewer
```

`ompig verify` reports a valid Ompiglet signature as `ok` and an unsigned file as `n/a`; because it also verifies arbitrary archives and data files, it applies the required-signature policy only after a Ompiglet signature block identifies the target. `ompig ompiglet verify` is Ompiglet-specific and succeeds only when the file carries a valid signature. Both commands reject a signed file whose key is revoked, and Ompiglet Binary startup enforces the required-signature policy for unsigned Binaries.

Check that a Binary still matches its source:

```bash
ompig ompiglet verify ./ompig-reviewer --source reviewer
```

```text
Ompiglet Binary: ./ompig-reviewer
Signature: signed by ed25519:<key-id> (not in your Ompiglet trust store: this proves it is unchanged since that key signed it, not who holds the key)
Provenance: matches /home/you/.ompig/ompiglets/reviewer.yaml
```

`--source` bakes the named Ompiglet as the build does and compares two digests with the ones the Binary signed: the source digest of the Ompiglet file and the Ompiglet digest of the definition baked into the Binary. A changed Ompiglet file fails the source digest. A change that alters the baked definition, such as an edited prompt file, skill, or base Ompiglet, also fails the Ompiglet digest. A Ompiglet built with `--workspace <dir>` is checked with the same `--workspace`. `ompig ompiglet lineage ./ompig-reviewer --source reviewer` reports the same check beside the `extends` chain.

Trust and policy are explicit local actions:

```bash
ompig ompiglet trust add ./reviewer-signing.key.pub
ompig ompiglet trust list
ompig ompiglet trust revoke ed25519:<key-id>
ompig ompiglet trust require on
```

A valid signature by the embedded author key proves that the file has not changed since that key signed it. It does not prove who controls the key. Add a reviewed public key to the trust store to attach local identity, and enable `require on` when every Ompiglet Binary must be signed by a trusted, non-revoked key.

## Publish to GitHub Releases

Set `release.version` in the Ompiglet. Then publish signed Binaries as one GitHub Release named `v<release.version>`:

```bash
ompig ompiglet publish ./agents/reviewer.yaml \
  --to github \
  --repo acme/reviewer \
  --sign-key ./reviewer-signing.key
```

Without `--yes`, publish is a dry run. It validates the Ompiglet, the signing key, and every target, checks that the release does not exist yet, and prints the assets it would upload. It builds and uploads nothing. Add `--yes` to build, sign, and upload these assets:

- one signed Binary per target, named `ompig-<name>-<os>-<arch>` (with `.exe` for Windows);
- `SHA256SUMS`, which lists the SHA-256 of each Binary;
- `ompiglet-release.json`, the release index, signed with the same key.

Targets come from `--targets os/arch,...`, else `build.targets`, else the host target. The native builder builds only the host target, and a container builder does not sign, so one machine can usually publish only its own target. When a target has no ready signing builder, publish lists each missing target with every builder's reason and uploads nothing. Each build also writes the managed build record that `ompig ompiglet build` writes.

To publish several targets, build each target on a matching host, collect the Binaries in one directory, and publish them without building:

```bash
ompig ompiglet publish ./agents/reviewer.yaml --to github --repo acme/reviewer \
  --sign-key ./reviewer-signing.key --artifacts ./dist --yes
```

Publish identifies each file in `--artifacts` by its signed manifest. Every file must be a Binary of this Ompiglet source and `release.version`, signed by `--sign-key`. The directory must hold exactly one Binary for each target, and every Binary must come from the same Ompig version. Before the upload, Ompig checks each staged asset against the signed index with the checks that `ompig ompiglet pull` applies.

Ompig runs the GitHub CLI (`gh release view` and `gh release create`) with your existing `gh` authentication and never handles a GitHub token. It refuses a release that already exists, because a published release is immutable. A SemVer prerelease version, such as `2.0.0-rc.1`, publishes as a GitHub prerelease. `--commit <sha>` creates a missing release tag at that commit. The index records the Ompiglet source as `git:github.com/<owner>/<repo>@<commit or tag>` unless `--source-ref` names another npm or Git source. Publication contacts GitHub only through `gh`, and it never contacts pi.dev or a Ompig service.

### Publish from a monorepo

Use an explicit Ompiglet-name namespace when several Ompiglets share a repository (D18). For `ompig-with-batteries` in `MichaelKinsy/pigpen`, use the following shape. Replace `<version>` with a published release version:

```bash
ompig ompiglet publish ./ompiglets/ompig-with-batteries/ompiglet.yaml \
  --to github --repo MichaelKinsy/pigpen --tag-prefix ompig-with-batteries/ \
  --sign-key ./ompig-with-batteries-signing.key --artifacts ./dist/ompig-with-batteries --yes
ompig ompiglet pull 'github:MichaelKinsy/pigpen/ompig-with-batteries@<version>'
```

`--tag-prefix` must be `<manifest-name>/`. It produces `ompig-with-batteries/v1.2.3` while `release.version` stays `1.2.3`. Omitting the option keeps `v1.2.3`. Ompig does not infer a namespace by scanning a checkout, so adding another Ompiglet cannot change existing release names. Dry runs, existing-release checks, uploads, default Git source refs, and download URLs use the same namespace. URLs escape the slash as `%2F`.

The signed index records the GitHub repository and tag prefix independently of the source ref. A named pull rejects a different Ompiglet name, version, repository, or prefix. Its receipt retains that signed identity. For independently addable monorepo source, provide `--source-ref 'git:https://github.com/MichaelKinsy/pigpen.git@<full-commit-sha>#subdirectory=ompiglets%2Fpig-with-batteries'`. Without `--source-ref`, the source ref identifies the repository's commit or namespaced tag, not a selected subdirectory.

The reusable workflow [`docs/examples/ompiglet-release.yml`](https://github.com/tmih06/ompig/blob/main/docs/examples/ompiglet-release.yml) builds each target on a native GitHub runner, attests the build provenance of each Binary, and publishes the release with `--artifacts`. Copy it to `.github/workflows/ompiglet-release.yml` in the Ompiglet's repository. Its header shows the workflow that calls it.

## Pull a published Binary

A publisher can place signed per-target Binaries and a signed `ompiglet-release.json` index on GitHub Releases or another HTTPS host. Pull the Binary for the current target:

```bash
ompig ompiglet pull https://github.com/acme/reviewer/releases/download/v1.2.3/ompiglet-release.json
```

Select another target or use a GitHub release reference explicitly:

```bash
ompig ompiglet pull github:acme/reviewer@1.2.3 --target linux/amd64
```

Ompig verifies the release-index DSSE signature, the complete asset size and SHA-256, the Binary signature trailer, and the Ompiglet, release, target, and Ompig identities before it installs any file. The first successful pull pins the signer key for that Ompiglet. A later signer change is refused unless you name the new key exactly with `--accept-signer ed25519:<key-id>`. Local revocation and `trust require on` apply to pulled releases.

Pulled Binaries live under `~/.ompig/artifacts/ompiglets/`, and their signed receipts live under `~/.ompig/receipts/ompiglets/`. `ompig ompiglet pull` prints the Binary's path on its `binary:` line. Run that file directly; `ompig --ompiglet <name>` does not run a pulled Binary. `ompig ompiglet list` and `show` report them as Binary facets. Pull requests only the supplied index and the asset URL signed into that index; it performs no automatic catalog or Ompig service request.

### Update an installed GitHub Binary

```bash
ompig ompiglet update ompig-with-batteries
ompig ompiglet update ompig-with-batteries --version 2.0.0-rc.1
```

Update reads the repository and tag namespace from the current signed receipt. It enumerates GitHub's public release API and selects the highest stable SemVer in that namespace. It never uses repository-wide `latest`, even for unprefixed releases. Drafts, prereleases, unrelated prefixes, and invalid versions do not enter automatic selection. Use `--version` to request a prerelease explicitly. Discovery fails if the bounded release inventory is incomplete.

Update preserves the installed target unless `--target os/arch` selects another. It refuses rollback, repository or namespace changes, revoked keys, and unapproved signer changes. `--accept-signer ed25519:<key-id>` explicitly authorizes a new signer. The same checksum, size, signature, and manifest checks as pull apply before the current pointer changes. An already-current release is a no-op. An index without signed GitHub identity cannot supply automatic updates. Update does not refresh registered source Ompiglets or query a catalog. `OMPIG_OFFLINE` and `PI_OFFLINE` prevent update requests.

### Sigstore keyless provenance

An Ed25519 Ompiglet signature is offline artifact integrity and local trust policy. Sigstore keyless provenance is separate evidence that a named CI workflow built an artifact. A release workflow can publish GitHub build provenance, and the recipient can verify it explicitly:

```bash
ompig verify --provenance \
  --repo acme/reviewer \
  --signer-workflow acme/reviewer/.github/workflows/ompiglet-release.yml \
  ./ompig-reviewer
```

This command delegates to `gh attestation verify`, which checks the Sigstore bundle, certificate identity, and transparency-log inclusion. It requires the GitHub CLI and may contact GitHub's attestation service. Ompig never performs this network check at startup. Startup signature verification remains offline.

## What the Binary contains

A Ompiglet Binary always contains:

- the Stock Ompig execution engine;
- one fixed Ompiglet;
- its resolution and component plan;
- every extension or native component marked as Binary materialized;
- verification data required at startup.

A Binary can still require external items when its plan declares them:

- an external MCP service;
- a secret value;
- a configuration file;
- an interpreted runtime;
- a component supplied by a release or required agent environment.

Do not describe a Binary as self-contained unless its record proves that none of these requirements remain.

## Realization and materialization

Each executable component records two independent facts.

| Axis | Values | Question answered |
|---|---|---|
| Realization | fused, subprocess, external | How does the component execute? |
| Materialization | binary, release, `agentEnv`, external | Where do its bytes and runtime come from? |

A subprocess component does not create a different Binary type. It remains part of the same Ompiglet application and component plan.

## Fused Go extensions

A compatible Go factory can be fused into the Binary. Ompig compiles the extension into the executable and starts it through the same registration contract used by subprocess extensions.

Advantages:

- one target-native executable;
- no target-side Go compiler;
- no extension process startup;
- no serialization cost for process placement;
- exact extension source included in the build closure.

Costs:

- the Binary must be rebuilt when the extension changes;
- the artifact is target specific;
- the extension shares the Ompig process failure boundary;
- fused code requires stricter review;
- only compatible Go factories can fuse.

Fused behavior must match source subprocess behavior. Fusion is a delivery optimization, not a second extension API.

## Fused members with external Go modules

A fused extension or frontend member may require Go modules beyond the Ompig SDK. The build links them into the Binary through Ompig's own module, without writing into the Ompig source tree:

- The member's `require` directives join Ompig's requirements by Go's minimum version selection. Each module gets the highest version Ompig or any member requires, so a member never downgrades Ompig. When a member raises one of Ompig's own requirements, the build prints a `note:` line naming the module and both versions.
- Ompig and its extension SDK are pinned by the Ompig source being built. A member that requires a newer SDK version than that source fails the build with an error naming both versions; build it with a Ompig source at that version or later.
- The member's `replace` directives for other modules apply to the build, with relative paths resolved against the member. Two members, or a member and Ompig, that replace one module differently fail the build.
- Checksums come from the member's `go.sum` together with Ompig's `go.sum` and, for a checkout with `go.work`, its `go.work.sum`. Keep the member tidy (`go mod tidy`) so its `go.sum` covers its module graph. The build verifies these checksums under `-mod=readonly`, works offline once the modules are in the Go module cache, and succeeds from a read-only Ompig checkout. A missing checksum stops the build instead of being written into the checkout.

The member's `go.mod` and `go.sum`, and the source of each local replacement, are part of the Binary's recorded build inputs, so a change to the member's module graph changes the Binary's identity.

### What the fused vet checks

Before it links fused members, the build vets them for process-global hazards: calls such as `os.Exit`, `os.Chdir`, `log.Fatal`, and writes to standard output, which a fused member would run inside the Ompig process instead of an isolated subprocess (see [Fused Go extensions](#fused-go-extensions)). The vet reads Go source and follows imports, so its reach is the source it can read in the build:

- **Vetted:** each member package and every package it imports, directly or indirectly, from the member's own module and from the workspace modules it names.
- **Not vetted:** every other module the members require, direct or indirect, including the modules a local `replace` points to. The vet does not read their source, so a hazard in that code reaches the Binary unchecked. Reading it would mean fetching and parsing the whole third-party graph on every build, and a hazard in a library is not always a defect: a command-line library may call `os.Exit` on a path your member never takes. Ompig therefore reports the gap instead of guessing.

When a build fuses members that require such modules, `ompig ompiglet build` prints one `note:` after the vet passes. It lists each unvetted module once as `path@version`, sorted by module path, at the version the build selects: the highest version Ompig or any member requires. When a `replace` directive supplies the module's source, the entry ends with `=> ` and the replacement module or directory. A build whose members have no external modules prints no note. Ompig and its extension SDK are not listed, because the Ompig being built is the one that is linked. The list comes from the `require` directives of the members' and workspace modules' `go.mod` files, so it can also name a module that only the members' tests use. The note follows from those `go.mod` files and the Ompig source, which are already part of the Binary's recorded build inputs, so the Binary's record needs no separate entry for it.

Review the listed modules yourself, or keep a module with hazards out of a fused member by running that member as a subprocess extension.

## Subprocess extensions

A normal Ompiglet Binary can start exact subprocess components when its plan permits them.

Advantages:

- process failure isolation;
- support for Go, Rust, Python, Node, and native programs;
- independent heartbeat and cancellation handling;
- easier source development and reload.

Costs:

- process and IPC overhead;
- explicit runtime requirements;
- more lifecycle and transport handling;
- the Binary might not contain every required runtime.

A prebuilt native subprocess component can avoid a target-side compiler. An interpreted component still needs its recorded runtime.

### TypeScript and JavaScript extensions

A Ompiglet Binary can carry TypeScript and JavaScript extensions. The build packs each Node extension's sources into one archive per Node cell and embeds the archive in the Binary. It archives a directory extension whole, including its `node_modules`, but not its `.git` directory, and it archives a single-file extension as that file alone. A symbolic link that points outside the extension directory fails the build (a link to the directory itself is kept), because the Binary would run without the code the link points to. Keep an extension's local imports and dependencies inside its directory.

The Binary does not contain Node. It runs the extension with the Node on the user's `PATH`, which must be version 22.13 or newer, the same requirement Stock Ompig has. At startup the Binary checks the archive against the digest in its embedded cell manifest and extracts it once per digest under `ompiglet-binary-cells/node/` in the Ompig configuration directory (`~/.ompig` by default). A later start reuses that directory and does not check its files again, so a file changed or deleted there stays changed until you delete the `<digest>` directory; the next start then extracts the archive again. It then plans, starts, and reloads the extension through the same Node cell path that Stock Ompig uses for the same extension. The extension registers the same tools, commands, and handlers, in the same order.

The component plan records a Node extension as a subprocess component with binary materialization and a `node` runtime requirement. The signed manifest lists the archive among its embedded files, and the signature covers its bytes.

The Binary costs what the extension costs in Stock Ompig. On an Apple M4 Pro, a Binary with one fused Go extension and one small TypeScript extension is 16,528 bytes larger than the same Binary without the TypeScript extension. Its `-p` run with the faux provider took 178 ms (median of 10), against 179 ms for Stock Ompig with `-e` and the same extension, and 34 ms for the Binary without it. Almost all of that time is Node starting the extension.

When Node is missing or too old, the Binary behaves like Stock Ompig. It reports the failed extension, names the Node version it needs, and prints the `-ne` hint. It then exits with status 1 before the session starts:

```text
Error: Failed to load extension "/Users/me/.ompig/ompiglet-binary-cells/node/<digest>/sources/greeter": Failed to load extension: reload packed Node cell packed-node:<key>: spawn:runtime_unavailable: TypeScript extensions need Node.js 22.13 or newer; node was not found on PATH
Hint: Start without extensions using "ompig -ne".
```

A Ompiglet that strips `node-extensions` cannot carry a Node extension. The build stops with the conflict described in [Stripped built-ins](#stripped-built-ins).

## External components

An external realization connects to a separately managed service. The Binary records the requirement but does not embed or operate that service.

Use an external realization for network services that have their own lifecycle, scaling, credentials, or deployment owner.

## Require all extensions to fuse

A Ompiglet can reject subprocess fallback:

```yaml
build:
  extensionRealization: fused
```

The build then requires every selected extension to be a fuse-compatible Go factory. If one extension resolves to a subprocess, the build fails and names that extension and the reason.

Ompig Standard uses this requirement. Every extension added to Ompig Standard must:

- use the public Go extension factory contract;
- work in source mode through the normal subprocess host;
- work as a fused component;
- pass the same behavior and lifecycle tests in both modes.

## Frontend members

A Ompiglet Binary can fuse one frontend member that draws the interactive mode in place of Ompig's terminal renderer (D91). Pi has no equivalent: Stock Ompig and a Ompiglet without a frontend member paint exactly as Pi does.

```yaml
slots:
  frontend:
    member: ./frontend
```

`slots.frontend.member` names a directory inside the directory of the Ompiglet file that names it. It holds a Go module whose root package exports:

```go
func Frontend() frontend.Frontend
```

The `frontend` package is `github.com/tmih06/ompig/extensions/sdk/frontend`. A frontend member depends on Ompig only through the Go SDK module; it may require other Go modules, as described in [Fused members with external Go modules](#fused-members-with-external-go-modules). It is not an extension: it has no wire protocol and runs only fused, and only the native builder fuses it. A Ompiglet may have a frontend member and no extensions, as a Ompiglet may have neither (see [Build a Binary](#build-a-binary)).

A Ompiglet that extends another inherits its frontend member, which still resolves against the base Ompiglet's directory. The child replaces it by naming its own `slots.frontend`, or removes it so Ompig's terminal renderer applies again:

```yaml
extends:
  source: local:./ompig-tern/ompiglet.yaml
  remove:
    slots: [frontend]
```

`ompig ompiglet show` lists the frontend member in its `Slots` section as `frontend  replaced(<member directory>)`, beside any stripped built-ins (see [Ompiglets](ompiglets.md#strip-built-ins)). The Binary's own embedded Ompiglet carries no `slots`, because the member is compiled into the Binary.

On every interactive start, Ompig calls the member's `Open` after the terminal enters raw mode and before the first paint. A member that returns no session leaves the terminal renderer in place. A session receives the interactive screen as retained-tree frames: transcript entries in the main region and the input dock in the dock region. Tool calls arrive as tool nodes with their arguments, status and output, and an edit's result with its unified diff; a call the user aborted has the status `cancelled`. User and assistant messages arrive as Markdown nodes with their role and Markdown source, and thinking as thinking nodes that say whether the user hid it; an assistant message is one node per part, and the part still being generated is marked as streaming, so a frontend can append what a frame adds. Every other component arrives as the terminal lines it rendered, at the width the session reports for its region. Terminal input that the session claims never reaches the editor.

The input editor arrives as an editor node with its text, its cursor in UTF-16 code units, a placeholder and whether it is sendable. Ompig draws no ANSI border around it, since the frontend frames it. Above it comes a working node while Ompig works, compacts, summarizes a branch or waits to retry: its kind, its message, and an extension's spinner frames and interval, without the current frame, so the frontend animates the spinner. Below it come the lines of its completion list and the components under it, then a footer node while Ompig's footer shows: the working directory, git branch and session name, the session's token and cost totals, the cache-hit rate, the context usage, the model with its provider and thinking level, and the extension statuses. A footer that an extension sets replaces the footer node and arrives as lines. While a component Ompig does not model replaces the editor, or an overlay that takes the keys shows, the dock arrives as lines. The session acts on the editor through `Env`: `Edit` replaces a range of its text as one undo step, `Undo` steps back through the editor's own history, and `Send` submits a prompt through the path Enter takes, so commands, multi-line prompts and steering behave as typed. Ompig runs these on its own loop in the order they arrive, ignores an edit computed against text that has changed since, and reports the editor sendable only once its input loop runs and while the editor has the keys.

A selector or dialog that takes the editor's place, such as `/model`, `/resume`, `/tree`, `/fork`, `/login`, the thinking level picker or an extension's select or confirm dialog, arrives in its place as a selector node: its title and description, the search text, the items shown in the order the arrow keys step through them with a label, a detail, a check for the current or enabled entry and a tree depth, the selected item, its tabs (such as the scope of the model and session lists), a status message, a loading flag and a confirm question. `/settings` arrives as a settings node: each setting with its value, the values it cycles through, and whether it opens a submenu, which then arrives as a selector under the same id. Keys still reach the selector as typed. A session acts on one with `Env.Act`: highlight, choose, dismiss, filter, switch tab or set a value. Ompig performs each action as the keys it stands for, so the selector behaves exactly as with the keyboard. Overlays, such as an extension's custom component shown as an overlay, arrive in an overlay region as overlay nodes with their lines, width, anchor and whether they take the keys. Components Ompig does not model, such as an extension's input or editor dialog, the login dialog, and the session rename and tree label fields, arrive as lines.

The first frame also names the session file, the file the session persists to, or none for a session that is not persisted (`--no-session`). A later frame names it again only when it changes, such as after `/new`, `/resume`, `/fork` or `/clone`. A frontend can pass it on, for example to let another program read the conversation.

A session is also the terminal that Ompig's interactive mode reports its program status to: working while a run or compaction goes on, blocked while a dialog or a login waits for the user, then done, error or idle (the OSC 7501 Program Status Protocol of Pi 1.4.0). A session that implements `frontend.ProgramStatusSession` receives each change through `ProgramStatus` as the report Pi writes to a supporting terminal, including the report that clears the status. Ompig calls it on its owner loop in the order the status changed, so it must return without blocking. It clears the status before `Suspend` and `Close` and reports the latest one again after `Resume`. A session without the method receives nothing. Either way, while a session may draw, Ompig writes neither the OSC 7501 support query nor a report to the terminal itself.

When a session asks to fall back, or fails to draw a frame, Ompig closes it, repaints with its configured terminal renderer and shows a warning. While a session draws, Ompig refuses to switch the TUI mode. `--print`, JSON and RPC modes never consult the member.

The member's source is part of the Binary's identity: its tree digest is a `frontend` input in the resolution record.

## Stripped built-ins

A Ompiglet's `strip` list (D92, see [Ompiglets](ompiglets.md#strip-built-ins)) carries into its Binary. The native builder passes one Go build tag per stripped built-in extension, model API and compile-out feature (`ompig_strip_<id>`, for example `ompig_strip_mcp` or `ompig_strip_node_extensions`), so the Binary does not link that code; the build prints the tags it uses. The baked Ompiglet keeps the list, so the Binary also disables the runtime-only entries (tools, commands, `themes`, `skills`, `prompt-templates`) at startup. A Ompiglet cannot compile out an extension runtime its own extensions run on: a Node or Python extension, or a Go or Rust extension the Binary builds at runtime, fails the build with the strip entry named. A Ompiglet whose strip list removes every built-in tool and `/quit` together does not build either (see [Ompiglets](ompiglets.md#strip-built-ins)).

Stripping cuts size and startup work. On darwin/arm64 a Ompiglet Binary with one fused Go extension is 62.2 MB; stripping MCP, codemode, llama.cpp, tool-search and three provider APIs makes it 54.2 MB, and stripping every binary ID makes it 31.7 MB.

The Binary record lists each stripped ID with its kind and disposition in its `strip` field, so the strip list is part of the Binary's identity and record digest. A Binary without a strip list has no `strip` field. `ompig ompiglet show` prints the list under each Binary in a `Slots` section (`extensions.mcp  stripped(binary)`, `tools.bash  stripped(runtime)`), and `ompig ompiglet show --record <path>` prints it for one record.

A keep-mode list ([Ompiglets](ompiglets.md#keep-mode)) reaches the Binary expanded: the build compiles out every built-in of that list the Ompiglet does not keep, and the `strip` field lists each one. The record states the intent too: `stripKeep` lists each keep-mode list with the IDs it keeps (`[{"kind": "extension", "ids": []}]`), and `ompig ompiglet show` prints it above the expanded rows (`extensions  keep([])`). A deny-mode Ompiglet has no `stripKeep` field. The baked Ompiglet keeps the keep lists, not their expansion, and the Binary expands them again at startup against its own table, which is the table it was built from.

Every Binary record also keeps `stripTable`, the strip table of the Ompig that built it, one list of IDs per kind. A later Ompig compares its own table with it and prints the [strip delta report](ompiglets.md#strip-delta-report) on `ompig ompiglet build` and `ompig ompiglet show`: the new built-ins a keep-mode list leaves out and the ones a deny-mode list now includes.

## Accept overlay Ompiglets

A Ompiglet Binary runs its built-in Ompiglet and refuses `--ompiglet`, `OMPIG_OMPIGLET_PATH` and `OMPIG_OMPIGLET_NAME` (D202). A base Ompiglet can declare that its Binary accepts one more Ompiglet, selected at launch, as an overlay. Set `build.allowOverlays` in the Ompiglet you build:

```yaml
name: ompig-base
build:
  allowOverlays: true
strip:
  extensions: [mcp]
```

The default is `false`. Like all of `build`, the setting belongs to the Ompiglet you build: a Ompiglet that `extends` a base with `allowOverlays: true` does not inherit it. The builder keeps the setting in the Ompiglet it bakes, so it is part of the digest the Binary's record and signed manifest name; a signed Binary whose setting was changed refuses to start.

A Binary that accepts overlays takes the same selectors stock Ompig does and resolves them the same way, with the same project trust, user and system Ompiglet directories, and Package install flow (lifecycle scripts stay off unless you allow them):

```bash
./ompig-base --ompiglet ./extra.yaml
OMPIG_OMPIGLET_PATH=./extra.yaml ./ompig-base
```

The overlay is an ordinary Ompiglet file with its own `name`; its own `extends` chain works as usual. Ompig merges it onto the Binary's Ompiglet as `extends` merges a child onto its base. Its extensions and skills follow the Binary's, its Packages and `strip` lists add, `model` and `description` replace, and the Binary keeps its name for `/ompiglet` and for strip messages. `/ompiglet` also prints `Overlay of: <Binary>`. A Binary starts exactly as before when you select no Ompiglet: it installs nothing and resolves nothing.

The Binary's own contents stay what was verified. An overlay can add and narrow; it cannot change what the Binary baked. Each refusal names the field and the Binary and stops the start; the Binary never falls back to its own Ompiglet when an overlay fails:

| The overlay | Result |
|---|---|
| adds extensions, skills or Packages with origins; sets `model`; strips more built-ins; narrows `tools` | Allowed. Its extensions resolve at run time as under stock Ompig, so a `local:` origin runs as a subprocess extension. |
| names an extension or skill the Binary carries, or removes one through its own `extends.remove` | Refused: `extensions[<name>]: Ompiglet Binary "<name>" already carries one with this name; an overlay cannot replace it`. Baked members are fused or embedded and checked at startup. |
| sets `slots.frontend` | Refused: a frontend member is Go code compiled into the Binary. |
| sets `systemPrompt` | Refused, as `--system-prompt` is: use `--append-system-prompt`. |
| widens `tools`, `discovery`, `agentEnv` or an extension's `tools` | Refused. An overlay has no `allowWiden`; rebuild the Binary's Ompiglet to widen. |
| re-enables a stripped built-in or slot | Refused: `strip.tools: "bash" is stripped by Ompiglet Binary "<name>"; an overlay cannot re-enable it`. |

Without `allowOverlays`, the Binary refuses the selector and names itself:

```text
ompig: this Ompiglet Binary (ompig-base) runs only its built-in Ompiglet and does not accept --ompiglet overlays (nor OMPIG_OMPIGLET_PATH or OMPIG_OMPIGLET_NAME); build its Ompiglet with build.allowOverlays: true to accept them, or run the Ompiglet with stock ompig (ompig --ompiglet <name|path>)
```

Accepting overlays changes neither the Binary's verification nor its update ownership: it still verifies its signature, Ompiglet and component closure before anything else, and `ompig update` still refuses to modify an immutable Binary. The overlay is input you supplied, outside that closure. A Binary you ship to others with `allowOverlays` runs whatever Ompiglet its user selects, within the rules above.

## Startup verification

A Ompiglet Binary verifies its embedded Ompiglet, component closure, and optional Ed25519 signature before command dispatch. A signed Binary refuses to run when any executable or signature byte changes, within the limit that the verification cache below describes. An unsigned Binary reports unsigned and runs unless the local trust policy requires every Ompiglet Binary to carry a trusted signature.

Signature verification reads and hashes the whole executable. For a 63 MB darwin/arm64 Binary on an Apple M4 Pro the hash takes about 25 ms. An unsigned Binary has no signature block to check and skips the hash.

Signature verification reads and hashes the whole executable at every start. For a 62 MB darwin/arm64 Binary on an Apple M4 Pro this adds about 26 ms. An unsigned Binary has no signature block to check and skips the hash.

Startup does not silently:

- install a Package;
- download a missing component;
- substitute a same-named executable from `PATH`;
- log in to a provider;
- weaken an environment requirement;
- replace a failed fused extension with a subprocess.

A missing or tampered requirement causes a clear failure.

### Verification cache

A signed Binary hashes its executable only when its file changed since its last passing start (D18). After every startup check passes, the Binary records the check in `state/ompiglet-verify/verified.json` under the Ompig config root (`$OMPIG_HOME`, else `$XDG_CONFIG_HOME/ompig`, else `~/.ompig`). The entry holds:

- the Binary's path with symbolic links resolved;
- its file identity: device, inode, size, modification time, and status-change time; on Windows the volume serial number, file index, size, last write time, and change time;
- the SHA-256 of its signature block;
- the trust policy that applied: `require on`, the trusted, revoked, and embedded keys.

At the next start the Binary still reads its signature block and checks the signature, the signed size, the signer under the current trust policy, and the signed manifest. When every recorded field matches, it skips the hash of its executable bytes. Any difference causes a full check: a new path, a touched or rewritten file, a file replaced by another one, a changed trust store or `require on`. A missing, corrupt, unreadable, or not owner-only cache also causes a full check. An entry never vouches for another path.

- A failed check is never recorded. A check is recorded only after the signed manifest matches the Binary.
- The cache file is readable and writable by its owner only and is replaced atomically. It keeps one entry per Binary path, at most 8, and drops the oldest entry first.
- `ompig verify` and `ompig ompiglet verify` never read the cache. They always hash the whole file.
- When Ompig cannot write the cache, the Binary still starts and prints `ompig: warning: write Ompiglet verification cache: ...` on stderr. The next start checks the Binary in full again. On a config root this user cannot write (a read-only file system or a permission denial) the warning is not printed, because that is a deployment choice; `ompig diagnose` reports the read-only config root and that every start checks the Binary in full (D157).

On an Apple M4 Pro (median of 10, `-p` with the faux provider), Stock Ompig took 31.1 ms. The 63 MB signed Binary took 56.1 ms with a full check at every start, 59.3 ms for a start that also writes the cache, and 33.3 ms for a start with a matching cache entry.

What the cache keeps and what it gives up:

- It keeps detection of every change made through a file system that records a status-change time (on Windows, NTFS's change time). Writing to the Binary changes that time even when the writer restores the modification time, and replacing the Binary changes its inode, so the next start hashes the new bytes and refuses a change.
- It gives up detection, at start, of corruption that keeps the size and every recorded time, such as a storage fault below the file system, and of an in-place write that restores the modification time on a file system without a status-change time, such as FAT. On Windows a writer can also set the change time back through the file information API; such a writer is not detected at start. `ompig ompiglet verify` still detects all of these.
- The startup check is a self-check. Anyone who can rewrite the Binary can also rewrite the code that checks it, with or without a cache. The check does not stop an attacker who can write the Binary. Protect the Binary with file permissions, and run `ompig ompiglet verify` on a Binary you did not build before you run it.

## Failure and recovery

Fusion and subprocess execution have different physical failure boundaries, but they use the same extension behavior contract.

For subprocess components, Ompig provides bounded heartbeat, cancellation, crash detection, quarantine, reload, and replacement behavior. A failed replacement does not remove the last working extension set.

Ompig does not automatically replay an interrupted tool or command. A replay could duplicate file changes, messages, or network actions. The caller decides whether an operation is safe to retry.

A fused extension cannot lose a process socket because it runs inside Ompig, but it can still panic, block, or misuse resources. Fused extensions therefore require lifecycle, shutdown, and leak verification before release.

See [Extensions](/docs/latest/extensions) for the complete tradeoff and recovery model.

## Portability

Build one Binary per target:

```yaml
build:
  targets:
    - linux/amd64
    - darwin/arm64
    - windows/amd64
```

Do not claim target support until the artifact passes native verification on that target.

A Binary built on one target does not make an external service, secret, or interpreter portable. The record keeps these requirements visible.

## Source mode versus Binary mode

| Need | Use |
|---|---|
| Edit and reload extension source | Host Ompiglet source mode |
| Inspect a composition before release | Host Ompiglet source mode |
| Hand off one executable | Ompiglet Binary |
| Require one native all-fused application | Ompiglet with `extensionRealization: fused` |
| Carry a complete operating environment | Ompiglet Image when that producer is available |

## Related documentation

- [Ompiglets](/docs/latest/ompiglets)
- [Extensions](/docs/latest/extensions)
- [Derivative harnesses](/docs/latest/derivative-harnesses)
- [Packages](/docs/latest/packages)
