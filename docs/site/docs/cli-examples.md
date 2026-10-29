# Command line by example

[Command line](/docs/latest/cli) lists every verb and option. This page runs the common ones and shows what they print. Each block is a command, its output and its exit status (`exit N` after the output; 0 means success). The outputs were captured from one Ompig binary under a scratch home directory, so paths read `/home/you/...` here and your own paths differ. Commands that need a network or a model say so.

Version: `0.5.0+1.2.0` in the captures below. Yours prints your own release.

## Is it installed, and what is it?

```bash
ompig --version
```

```text
0.5.0+1.2.0
```

The number is `<Ompig version>+<Pi version it follows>`. Pi prints only its own version, so this is a difference (D63). `ompig version` is the detailed form:

```bash
ompig version
```

```text
ompig: 0.5.0
upstream pi: 1.2.0
go: go1.27.2
platform: linux/amd64
build: dev
```

`ompig diagnose` prints where Ompig keeps things. Start here when a path is not what you expected. It is not in `ompig --help`. The top of its output:

```text
ompig diagnostics
────────────────────────────────────────────────────────────
ompig-version:       0.5.0
upstream-pin:      1.2.0
build:             dev
go runtime:        go1.27.2
platform:          linux/amd64
binary:            /usr/local/bin/ompig
...
paths
────────────────────────────────────────────────────────────
config root:       /home/you/.ompig
agent dir:         /home/you/.ompig/agent
auth.json:         /home/you/.ompig/agent/auth.json
cwd:               /home/you/project
crash log:         /home/you/.ompig/agent/crashes.json
exit log:          /home/you/.ompig/agent/exit.log
```

`ompig setup` shows which toolchains your computer has. Ompig itself needs none of them. Extensions and Ompiglet builds do:

```text
Ompig toolchains (linux/amd64)

  go         /usr/lib/go-1.22/bin/go                      Go extension cells and native Ompiglet builds
  container  docker (/usr/bin/docker)                     container Ompiglet builds
  cargo      missing                                      Rust extension cells
  node       missing                                      Node and TypeScript extensions
  python3    /usr/bin/python3                             Python extensions
  git        /usr/bin/git                                 git: Package sources

Next steps:
  cargo      install Rust from https://rustup.rs
  node       install Node.js 22.13 or newer from https://nodejs.org

Ompig runs without these tools. They are needed only for the extension languages and Ompiglet builds listed above.
```

`ompig verify` checks the binary you are running and prints its SHA-256 digest. A development build is unsigned, and a released one shows its signature state:

```text
ompig 0.5.0+1.2.0 go1.27.2 linux/amd64 , build dev

/usr/local/bin/ompig (binary)
  sha256:d105f57d45d384422fbc3f7632dbf2aeb11f6e8c73dc3b65ba73cf5068e7648c
  n/a  ompiglet signature: unsigned (no signature: anyone who can rewrite this file can rewrite its build record)
```

## Sign in and check credentials

```bash
ompig login --list
ompig login --list --json
```

