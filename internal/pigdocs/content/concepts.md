# How Ompig fits together

Ompig is a small coding agent that you shape for the work in front of you. The
shortest accurate model is:

> Packages distribute available Resources. Ompiglets select and scope them.
> Ompiglet releases pin exact Resource closure and executable component plans.
> Carriers execute the same Ompiglet.

Return to the [docs router](README.md) at any time.

## Authored concepts

| Concept | What it does |
|---|---|
| Resource | contributes one extension, skill, prompt, hook, MCP definition, theme, compatible agent declaration, context input, or agent environment |
| Package | optionally distributes a versioned collection of Resources |
| Plugin | packages cross-harness capabilities using Agent Plugins/vendor formats |
| Ompiglet | defines one named agent by selecting/scoping Resources and declaring defaults/environment requirements |

Packages and Ompiglets are the two main management concepts. Packages never own
new Ompiglets.

## Package: distribute Resources

`ompig install` adds a Package to settings, so normal upstream-compatible discovery
loads its enabled members:

```bash
ompig install ./my-package
ompig install git:https://github.com/example/my-package.git
```

A Ompiglet can also reference a Package without changing settings and activate
only explicitly selected members. See [Packages](packages.md).

## Ompiglet: shape and run an agent

A Ompiglet selects extensions and skills, scopes tools/commands, sets model and
prompt defaults, connects MCP servers, controls discovery, and may require an
agent environment:

```bash
ompig --ompiglet research
```

The Ompiglet remains editable source and points at exact Resource origins instead
of copying them. See [Ompiglets](ompiglets.md).

## Ompiglet carriers and releases

A Ompiglet may have these execution carriers:

| Carrier | Meaning |
|---|---|
| Host Ompiglet | Ompig resolves and runs Ompiglet source on the host or inside its required environment |
| Ompiglet Binary | target-native Ompig executable built for one immutable Ompiglet composition |
| Ompiglet Image | OCI artifact materializing the Ompiglet and required runtime environment |

Binary and image are build outputs, not files you author or peer management
namespaces. Ompiglet source and several target artifacts may coexist in one
Ompiglet release.

## Component execution is not an artifact kind

Raw Ompig normally hosts extensions as subprocesses. A Ompiglet Binary is still Ompig
and can use the same subprocess host while additionally fusing compatible Go
factories.

Each component has two independent facts:

| Axis | Values | Meaning |
|---|---|---|
| realization | fused, subprocess, external | how it executes |
| materialization | binary, Ompiglet release, `agentEnv`, external | where its bytes/runtime come from |

A subprocess component does not make a different kind of Ompiglet Binary.
Package/Resource origin tracking and the Ompiglet release bind the exact component
version, digest, target/runtime, and location.

For example, one Ompiglet Binary may fuse a Go review extension, start a locked
Python analysis extension from its release/environment closure, and connect to an
external MCP service. Direct binary execution verifies that closure before model,
session, or tools start. It does not accept an arbitrary same-named extension
from `PATH` or ambient Package discovery.

## Required environment

`agentEnv` constrains the whole agent, independent of carrier and component
realization:

- no `agentEnv`: run on the current compatible host;
- required `agentEnv`: Host Ompiglet, Ompiglet Binary, Ompiglet Image, and managed deployments
  enter or verify it before startup;
- only explicit local one-run `--unsafe-host` may bypass it.

The code-execution sandbox is separate from the whole-agent environment.

## The full flow

```text
install/reference Resources
          |
          v
create/run Ompiglet
          |
          +---- keep using Host Ompiglet
          |
          v
build optional Binary/Image
          |
          v
publish one Ompiglet release with exact Resource closure and executable plan
```

Every stage after Resource acquisition is optional. You can use bare Ompig, run a
Ompiglet indefinitely without building, or publish selected target artifacts.

## Packages versus Ompiglets

| Question | Package | Ompiglet |
|---|---|---|
| Job | distribute Resources | compose, launch, build, and ship an agent |
| Activation | settings/discovery for direct install | only selected/scoped Ompiglet members |
| Contains/references | owned Resources | direct/Plugin/Package origins plus model/prompt/MCP/environment/build defaults |
| Runs by itself | no | yes through Host, Binary, or Image carrier |
| Marketplace kind | Package | independent Ompiglet release |

## Version and compatibility

- Ompiglet schema uses top-level `version: 1`.
- Ompiglet release SemVer uses `release.version`.
- The first production Ompiglet record schema is version 1.
- The extension subprocess wire has one current unversioned shape. It changes
  atomically with Ompig and the staged SDKs.
- Undeployed interfaces may change directly. Deployed external contracts need an
  explicit compatibility decision.

## Where to go next

- [Packages](packages.md) for installation and Package provenance.
- [Ompiglets](ompiglets.md) for composition, carriers, and environments.
- [Extensions](extensions.md) for capability authoring.
- [Skills](skills.md) for reusable instructions.
