# Run Ompig on Android with Termux

Ompig runs on Android through [Termux](https://termux.dev/), a terminal emulator and Linux environment. Text input, file tools, and shell commands are supported. Ompig copies text to and pastes text from the Android clipboard with Termux:API. Clipboard image paste is not supported.

Termux runs Ompig's `android-arm64` release binary: a position independent executable linked to Android's C library. Do not use the `linux-arm64` binary in Termux; Termux's loader cannot start it (`has unexpected e_type: 2`). Android on other CPUs (32-bit Arm, x86) has no release binary. No hosted CI runner executes Ompig on Android, so treat Termux support as experimental until a device run is recorded in the release evidence.

## Before you begin

Install Termux from [GitHub, F-Droid or Google Play](https://github.com/termux/termux-app#installation). Android does not let an app run files from its own data directory. Termux works around that with `termux-exec`, which starts each program through `/system/bin/linker64`; Ompig does the same for the programs it starts, because its Go runtime bypasses `termux-exec`.

[Termux:API](https://github.com/termux/termux-api#installation) is optional. Install it only when you want Ompig to copy or paste Android clipboard text, or when shell commands need Android device APIs.

## Install Ompig

1. Update Termux packages:

   ```bash
   pkg update && pkg upgrade
   ```

2. Install curl and Git:

   ```bash
   pkg install curl git
   ```

3. Install Ompig. The installer detects Termux, downloads the `android-arm64` release, verifies its SHA-256, and installs `ompig` into `$PREFIX/bin`:

   ```bash
   curl -fsSL https://ompig.dev/install.sh | sh
   ```

   To install with npm instead, install Node.js and then Ompig. Node.js reports its platform as `android` in Termux, so npm selects `@ompig/ompig-android-arm64`:

   ```bash
   pkg install nodejs
   npm install -g @ompig/ompig
   ```

4. Verify the installation:

   ```bash
   ompig --version
   ```

5. Open the folder you want to work in and start Ompig:

   ```bash
   cd /path/to/working-folder
   ompig
   ```

Continue with the main [Quickstart](quickstart.md#authenticate-a-provider) to connect a model and run your first task.

Update a script installation with `ompig update`. Update an npm installation with `npm update -g @ompig/ompig`.

## Find ripgrep and fd

Ompig does not download `rg` or `fd` on Android, because the Linux release binaries it would fetch do not run on Android's C library. It tells you to install them from Termux:

```bash
pkg install ripgrep fd
```

## Access Android shared storage

Termux cannot access shared Android storage until you grant permission. Run this once:

```bash
termux-setup-storage
```

After approval, Android shared storage is available under `/storage/emulated/0` and through the links Termux creates under `~/storage/`.

Only grant this permission when Ompig should be able to access those files. Commands and tools running in Termux use the same storage permissions as the Termux process.

## Use clipboard commands

Ompig uses `termux-clipboard-set` to copy text and `termux-clipboard-get` to paste it. Shell commands can use `termux-clipboard-set` and `termux-clipboard-get` directly. Install the Termux:API app and its command-line package:

```bash
pkg install termux-api
```

Verify the integration:

```bash
printf 'Ompig clipboard test' | termux-clipboard-set
termux-clipboard-get
```

The second command should print `Ompig clipboard test`.

The Termux clipboard API supports text only. Ompig's clipboard-paste shortcut (`Ctrl+V`) reads text through `termux-clipboard-get` and attaches no clipboard images. A failed copy reports that the Termux:API app and `termux-api` package are required.

## Networking, temporary files, and programs Ompig starts

The Android build links Android's C library, so it resolves names through Android and trusts Android's system certificates. In Termux it also sets two variables when you have not: `SSL_CERT_FILE` to `$PREFIX/etc/tls/cert.pem`, and `TMPDIR` to `$PREFIX/tmp`. Commands Ompig starts inherit them. Set either variable yourself to use another value.

When Termux starts Ompig through `/system/bin/linker64`, Ompig starts each program below Termux's data directory the same way. It runs a script through its interpreter and reads `/bin` and `/usr/bin` in a `#!` line as `$PREFIX/bin`. `ompig update` and Ompig's own re-launches find Ompig's path through `TERMUX_EXEC__PROC_SELF_EXE`, because `/proc/self/exe` is the linker in such a process.

## Add Termux-specific instructions

Ompig detects that it is running in Termux, but it cannot infer how you want it to interact with Android. Add only the environment details relevant to your work to `~/.ompig/agent/AGENTS.md`:

````markdown
# Termux environment

- Ompig runs in Termux on Android.
- Shared Android storage is under `/storage/emulated/0`.
- Open URLs with `termux-open-url "https://example.com"`.
- Open files with `termux-open <path>`.
- Do not access shared storage unless the task requires it.
````

Run `/reload` after changing the file during an active session.

### Check that Ompig read the file

This part of the page was run on Linux, because it does not depend on Android: Ompig reads `AGENTS.md` the same way everywhere. The model was a scripted test server. Save the block above as `~/.ompig/agent/AGENTS.md`, start `ompig`, and type `/reload`. Ompig lists the file it loaded and confirms:

```text
[Context]
  ~/.ompig/agent/AGENTS.md

 Reloaded keybindings, extensions, skills, prompts, themes, and context files
```

Then send any message. The request that the model server received began with the system prompt, and after the `<docs>` block it held:

```text
<project_context>
Project-specific instructions and guidelines:

<project_instructions path="/home/you/.ompig/agent/AGENTS.md">
# Termux environment

- Ompig runs in Termux on Android.
- Shared Android storage is under `/storage/emulated/0`.
...
```

The same text is the `project_context` entry of the system message that `ompig --mode json` prints first (see [JSON event stream mode](/docs/latest/json#worked-example)). A file created or edited while a session runs was picked up by `/reload` with no restart. If `/reload` does not list the file, check that it is in Ompig's agent directory (run `echo $HOME` and look in `.ompig/agent` below it, unless `OMPIG_HOME` is set).

### What was not run on Android

Nothing else on this page was run on an Android device. The steps that depend on Termux (the `termux-clipboard-*` commands, `termux-setup-storage`, the `android-arm64` release, `pkg` installs, the linker behaviour described under [Networking, temporary files, and programs Ompig starts](#networking-temporary-files-and-programs-ompig-starts)) are written from the Termux and Ompig source and release notes. Check them on your device. The `android-arm64` binary is linked against Android's C library, so a build on another system does not stand in for it.

## Build Ompiglets

`ompig ompiglet build` uses Termux's own toolchains. `ompig setup go` cannot download Go for Android, so install Go with `pkg`:

```bash
pkg install golang clang
```

Install Rust only when a Ompiglet has Rust extensions:

```bash
pkg install rust
```

`ompig setup` prints which build toolchains Ompig finds.

- `--format script` writes a launcher script.
- `--format binary` builds a native Android Ompiglet Binary. A release Ompig fetches its own source on the first build, so that build needs network access and takes a few minutes. Later builds reuse the cache.
- A Ompiglet Binary embeds Go and Rust extensions, and carries TypeScript and JavaScript extensions, which run with the Node on `PATH`. A Ompiglet with only Python extensions builds only as a script. A Ompiglet with no extensions builds a Binary of Ompig's own parts with the Ompiglet baked in.
- The container builder does not run in Termux. Termux has no Podman or Docker, and the container builder builds only Linux targets, not the Android host.

Keep about 2 GB free for the first Binary build. Go's module and build caches and Ompig's build cache use about 1.3 GB.

## Build from source

Termux's Go toolchain builds Ompig for Android with cgo:

```bash
pkg install git golang
git clone https://github.com/tmih06/ompig.git
cd Ompig
go build -o "$PREFIX/bin/ompig" ./cmd/ompig
ompig --version
```

Go 1.27.3 builds Ompig; it is the release `go.mod` names on its `toolchain` line and the release every Ompig build, test and image uses. The modules retain Go 1.26 as their language floor. If the Termux package is older than Go 1.27.3, `GOTOOLCHAIN=auto` asks for a `go1.27.3.android-arm64` toolchain that Go does not publish. Set `GOTOOLCHAIN=local` to build with the installed Go, which satisfies the 1.26 floor, or update the package.

## Troubleshooting

### Ompig is not found after installation

The script installer writes `$PREFIX/bin/ompig`, which is on Termux's `PATH`. For an npm installation, open a new Termux shell and run:

```bash
npm prefix -g
command -v ompig
```

Confirm that the global npm binary directory is on `PATH`, then reinstall Ompig if the package is missing. Do not install with `--omit=optional`, which leaves the native binary out.

### The installer reports an unsupported CPU

Run `uname -m`. Ompig publishes arm64 only; `aarch64` is that CPU. A 32-bit Termux (`armv7l` or `armv8l`) or an x86 Android device has no release binary. Build from source instead.

### Clipboard integration fails

Confirm that you installed both components:

1. The Termux:API Android app from the same source as Termux
2. The `termux-api` command-line package

Then run the clipboard verification commands above outside Ompig. If they fail there, fix the Termux:API installation before retrying Ompig's copy command.

### A Ompiglet build reports `link ... permission denied`

Ompig 0.4.1 and earlier write Ompiglet scripts, Ompiglet Binaries, and added Ompiglets with hard links, and Android does not allow hard links in Termux's data directory. The build fails at its last step with `commit ...: link ...: permission denied`. Update to Ompig 0.5.0 or later, which renames the file into place where a hard link is refused.

### `ompig update` reports `preserve current executable for rollback: link ... permission denied`

Ompig 0.4.1 and earlier keep a rollback copy of the running `ompig` with a hard link before they replace it, and Android does not allow hard links in Termux's data directory. The update stops before it changes anything. Such a release cannot update itself in Termux, so update a script installation by running the installer again:

```bash
curl -fsSL https://ompig.dev/install.sh | sh
```

Ompig 0.5.0 and later copy the running `ompig` where the hard link is refused, and `ompig update` works.

### Shared storage reports permission denied

Run `termux-setup-storage`, approve the Android permission request, and retry the path under `~/storage/` or `/storage/emulated/0`.

### Ompig fails with `has unexpected e_type: 2`

You installed the `linux-arm64` binary, a static executable that `/system/bin/linker64` cannot map. Reinstall with the installer or `npm install -g @ompig/ompig`, which choose `android-arm64` in Termux.

### `ompig install` fails with `fork/exec ... permission denied`

Ompig 0.4.0 started `npm` and `git` for `ompig install` and `ompig remove` directly, which Android rejects for a file below Termux's data directory. Ompig 0.4.1 starts them through `/system/bin/linker64` like every other program. Upgrade; if you installed a package that failed, run `ompig remove` for it, or delete its entry from the `packages` list in `~/.ompig/agent/settings.json`. Ompig 0.4.1 records a package only after it installs, as Pi does.

### `ompig setup go` reports no archive for android/arm64

Go publishes no archive for Android. Run `pkg install golang`; Ompig uses the `go` on `PATH`, and `ompig setup status` lists it.

### Validating a Rust extension stops after two minutes

Set `OMPIG_CELL_BUILD_TIMEOUT` to a longer duration, such as `30m`. It bounds both extension builds and `ompig install <dir> --validate-only`.

## Upstream Pi

See [upstream Pi's Termux documentation](https://pi.dev/docs/latest/termux) when you want to run the TypeScript reference implementation.
