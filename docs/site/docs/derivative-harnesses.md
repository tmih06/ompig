# Build a derivative harness

A derivative Ompig harness is normally an agent application built from a Ompiglet and ordinary Resources. It should not require a fork of Stock Ompig.

Use this model when a product, team, or workflow needs its own identity and capabilities.

## Architecture

```text
Stock Ompig
   |
   +-- Package A -------- extensions and skills
   |
   +-- Package B -------- prompts and themes
   |
   +-- Ompiglet ----------- one agent application
          |
          +-- source execution
          |
          +-- Ompiglet Binary
```

Stock Ompig owns generic execution. Packages distribute capabilities. The Ompiglet selects one application. The Ompiglet Binary is an optional delivery form.

## Why use a Ompiglet instead of a new harness

A new harness executable often duplicates:

- model and session behavior;
- extension discovery;
- configuration parsing;
- security controls;
- update behavior;
- release tooling;
- documentation;
- failure recovery.

The duplicate then drifts from its base project.

A Ompiglet keeps application choices in one explicit composition while reusing the verified Ompig engine.

## What belongs in the application

Put application-specific behavior in ordinary Resources:

| Application need | Resource or field |
|---|---|
| Tools, commands, providers, and interactive behavior | Extension |
| Task procedures | Skill |
| Application instructions | `systemPrompt` or Prompt Resource |
| Visual identity | Login or theme extension Resource |
| Built-in tool policy | Ompiglet `tools` |
| Model preference | Ompiglet `model` |
| Ambient Resource policy | Ompiglet `discovery` |
| Runtime requirements | Ompiglet `agentEnv` and `secrets` |
| Built-ins the application does not offer | Ompiglet `strip`, in deny mode or keep mode (D92) |
| A custom interactive renderer | Ompiglet `slots.frontend`, a Go frontend member fused into the Binary (D91) |

Do not add these choices to `cmd/ompig` or Stock Ompig startup.

Pi lets a source fork rename its command and configuration directory through a `piConfig` block in `package.json`. Ompig has no such block: the command name `ompig` and the `~/.ompig` directory are part of the binary (D2). Use a Ompiglet to give an application its own name and identity.

## What belongs in Stock Ompig

Change Stock Ompig only for a generic requirement such as:

- behavior required for Pi compatibility;
- a stable public extension capability;
- Package, Ompiglet, or Binary infrastructure;
- provider or session behavior used by all compositions;
- generic lifecycle, security, or verification support.

A product command, product login, private API, or application workflow does not belong in Stock Ompig.

## Build the application

### 1. Create extensions

Create one focused Go factory for each executable capability:

```bash
ompig extension init ./extensions/review --lang go
```

Validate it without installing it:

```bash
ompig install ./extensions/review --validate-only --json
```

Use more than one extension when the capabilities have independent state, release, or failure ownership. Do not split one cohesive capability only to create more packages.

### 2. Create the Ompiglet

```yaml
name: review-agent
description: "Repository review application"

extensions:
  - name: review
    origins: [local:./extensions/review]

skills:
  - name: review-policy
    origins: [local:./skills/review-policy]

tools: [read, grep, find]

discovery:
  extensions: []
  skills: []

systemPrompt:
  file: ./prompts/review.md
```

Explicit empty discovery lists prevent user or workspace Resources from changing this application.

### 3. Run from source

```bash
ompig ompiglet validate ./review-agent.yaml
ompig --ompiglet ./review-agent.yaml
```

Use source execution during development. It provides normal extension validation, process isolation, and reload behavior.

### 4. Build a Binary

```bash
ompig ompiglet build ./review-agent.yaml \
  --format binary \
  --out ./ompig-review-agent
```

Run the resulting agent application directly:

```bash
./ompig-review-agent
```

## Native derivative applications

Require all selected extensions to compile into the Binary:

```yaml
build:
  extensionRealization: fused
```

This requirement is appropriate when you need one native executable and every application extension is a reviewed Go factory.

The build fails if any selected extension cannot fuse. It does not silently produce a mixed application.

## Mixed derivative applications

Omit `extensionRealization` when the application intentionally uses another language, a native subprocess, or an external service. A Ompiglet Binary can carry TypeScript and JavaScript extensions; it runs them with the installed Node, version 22.13 or newer.

The component plan then records each realization and materialization. The application remains one Ompiglet even when its implementation spans processes.

## Package the Resources

Use a Package when several applications or teams need the same Resources:

```text
review-resources/
├── package.json
├── extensions/
├── skills/
└── prompts/
```

A Package makes the Resources available. Each Ompiglet still selects the members it needs.

Do not put Ompiglet activation in Package installation. Installing a capability must not change which agent application starts.

## Application identity

An application can select an extension that sets its login or header identity. Stock Ompig provides the generic host API but does not own the derivative artwork.

Ompig Standard follows this rule. Its `piglogin` extension owns the Ompig Standard identity and `/sprite` command. Its `pigrunner` and `angrypigs` extensions own the `/runner`, `/ompig-runner` and `/angry-pigs` games and their state, and `bunruntime` owns `/bun-runtime`.

## Failure boundaries

Choose realization with the failure model in mind.

### Subprocess

Use a subprocess when you need language portability or process isolation. Ompig can detect transport loss, cancel pending calls, quarantine failed cells, and retain a previous working extension set during failed reload.

The current operation fails. Ompig does not replay it automatically.

### Fused

Use fusion when you need one native executable and accept a shared process boundary. A fused extension has lower process overhead but requires stricter review because a panic, deadlock, or resource leak can affect the whole application.

### External

Use an external component when another service owns deployment, scaling, credentials, and availability. Keep its endpoint and secret values outside portable Ompiglet source.

## Avoid these patterns

Do not:

- fork Ompig only to register application extensions;
- add a product switch to Stock Ompig;
- make Package installation select a Ompiglet;
- hide application behavior in a launcher script;
- discover every installed Resource implicitly;
- call an external component “embedded”;
- claim a Binary is self-contained when it still has external requirements;
- replay interrupted tool calls automatically.

## Ompig Standard as the reference composition

Ompig Standard is the repository's derivative-harness example. It uses:

- an explicit Ompiglet;
- ordinary public extension contracts;
- application-owned identity;
- application-owned games with timer-driven UI;
- no automatic Stock Ompig activation;
- an all-fused Binary requirement.

New Ompig Standard capabilities follow the same boundary. They do not become built-in Stock Ompig commands.

## Share the application

Publish the Ompiglet's source to npm with `ompig ompiglet publish <name> --to npm`, and its signed Binaries to GitHub Releases with `ompig ompiglet publish <name> --to github`. Recipients add the source with `ompig ompiglet add npm:<name>` or pull a Binary with `ompig ompiglet pull`. See [Publish a Package or Ompiglet](/docs/latest/publishing).

## Related documentation

- [Ompig concepts](/docs/latest/concepts)
- [Extensions](/docs/latest/extensions)
- [Packages](/docs/latest/packages)
- [Ompiglets](/docs/latest/ompiglets)
- [Ompiglet Binaries](/docs/latest/ompiglet-binaries)
