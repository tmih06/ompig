# Install and use Ompig

Use this page to get from a fresh binary to a working session. Return to the
[docs router](README.md) for focused tasks.

## 1. Build or install the binary

On macOS, Linux or Android with Termux (arm64 only), install the latest release with
`curl -fsSL https://ompig.dev/install.sh | sh`; in Termux it installs the
`android-arm64` release into `$PREFIX/bin`. Or download an archive from
[GitHub Releases](https://github.com/tmih06/ompig/releases). Windows
support is a preview. From a source checkout, build it with Go 1.27.3. Go 1.27 release binaries for macOS require macOS 13 or later:

```bash
go build -o bin/ompig ./cmd/ompig
./bin/ompig --version
```

Install the command with npm (Node.js 18 or newer):

```bash
npm install -g @ompig/ompig
ompig --version
```

Or run it once without installing:

```bash
npx @ompig/ompig --version
```

`ompig` itself is the native binary: npm installs the matching platform package
(`@ompig/ompig-<os>-<cpu>`, for macOS, Linux and Windows on x64 and arm64, and Android on arm64 for
Termux) as
an optional dependency, and Node.js runs only a small launcher. Do not install
with `--omit=optional` or `--no-optional`, which leaves the binary out. Update
with `npm update -g @ompig/ompig` and uninstall with
`npm uninstall -g @ompig/ompig`. `ompig update` does not replace an
npm-installed binary; update through npm.

Install the command with Go 1.26 or newer, without a source checkout:

```bash
go install github.com/tmih06/ompig/cmd/ompig@latest
```

Use an exact version for a reproducible installation:

```bash
go install github.com/tmih06/ompig/cmd/ompig@v0.2.0
```

Go writes the `ompig` executable to `GOBIN`, or to the `bin` directory below the
first `GOPATH` entry when `GOBIN` is empty: `$(go env GOPATH)/bin`, which is
`~/go/bin` by default (`%USERPROFILE%\go\bin` on Windows). Add that directory to
`PATH`. With an older Go 1.21 or later and the default `GOTOOLCHAIN=auto`, Go
downloads a new enough toolchain automatically. Update this installation by
running `go install` again. Ompig does not treat a Go-installed executable as a
standalone release download, so `ompig update` refuses to replace it without a
matching installation receipt.

Each release tags the root module (`v0.2.0`) and the nested Go extension SDK
module (`extensions/sdk/v0.2.0`) on the same commit; `go install` needs both.

After the project publishes a release, you can also download an artifact from
its release page. Verify the published checksum before you place `ompig` on
`PATH`.

Ompig stores user state under `~/.ompig`. Set `OMPIG_HOME` only for an isolated config
root, tests, or CI.

### Change what the install script does

`install.sh` reads these environment variables. Set them on the `sh` side of the pipe, for example `curl -fsSL https://ompig.dev/install.sh | OMPIG_VERSION=0.5.0 sh`.

| Variable | Meaning |
|---|---|
| `OMPIG_VERSION` | Install this release (for example `0.5.0`) instead of the latest. |
| `OMPIG_INSTALL_DIR` | Directory that receives `ompig`. The default is `$HOME/.local/bin`, or `$PREFIX/bin` in Termux. |
| `OMPIG_HOME` | Ompig's settings directory, where the install receipt goes. The default is `~/.ompig`. |
| `OMPIG_API_BASE` | The API that names the latest release. The default is `https://ompig.dev/api`. |
| `OMPIG_LATEST_RELEASE_URL` | The page that redirects to the latest release, used when the API does not answer. |
| `OMPIG_DOWNLOAD_BASE` | The release download root. The default is the GitHub Releases download URL of this repository. |
| `OMPIG_UPDATE_URL` | The update manifest that `ompig update` uses. The default is the latest release's `update.json`. |

The script verifies the archive against the release's `SHA256SUMS` before it writes anything to the install directory, whichever mirror these variables select.

### macOS Gatekeeper after a manual download

A browser download can add the local `com.apple.quarantine` attribute to a Ompig
archive. First verify the published SHA-256 checksum. For a verified Ompig binary,
run:

```bash
xattr -p com.apple.quarantine /path/to/ompig
xattr -d com.apple.quarantine /path/to/ompig
/path/to/ompig --version
```

Do not recursively remove quarantine attributes from a broad directory.

## 2. Authenticate a model provider

```bash
ompig login github-copilot
```

API-key providers may use their documented environment variable:

```bash
export OPENAI_API_KEY=...
ompig --model openai/gpt-4.1
```

See [Providers](providers.md). Product extensions may contribute additional
login targets, such as a platform host, through the same `ompig login` surface.
Use `ompig login --help`/`ompig --help` for the targets available in your binary.

## 3. Install packages

Raw Ompig accepts local, git, npm, and HTTP package sources:

```bash
ompig install ./my-package
ompig install git:https://github.com/example/my-package.git
ompig install npm:@example/package
```

An installed product extension may add another resolver scheme, such as a
private marketplace. Its authentication and catalog commands are product
features, not assumptions in raw Ompig.

## 4. Run Ompig

```bash
ompig                              # interactive TUI
ompig -p "summarize this repo"     # one-shot prompt
ompig --model github-copilot/gpt-5-mini
```

## 5. Install and run a ompiglet

Install and validate a registerable Ompiglet, then invoke it by name:

```bash
ompig ompiglet add ./ompiglets/my-agent.yaml
ompig ompiglet validate my-agent
ompig --ompiglet my-agent
```

A Ompiglet with relative `local:` extension or skill origins is source-bound.
Run it directly instead of registering it:

```bash
ompig ompiglet validate ./ompiglets/my-agent.yaml
ompig --ompiglet ./ompiglets/my-agent.yaml
```

See [Ompiglets](ompiglets.md) for schema and resolution.

## 6. Build an explicit Ompiglet entry point or Binary

A source script is a thin Host Ompiglet entry point. It is not managed by Ompig and
creates no Ompig state or record:

```bash
ompig ompiglet build my-agent --format script --out ~/.local/bin/my-agent
my-agent
```

On Windows the script is a cmd.exe batch file, so give it a `.cmd` name, for
example `--out my-agent.cmd` (D69).

A Ompiglet Binary is a target-native executable for one immutable Ompiglet
composition:

```bash
ompig ompiglet build my-agent --format binary --out ./ompig-my-agent
./ompig-my-agent
```

The Ompiglet artifact lifecycle lives under `ompig ompiglet`.

A Ompiglet Binary may fuse compatible Go components and host exact subprocess
components through Ompig's ordinary subprocess extension host. Its registered
Ompiglet release records where each component comes from: binary, release
closure, required `agentEnv`, or external mapping. Subprocess use does not create
a different binary kind.

Ompiglet Image is the OCI carrier when the Ompiglet needs an environment-complete
runtime. Artifact startup never silently installs, pulls, logs in, or substitutes
an origin.

## 7. Set up an extension toolchain only when authoring/building

You need a language toolchain to author a source extension or build its release
closure. A consumer needs no compiler for a prebuilt native component; interpreted
subprocess components still need the exact runtime recorded by the Ompiglet
release or supplied by `agentEnv`.

Check before installing anything:

```bash
go version
cargo --version
python3 --version
node --version
```

Prefer a version manager already present on the machine. Otherwise use the OS
package manager or the language's official per-user installer, with the user's
consent. Do not remove or replace an existing system toolchain without asking.

For a Go extension, the required version is the `go` directive in the staged SDK
module. `ompig extension init` stages an offline-resolvable project and
`ompig install <path> --validate-only --json` validates it.

## Troubleshooting

| Symptom | Likely cause | Action |
|---|---|---|
| `ompig` not found | binary directory absent from `PATH` | add the install directory |
| no selectable models | no provider credential | run `ompig login <provider>` or set its API key env var |
| contributed source scheme is unknown | product resolver is not installed/loaded | install or run the product distribution that owns it |
| ompiglet origin does not resolve | missing local source/package/product resolver | run `ompig ompiglet validate <name>` and fix the reported origin |
| source extension fails on first use | missing language toolchain | install it with consent or use a prebuilt Ompiglet release/Image |
| Ompiglet artifact cannot start a component | registered release/environment/external requirement missing | rebuild with a supported format or follow the reported mapping remediation; do not substitute PATH content |

## Offline docs

```bash
ompig docs list
ompig docs show install
ompig docs path
```
