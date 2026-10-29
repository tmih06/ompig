# Windows setup

Ompig supports source builds on Windows. Do not treat Windows as release-supported until native release verification passes for the published artifact. The command environment follows Pi 1.4.0 ([Pi's Windows guide](https://github.com/earendil-works/pi/blob/v1.2.0/packages/coding-agent/docs/windows.md)).

## Build from source

Install Git and Go 1.27.3. Then run:

```powershell
git clone git@github.com:tmih06/ompig.git
Set-Location Ompig
go build -o bin\ompig.exe .\cmd\ompig
.\bin\ompig.exe --version
```

Windows support is a preview. Instead of building from source, install with `npm install -g @ompig/ompig` or `go install github.com/tmih06/ompig/cmd/ompig@latest`, or download the `windows-amd64` or `windows-arm64` zip from [GitHub Releases](https://github.com/tmih06/ompig/releases). The `install.sh` installer supports macOS and Linux only.

## Configuration paths

Ompig stores configuration below the user profile by default:

```text
%USERPROFILE%\.ompig
%USERPROFILE%\.ompig\agent
```

Set `OMPIG_HOME` to use an isolated configuration root. Set `OMPIG_CODING_AGENT_DIR` only when you need to override the agent directory.

## What was checked for this page, and what was not

Ompig's Windows support is a preview, and this page was prepared on Linux. The table says which statements were observed and which are read from the code. Nothing in it was run on a Windows machine.

| Statement | How it was checked |
|---|---|
| The source builds for Windows on both architectures | Observed: `GOOS=windows GOARCH=amd64 go build -o ompig.exe ./cmd/ompig` took 28 s and `file ompig.exe` printed `PE32+ executable (console) x86-64, for MS Windows`. The same build with `GOARCH=arm64` printed `PE32+ executable (console) Aarch64, for MS Windows`. `ompig.exe` was 90,811,904 bytes. |
| `ompig.exe` runs | **Not run.** A cross-build proves the program links for Windows, not that it starts (see [Verification](#verification)). |
| The configuration root | Observed on Linux with the same code, which does not branch on the operating system (`internal/configroot/configroot.go:50-67`): see the next section. |
| Editor order | Read from `resolveExternalEditorCommand` in `internal/codingagent/settings.go`: a non-blank `externalEditor` setting, then `VISUAL`, then `EDITOR`, then `notepad` on Windows (`nano` on every other system). Not run on Windows. |
| The npm update quarantine | Read from `internal/codingagent/windows_self_update.go` (`quarantineDirName`, `.ompig-native-quarantine`). Not run on Windows. |

### Where the configuration lives: observed

`OMPIG_HOME`, `XDG_CONFIG_HOME` and the profile directory are tried in that order. Pi has none of these: it always uses `~/.pi/agent` unless `PI_CODING_AGENT_DIR` is set (Pi `config.ts:605-611`). Ompig's own configuration root is divergence D2. Each command below ran `ompig auth check --provider openai` once with a fresh, empty home directory and printed where `auth.json` appeared (paths shortened). The Windows profile directory plays the part of `HOME`:

```text
HOME=h                                         → h/.ompig/agent/auth.json
HOME=h XDG_CONFIG_HOME=x                       → x/ompig/agent/auth.json
HOME=h XDG_CONFIG_HOME=x OMPIG_HOME=iso          → iso/agent/auth.json
HOME=h OMPIG_HOME=iso OMPIG_CODING_AGENT_DIR=a     → a/auth.json
```

On Windows `HOME` is `%USERPROFILE%`. So the default is `%USERPROFILE%\.ompig\agent` only when `XDG_CONFIG_HOME` is not set. If you set `XDG_CONFIG_HOME` for another tool, Ompig will use `%XDG_CONFIG_HOME%\ompig\agent` and your credentials will not be where the first section says. `OMPIG_CODING_AGENT_DIR` replaces the agent directory only: it is the `agent` folder itself, so `auth.json` sits directly in it. See [Environment variables](/docs/latest/environment-variables).

## Choose native Windows or WSL

Ompig runs either as a native Windows process or inside Windows Subsystem for Linux (WSL). Native Windows uses Git Bash by default for Bash commands and can optionally expose PowerShell to the model. Ompig inside WSL uses the Linux environment and its Bash installation.

| Environment | Command environment | Use it when |
|---|---|---|
| Native Windows with Git Bash | Git Bash for the built-in `bash` tool and `!` commands | Your files and development tools primarily live on Windows |
| Native Windows with the `powershell` tool | PowerShell for model tool calls; Bash remains available for `!` commands | The task depends on PowerShell modules or Windows-native commands |
| WSL | Linux Bash and tools inside the selected WSL distribution | Your files and toolchain already live in Linux or WSL |

## Use Git Bash on native Windows

For most native Windows users, installing [Git for Windows](https://git-scm.com/download/win) is sufficient.

Ompig resolves Bash in this order:

1. `shellPath` from `~/.ompig/agent/settings.json`.
2. Git Bash at `%ProgramFiles%\Git\bin\bash.exe`.
3. Git Bash at `%ProgramFiles(x86)%\Git\bin\bash.exe`.
4. `bash.exe` on `PATH`, including Cygwin, MSYS2, or legacy WSL Bash.

Type this command in Ompig to check the shell:

```text
!printf 'Bash is working\n'
```

If Ompig cannot find Bash, it reports the Git Bash locations it checked. Install Git for Windows, put another Bash executable on `PATH`, or configure `shellPath`.

## Let the model use PowerShell

The `powershell` tool is off by default and exists only when Ompig runs as a native Windows process. It runs `pwsh.exe` when available, then falls back to Windows PowerShell. It starts PowerShell with `-NoProfile -NonInteractive -ExecutionPolicy Bypass`. Administrator-enforced execution policies can still take precedence.

PowerShell discovery checks these locations in order:

1. `pwsh.exe` on `PATH`
2. `%ProgramFiles%\PowerShell\7\pwsh.exe`
3. `powershell.exe` on `PATH`
4. `%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe`

To replace the model's `bash` tool with `powershell`, set this in `~/.ompig/agent/settings.json`:

```json
{
  "defaultTools": ["-bash", "+powershell"]
}
```

`["read", "powershell", "edit", "write"]` does the same by listing the tools explicitly. Restart Ompig. The `!` and `!!` commands still use Bash.

## WSL

WSL path conversion on Windows and extra Git branch polling in WSL recognize `/mnt/<drive>`, not custom mount roots.

When Ompig uses the legacy WSL `bash.exe` launcher from Windows (`bash.exe` in the `System32` or `Sysnative` directory of `%SystemRoot%`), it sends commands over stdin to avoid command-line quoting problems.

## Use a custom Bash executable

Set `shellPath` when Bash is installed somewhere Ompig does not discover automatically:

```json
{
  "shellPath": "C:\\cygwin64\\bin\\bash.exe"
}
```

JSON uses backslashes for escape sequences. When you write a Windows path with backslashes, write each backslash twice, as shown above.

See [Shell aliases](/docs/latest/shell-aliases) for a command prefix and the complete shell-resolution behavior.

## Configure Windows Terminal

Use Windows Terminal or another terminal that supports the required input and ANSI behavior.

Ctrl+Enter can provide multiline input where Shift+Enter is intercepted by the terminal. See [Terminal setup](/docs/latest/terminal-setup#windows-terminal) and the Windows defaults in [Keybindings](/docs/latest/keybindings#platform-differences).

## External editor

Ompig selects an editor in this order:

1. `externalEditor` in `~/.ompig/agent/settings.json`;
2. `VISUAL`;
3. `EDITOR`;
4. Notepad on Windows.

Use Ctrl+G to open the current editor buffer.

## Extensions

A source extension requires its implementation toolchain. A Go extension needs Go. A Rust extension needs Cargo and Rust. A Python extension needs Python. A Node extension needs Node.

Use a prebuilt Ompiglet Binary when you do not want a target-side Go or Rust build. Interpreted components still need their recorded runtime.

## Updating

`ompig update` updates an npm or pnpm installation through its package manager, as Pi does. Before an npm update, Ompig moves the running `ompig.exe` into `node_modules\.ompig-native-quarantine` and copies it back, because Windows cannot delete a running program. The next start removes the quarantine. If anti-virus or a search indexer has `ompig.exe` open, the move waits for it, for up to 10 seconds. Pi does not wait (D99). Ompig refuses to self-update a yarn or bun installation on Windows, as Pi does.

Ompig does not replace a standalone `ompig.exe` in place (D39). Download the new release and replace the file.

## Verification

Run the Windows-native unit and integration gates before claiming support for an artifact. A Linux cross-build alone does not prove Windows terminal, process, filesystem, update, or extension behavior.

## Upstream Pi

See [upstream Pi's Windows documentation](https://pi.dev/docs/latest/windows) when you need the TypeScript reference implementation.
