# Quickstart

Ompig is under active development. Install it on macOS or Linux with `curl -fsSL https://ompig.dev/install.sh | sh`, on any supported platform with `npm install -g @ompig/ompig` or `go install github.com/tmih06/ompig/cmd/ompig@latest`, or download an archive from [GitHub Releases](https://github.com/tmih06/ompig/releases). Windows support is a preview. To build from source instead, follow the steps below. Pi 1.4.0 documents its own install routes in [Pi's quickstart](https://github.com/earendil-works/pi/blob/v1.2.0/packages/coding-agent/docs/quickstart.md).

If you have not used a terminal before, start with [Your first ten minutes](/docs/latest/first-run). It explains each step.

## Install with the install script

On macOS and Linux, the install script downloads the newest published release, checks it against the release's `SHA256SUMS`, and installs `ompig` in `~/.local/bin`:

```bash
curl -fsSL https://ompig.dev/install.sh | sh
ompig --version
```

When `~/.local/bin` is not on `PATH`, the script prints the `export PATH=...` line to run. [Install troubleshooting](/docs/latest/install-troubleshooting) lists the script's options, for example `OMPIG_VERSION` to install a given release.

## Requirements

You need these tools only to build Ompig from source. Build the `ompig` executable with Go 1.27.3. Go 1.27 release binaries for macOS require macOS 13 or later.

The complete verification suite also uses:

- Git;
- Node.js 24.22.1 and npm 12.1.0;
- Python 3.12;
- Rust 1.97.1;
- tmux on Unix.

You do not need every verification tool to run a previously built executable.

## Build from source

Clone the repository and build Ompig:

```bash
git clone https://github.com/tmih06/ompig.git
cd Ompig
go build -o bin/ompig ./cmd/ompig
./bin/ompig --version
```

Do not install Ompig by running an upstream Pi npm command. The package `@earendil-works/pi-coding-agent` installs Pi, not Ompig.

## Install with npm

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
(`@ompig/ompig-<os>-<cpu>`, for macOS, Linux and Windows on x64 and arm64) as
an optional dependency, and Node.js runs only a small launcher. Do not install
with `--omit=optional` or `--no-optional`, which leaves the binary out. Update
with `npm update -g @ompig/ompig` or `ompig update`, which runs
`npm install -g @ompig/ompig@<version>` for the release it finds, and uninstall
with `npm uninstall -g @ompig/ompig`.

## Install with Go

Go 1.26 or newer installs the command without cloning the repository. Go downloads Ompig's source and its dependencies and compiles them on your computer, which can take several minutes the first time:

```bash
go install github.com/tmih06/ompig/cmd/ompig@latest
```

Use an exact version for a reproducible installation:

```bash
go install github.com/tmih06/ompig/cmd/ompig@v0.5.0
```

Go writes the `ompig` executable to `GOBIN`. When `GOBIN` is empty, Go uses the
`bin` directory below the first path in `GOPATH`: `$(go env GOPATH)/bin`, which
is `~/go/bin` by default (`%USERPROFILE%\go\bin` on Windows). With an older Go
1.21 or later and the default `GOTOOLCHAIN=auto`, Go downloads a new enough
toolchain automatically. Add that directory to `PATH`, then inspect the
installed module identity:

```bash
ompig version
go version -m "$(command -v ompig)"
```

Update a Go installation by running `go install` again with a newer exact
version or with `@latest`. Ompig does not treat a Go-installed executable as a
standalone release download, so `ompig update` refuses to replace it without a
matching installation receipt.

Explicit `ompig update` version checks retry transient transport failures and transient HTTP statuses at most twice within one ten-second budget. Startup checks do not retry. Signature and manifest errors stop the update. Native standalone updates reject concurrent replacement of the same executable and hold the installation lock until receipt commit or rollback completes (D39). The `<executable>.update.lock` sidecar exists only while an update runs; the update removes it when it ends.

Each release tags the root module (`v0.5.0`) and the nested Go extension SDK module (`extensions/sdk/v0.5.0`) on the same commit; `go install` needs both.

## Start Ompig

Run Ompig in the folder of the project you want it to work on:

```bash
cd /path/to/project
ompig
```

A source build is at `bin/ompig` in the Ompig checkout. Run it by its full path, for example `/path/to/Ompig/bin/ompig`, or add that folder to `PATH`.

Ompig stores user state below `~/.ompig`. Use `OMPIG_HOME` only when you need an isolated configuration root.

On an interactive start in the default agent directory that holds no `settings.json` yet, which includes every fresh `OMPIG_HOME`, Ompig shows a first-time setup screen: theme, sprite and usage-analytics choices. `Esc` skips it and saves nothing. A custom agent directory (`OMPIG_CODING_AGENT_DIR`) and print, JSON and RPC modes skip it (`coding/cli/extensions.go:93-95`). Pi 1.4 shows its screen (theme and usage-analytics choices) only when all of these hold (`startup-ui.ts` `shouldRunFirstTimeSetup`): the binary is Pi's own distribution (package `@earendil-works/pi-coding-agent`, app `pi`, config directory `.pi`), `PI_EXPERIMENTAL=1` is set, `PI_CODING_AGENT_DIR` is not set, and `~/.pi/agent/settings.json` does not exist yet. This is divergence D88. Change the theme later with `/settings` and the ompig with `/sprite`. [Your first ten minutes](/docs/latest/first-run) shows the screen.

## Authenticate a provider

List login targets available in the current binary:

```bash
ompig login --list
```

Start an available OAuth login:

```bash
ompig login github-copilot
```

For an API-key provider, set the provider's documented environment variable:

```bash
export OPENAI_API_KEY=...
ompig --model openai/gpt-4.1
```

Use `ompig --help` and `ompig login --help` as the authority for your binary.

## Start a first session

Type a request and press Enter:

```text
Summarize this repository and tell me how to run its checks.
```

Stock Ompig gives the model its normal built-in coding tools unless the command line, settings, or an active Ompiglet narrows them.

## Use print mode

Run one prompt without the interactive TUI:

```bash
ompig -p "Summarize this repository"
```

Select a model explicitly when needed:

```bash
ompig --model github-copilot/gpt-5-mini -p "Find the test entry points"
```

## Run Ompig Standard

Ompig Standard is not part of bare Stock Ompig startup. Select it explicitly from a source checkout:

```bash
ompig --ompiglet ompiglets/standard/ompig-standard.yaml
```

The current Standard composition provides its Ompig login, `/sprite`, `/runner`, and `/ompig-runner` through ordinary extension Resources.

## Run your own Ompiglet

Validate and run one agent application:

```bash
ompig ompiglet validate ./agents/reviewer.yaml
ompig --ompiglet ./agents/reviewer.yaml
```

Register portable Ompiglet source when you want to invoke it by name:

```bash
ompig ompiglet add ./agents/reviewer.yaml
ompig --ompiglet reviewer
```

See [Ompiglets](/docs/latest/ompiglets).

## Build a Ompiglet Binary

```bash
ompig ompiglet build ./agents/reviewer.yaml \
  --format binary \
  --out ./ompig-reviewer
./ompig-reviewer
```

A Ompiglet Binary is a target-native executable for one fixed Ompiglet composition. It can still have explicit external requirements. The build needs Go, or Docker or Podman for the container builder. A `ompig` that you built yourself also needs a Ompig source checkout named by `OMPIG_SOURCE_ROOT`; a published release fetches its own source. See [Ompiglet Binaries](/docs/latest/ompiglet-binaries).

## Add an extension

A Go extension needs the Go toolchain. `ompig setup` shows whether `go` is found, and `ompig setup go` installs it. Create and validate a Go factory:

```bash
ompig extension init ./review --lang go
ompig install ./review --validate-only --json
```

Run it for one session:

```bash
ompig -e ./review
```

Use a Ompiglet when the extension belongs to a named agent application. [Your first extension](/docs/latest/first-extension) walks through each step.

## Verify a source checkout

Prepare the exact Pi comparator and source mirror:

```bash
make upstream-mirror
```

Run the primary gate:

```bash
make check
```

Use the toolchain versions declared by the repository. A green build does not replace parity review, vulnerability review, artifact inventory, or release authorization.

## Next steps

- [Using Ompig](/docs/latest/usage)
- [Ompig concepts](/docs/latest/concepts)
- [Extensions](/docs/latest/extensions)
- [Packages](/docs/latest/packages)
- [Ompiglets](/docs/latest/ompiglets)
- [Build a derivative harness](/docs/latest/derivative-harnesses)