The first form prints a table (see [Your first ten minutes](/docs/latest/first-run#2-choose-a-model)). The second prints the same targets for a program to read: `{"targets":[{"id":"anthropic","name":"Anthropic","callbackServer":true}, ...]}`.

`ompig auth check` answers whether a provider has usable credentials, with no model call. Exit 0 means ready and 1 means not ready (2 is invalid). Here no OpenAI key is set, and then one is set in the environment:

```bash
ompig auth check --provider openai --json
OPENAI_API_KEY=x ompig auth check --provider openai --json
```

```text
{"status":"not_ready","provider":"openai","reason":"credentials_not_configured"}
```

```text
{"status":"ready","provider":"openai","authType":"api_key"}
```

`ompig auth print-api-key --provider openai` prints the key Ompig would use (here `x`) for another program to use. Treat its output as a secret. `ompig logout openai` removes a credential stored in `auth.json`:

```text
Logged out from openai. Credentials removed from auth.json
```

## Run one prompt without the screen

Print mode runs a prompt and exits. It needs a model. These runs use your default model. Add `--model <provider>/<model>` to pick another one. The answers below came from a scripted stand-in model, so your model's words differ.

```bash
ompig -p "Summarise" @notes.txt
```

```text
The notes cover the API migration, a hire and the budget.
```

`@notes.txt` puts the file's text into the message ([Applied use cases: weekly report](/docs/latest/usecase-weekly-report)). `--continue` (`-c`) continues the previous session, so a script can ask a follow-up:

```bash
ompig -c -p "And one more thing"
```

```text
The follow-up answer.
```

`--mode json` prints every event as one JSON object per line, for a program to read:

```bash
ompig --mode json -p "Summarise"
```

The first lines of the output:

```text
{"type":"session","version":3,"id":"01a12bd7-6081-75aa-b191-02a07fdc9c8c","timestamp":"2026-10-11T16:41:37.665Z","cwd":"/home/you/project"}
{"type":"agent_start"}
{"type":"turn_start"}
{"type":"message_start","message":{"role":"system", ...
{"type":"message_end","message":{"role":"system", ...
{"type":"message_start","message":{"role":"user","content":[{"type":"text","text":"Summarise"}], ...
```

The events are described in [JSON mode](/docs/latest/json).

`--export` writes a saved session to an HTML page and exits, with no model and no screen:

```bash
ompig --export ~/.ompig/agent/sessions/--home-you-project--/2026-10-11T16-41-37-615Z_01a12bd7-604f-73fb-9fe8-0b43cc137b7c.jsonl report.html
```

```text
Exported to: report.html
```

## Look at what is installed

On a fresh install these print:

```bash
ompig list
ompig status
```

```text
No packages installed.
```

```text
Ompig status: healthy
  paths: home=/home/you/.ompig agent=/home/you/.ompig/agent project=/home/you/project/.ompig
  packages: 0 (user=0 project=0)
  resources: 0 (enabled=0 disabled=0)
  ompiglets: 0 (source=0 binary=0)
```

### Install a Package

A Package is a folder with a `package.json` and resource folders. This one carries a skill and a prompt template:

```text
mypkg/
├── package.json
├── prompts/review.md
└── skills/commit/SKILL.md
```

`prompts/review.md` and `skills/commit/SKILL.md` are short Markdown files. A front matter block names and describes each one ([Prompt templates](/docs/latest/prompt-templates), [Skills](/docs/latest/skills)):

```markdown
---
description: Review the current change
---
Review the change.
```

```markdown
---
name: commit
description: Write a commit message
---
Write a commit message.
```

`package.json`:

```json
{
  "name": "@example/team-kit",
  "version": "1.0.0",
  "keywords": ["ompig-package"],
  "pi": { "skills": ["skills/*"], "prompts": ["prompts/*.md"] }
}
```

```bash
ompig package validate ./mypkg
ompig install ./mypkg
ompig list
```

```text
Package valid: /home/you/project/mypkg
  skills: 1
  prompts: 1
```

```text
Installing ./mypkg...
Installed ./mypkg
```

```text
User packages:
  ../../project/mypkg
    /home/you/project/mypkg
```

`ompig status` now reports a problem if two resources collide. Here the package's `commit` skill and a `commit` skill in `~/.ompig/agent/skills` have the same name. The status turns `invalid`, names both files and exits 1 (exit 1):

```text
Ompig status: invalid
  paths: home=/home/you/.ompig agent=/home/you/.ompig/agent project=/home/you/project/.ompig
  packages: 1 (user=1 project=0)
  resources: 3 (enabled=3 disabled=0)
  ompiglets: 0 (source=0 binary=0)
Error: duplicate skills resource "commit" in user scope: /home/you/.ompig/agent/skills/commit and /home/you/project/mypkg/skills/commit
```

Remove one of them. `ompig remove ./mypkg` prints `Removing ./mypkg...` and `Removed ./mypkg`. See [Packages](/docs/latest/packages) for sources such as `npm:` and `git:`.

## MCP servers

`ompig mcp` edits and checks `mcp.json` without a session. Adding a server whose program is missing is accepted, and `ompig mcp list` reports the failure (exit 1):

```bash
ompig mcp list
ompig mcp add files -- npx -y @modelcontextprotocol/server-filesystem .
ompig mcp list
ompig mcp remove files
```

```text
No MCP servers configured. Add them to /home/you/.ompig/agent/mcp.json or .ompig/mcp.json.
```

```text
Added global MCP server "files" in /home/you/.ompig/agent/mcp.json.
Check it with: ompig mcp list
```

```text
files: failed (codemode, global)
  npx -y @modelcontextprotocol/server-filesystem .
  spawn npx ENOENT
```

```text
Removed global MCP server "files" from /home/you/.ompig/agent/mcp.json.
```

`spawn npx ENOENT` means `npx` is not installed. See [MCP servers](/docs/latest/mcp).

## Ompiglets

A Ompiglet file defines one agent. This one is read-only and has its own instructions:

```yaml
name: reviewer
description: "Reviews a change and lists risks."

systemPrompt:
  text: "You review changes. List the risks first, then the questions."

tools: [read, grep, find, ls]
```

Check, inspect, register, list and remove it:

```bash
ompig ompiglet validate ./agents/reviewer.yaml
ompig ompiglet show ./agents/reviewer.yaml
ompig ompiglet add ./agents/reviewer.yaml
ompig ompiglet list
ompig ompiglet remove reviewer
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

```text
Added reviewer (0 extensions, 0 skills) → /home/you/.ompig/ompiglets/reviewer.yaml
```

```text
NAME      SCOPE  SOURCE                                BINARY  IMAGE  DESCRIPTION
reviewer  user   /home/you/.ompig/ompiglets/reviewer.yaml  -       -      Reviews a change and lists risks.
```

```text
removed source: /home/you/.ompig/ompiglets/reviewer.yaml
```

The file format is strict: Ompig rejects a field it does not know and names the line. `ompig ompiglet schema` prints every allowed field. Two common mistakes are `system:` in place of `systemPrompt:`, and `inline:` in place of `text:`:

```text
FAIL: Ompiglet /home/you/project/agents/reviewer.yaml: parse ompiglet YAML: yaml: unmarshal errors:
  line 4: field system not found in type ompiglet.Ompiglet
FAIL: 1 validation error(s)
```

```text
FAIL: Ompiglet /home/you/project/agents/reviewer.yaml: parse ompiglet YAML: yaml: unmarshal errors:
  line 5: field inline not found in type ompiglet.PromptRef
FAIL: 1 validation error(s)
```

Both exit 1.

The prompt key is `text` (or `file`). Run the Ompiglet with `ompig --ompiglet ./agents/reviewer.yaml` or, once registered, `ompig --ompiglet reviewer`. Inside it, the command `/ompiglet` names the active Ompiglet:

```text
 Ompiglet: reviewer
 Source: /home/you/project/agents/reviewer.yaml
```

Write a script that launches it by name, for another person to run:

```bash
ompig ompiglet build reviewer --format script --out ./run-reviewer
cat ./run-reviewer
```

```text
Ompiglet script written: /home/you/project/run-reviewer
```

```text
#!/bin/sh
exec ompig --ompiglet '/home/you/.ompig/ompiglets/reviewer.yaml' "$@"
```

`ompig ompiglet keygen <file>` makes a signing key for Binaries, and `ompig ompiglet trust list` shows who you trust. Without a file name, `keygen` prints `Usage: ompig ompiglet keygen <private-key-path>` and exits 2:

```bash
ompig ompiglet keygen ./signing.key
ompig ompiglet trust list
```

```text
Ompiglet signing key ed25519:c74cf1a77b670764e896cfbf2800bb28
  private: ./signing.key (keep it secret; sign with ompig ompiglet build <name> --format binary --sign-key ./signing.key)
  public:  ./signing.key.pub (publish it; users trust it with ompig ompiglet trust add)
```

```text
Ompiglet trust store: /home/you/.ompig/ompiglet-trust
Require signature: off (unsigned Ompiglet Binaries run and report unsigned)
Trusted signers:
  none
```

See [Ompiglets](/docs/latest/ompiglets) and [Ompiglet Binaries](/docs/latest/ompiglet-binaries).

## Extensions

The shortest path from nothing to a working extension is on [Your first extension](/docs/latest/first-extension).

`ompig extensions cache stats` reports the build cache for extensions:

```text
Extension cache: 1 entries, 5087678 bytes
```

`ompig reload` re-stages the extension SDKs that Ompig embeds and drops builds that an older SDK made:

```text
ompig reload: staged go (98 files) to /home/you/.ompig/state/pigsdk/sdk-0cd9eb109aaf2c5f
ompig reload: staged python (24 files) to /home/you/.ompig/state/pigsdk/sdk-py-3400e290bc372488
ompig reload: staged rust (56 files) to /home/you/.ompig/state/pigsdk/sdk-rs-d045d500e4528762
ompig reload: dropped 0 unreachable build(s), 0.0 MB, kept 1
```

## Update

`ompig update` updates Ompig itself, and it refuses when Ompig cannot replace the file it is running from (a development build, or a copy placed by a package manager). It then says how (exit 1):

```text
Extensions are skipped. Run ompig update --extensions to update extensions.
ompig cannot self-update this installation. Executable: /usr/local/bin/ompig. Reinstall it with the package manager, wrapper, or source checkout that provided it, or download a new ompig binary from your provider's releases.
```

## Read the manual offline

`ompig docs list` lists the embedded pages, `ompig docs path` prints the folder that holds them and `ompig docs show <name>` prints one page:

```bash
ompig docs path
ompig docs show onboarding
```

```text
/home/you/.ompig/docs/bundle-7d53e1ebb619aaaf
```

The first lines of the page:

```text
# Get started with Ompig

Stock Ompig includes the reference documentation required to configure models,
author extensions, use Packages, and compose Ompiglets. ...
```

## Not shown on this page

These commands need a network, a login in a browser, a registry account or a terminal screen of their own, so the page does not print their output. Each has its own page.

| Command | Why | Where to read |
|---|---|---|
| `ompig login <provider>` | Opens your provider's sign-in | [Your first ten minutes](/docs/latest/first-run) |
| `ompig ompiglet pull`, `ompig ompiglet update`, `ompig ompiglet publish --to github` | Talk to GitHub Releases | [Ompiglet Binaries](/docs/latest/ompiglet-binaries) |
| `ompig ompiglet publish --to npm`, `ompig package publish` | Talk to npm, which asks for your login | [Publish a Package or Ompiglet](/docs/latest/publishing) |
| `ompig update` to a new release | Downloads and replaces the binary | [Quickstart](/docs/latest/quickstart) |
| `ompig config` | A full-screen list of resources | [Packages](/docs/latest/packages) |
| `ompig extension upgrade`, `ompig extension preview-login` | Rewrite or preview an extension written for another SDK | [Extensions](/docs/latest/extensions) |
| `ompig ompiglet trust add`, `ompig ompiglet trust require` | Change who you trust to sign Binaries | [Ompiglet Binaries](/docs/latest/ompiglet-binaries) |
