# Ompig concepts

Ompig separates execution, capability distribution, and agent composition. This separation keeps Stock Ompig small and prevents each agent application from becoming a new harness fork.

The short model is:

> Packages distribute Resources. Ompiglets select and scope Resources. Ompiglet Binaries deliver a fixed Ompiglet as a native executable.

## Stock Ompig

Stock Ompig is the product-neutral coding agent. It provides:

- Pi-compatible agent behavior;
- built-in coding tools;
- the extension host;
- Package installation;
- Ompiglet loading and validation;
- Ompiglet Binary construction and verification;
- embedded offline documentation.

Stock Ompig does not activate Ompig Standard or another product composition. It has no marketplace and no derivative-harness behavior. Two Ompig-only features ship in the Stock binary: the first-time setup dialog (D88) and the built-in `ompig-login` extension, which holds the fifteen built-in sprites and registers `/sprite` (D2).

## Resources

A Resource contributes one focused part of an agent application.

| Resource | Purpose |
|---|---|
| Extension | Add executable tools, commands, events, providers, or UI behavior. |
| Skill | Add task instructions that the agent loads when needed. |
| Prompt | Add a reusable prompt or template. |
| Theme | Change terminal appearance. |
| Hook | Describe an event command. |
| MCP definition | Describe an external tool service. |
| Agent environment | Define the environment required by the whole agent. |

A Resource does not activate itself.

## Packages distribute Resources

A Package is an optional distribution envelope. It can contain one or more Resources. Install a Package when you want to make its Resources available:

```bash
ompig install ./my-package
ompig install git:https://github.com/example/my-package.git
```

Package installation does not select a Ompiglet. A Package never owns or activates a Ompiglet.

See [Packages](/docs/latest/packages).

## Ompiglets define agent applications

A Ompiglet is one named whole-agent declaration. It can select:

- extensions and skills;
- built-in and extension tool scopes;
- a system prompt;
- a model preference;
- ambient discovery policy;
- secrets and environment requirements;
- Binary build requirements;
- built-in tools, commands, extensions, model APIs and features to strip (D92);
- a Go frontend member that draws the interactive mode in a Ompiglet Binary (D91).

Run a Ompiglet from source:

```bash
ompig --ompiglet ./agents/reviewer.yaml
```

A Ompiglet is the normal unit for an agent application. It keeps the application configuration explicit without creating a separate Ompig fork or wrapper harness.

Publish a Ompiglet's source to npm with `ompig ompiglet publish <name> --to npm`, and its signed Binaries to GitHub Releases with `--to github`. See [Ompiglets](/docs/latest/ompiglets) and [Publish a Package or Ompiglet](/docs/latest/publishing).

## Ompiglet Binaries deliver fixed applications

A Ompiglet Binary is a target-native Ompig executable for one fixed Ompiglet composition:

```bash
ompig ompiglet build ./agents/reviewer.yaml \
  --format binary \
  --out ./ompig-reviewer
```

A Binary is a delivery form of the same Ompiglet. It is not another configuration model.

Each executable component records two independent facts:

| Axis | Values | Meaning |
|---|---|---|
| Realization | fused, subprocess, external | How the component runs. |
| Materialization | binary, release, `agentEnv`, external | Where its bytes or runtime come from. |

See [Ompiglet Binaries](/docs/latest/ompiglet-binaries).

## Ompig Standard

Ompig Standard is an explicitly selected Ompiglet. It demonstrates how a derivative Ompig experience can use public extension and Ompiglet contracts without adding product behavior to Stock Ompig.

The Standard source is:

```text
ompiglets/standard/ompig-standard.yaml
```

Its Binary requires every selected extension to be a fuse-compatible Go factory. The build fails if an extension would use a subprocess fallback. The current composition selects `piglogin` for identity, `pigrunner` for the Ompig Runner game, `angrypigs` for the Angry Pigs game and `bunruntime` to install the Bun TypeScript runtime (see [TypeScript runtimes](/docs/latest/typescript-runtimes)).

## Choose the correct unit

| Need | Use |
|---|---|
| Add one executable capability | Extension |
| Share several Resources | Package |
| Define one complete agent application | Ompiglet |
| Hand off one native executable | Ompiglet Binary |
| Change Ompig core behavior | Stock Ompig change, only when the behavior is generic or required for Pi parity |

## Why this prevents harness sprawl

A conventional derivative harness often combines configuration, extension loading, branding, deployment, and application behavior in a new executable. Each new executable then owns a separate update path and can drift from its base agent.

Ompig keeps these responsibilities separate:

```text
Stock Ompig
   |
   +-- Packages make Resources available
   |
   +-- Ompiglet selects one agent application
          |
          +-- run from source
          |
          +-- build as a Ompiglet Binary
```

This model gives each application a clear identity while keeping one compatible execution engine.
