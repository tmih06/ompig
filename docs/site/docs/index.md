# Ompig documentation

Ompig is a faithful Go implementation of [Pi](https://github.com/earendil-works/pi), the reference terminal coding agent. Ompig follows observable Pi behavior unless a documented divergence explains a necessary difference.

Ompig is under active development. Install it on macOS or Linux with `curl -fsSL https://ompig.dev/install.sh | sh`, on any supported platform with `npm install -g @ompig/ompig` or `go install github.com/tmih06/ompig/cmd/ompig@latest`, or download an archive from [GitHub Releases](https://github.com/tmih06/ompig/releases). Windows support is a preview. You run these commands in a terminal. If you have not used one before, start with [Your first ten minutes](/docs/latest/first-run).

## Start here

- [Your first ten minutes](/docs/latest/first-run) takes you from install to one finished task: choose a model, run a walkthrough, come back later. It assumes no programming experience.
- [Quickstart](/docs/latest/quickstart) lists every install route, including a build from source.
- [Install troubleshooting](/docs/latest/install-troubleshooting) helps when `ompig` is not found or an install fails.
- [Applied use cases](/docs/latest/applied-use-cases) are step-by-step walkthroughs for people who do not program: summarise PDFs, organise files, and how to undo.
- [Using Ompig](/docs/latest/usage) covers the normal command-line and interactive surfaces.
- [Ompig concepts](/docs/latest/concepts) explains Stock Ompig, Resources, Packages, Ompiglets, and Ompiglet Binaries.
- [Security](/docs/latest/security) explains trust and process boundaries.
- [Providers](/docs/latest/providers) explains model authentication and configuration.

## Agent applications

Use a Ompiglet to define one agent application:

```bash
ompig --ompiglet ./agents/reviewer.yaml
```

A Ompiglet selects extensions, skills, tools, prompts, model preferences, discovery policy, secrets, and environment requirements.

Build that application as a native executable when you need direct delivery:

```bash
ompig ompiglet build ./agents/reviewer.yaml \
  --format binary \
  --out ./ompig-reviewer
```

Read:

- [Ompiglets](/docs/latest/ompiglets)
- [Ompiglet Binaries](/docs/latest/ompiglet-binaries)
- [Build a derivative harness](/docs/latest/derivative-harnesses)

## Extend Ompig

Ompig extensions can use Go, Rust, Python, or Node. Each language needs its own toolchain on your computer; `ompig setup` shows which ones Ompig finds. Source extensions normally use the subprocess host. Compatible Go factories can be fused into a Ompiglet Binary. [Your first extension](/docs/latest/first-extension) builds one step by step.

Create and validate a Go extension:

```bash
ompig extension init ./review --lang go
ompig install ./review --validate-only --json
```

Read:

- [Extensions](/docs/latest/extensions)
- [Packages](/docs/latest/packages)
- [Skills](/docs/latest/skills)
- [Themes](/docs/latest/themes)

## Stock Ompig and Ompig Standard

Stock Ompig contains the Pi-compatible engine and generic extension, Package, Ompiglet, Binary, and documentation machinery. It does not activate product extensions or Ompig artwork.

Ompig Standard is an explicit Ompiglet composition. It uses ordinary extension and Ompiglet contracts. Stock Ompig never selects it automatically.

Run Standard from a source checkout:

```bash
ompig --ompiglet ompiglets/standard/ompig-standard.yaml
```

Ompig Standard requires every selected extension to be compiled into its Ompiglet Binary.

## Project facts

| Item | Value |
|---|---|
| Executable | `ompig` |
| Module | `github.com/tmih06/ompig` |
| User configuration root | `~/.ompig` or `OMPIG_HOME` |
| Agent directory | `~/.ompig/agent` or `OMPIG_CODING_AGENT_DIR` |
| Pinned Pi release (behavior oracle) | Pi 1.4.0+6c384aa3 |
| License | MIT |

## Upstream Pi

Pi remains the reference implementation. Ompig is not an official Pi release and does not imply endorsement by Pi's maintainers.

Use the upstream project when you need Pi itself:

- [Pi source](https://github.com/earendil-works/pi)
- [Pi documentation](https://pi.dev/docs/latest)
- [Pi community](https://discord.com/invite/3cU7Bz4UPx)

The Ompig website preserves upstream attribution and labels upstream commands and TypeScript APIs when it presents them for comparison.
