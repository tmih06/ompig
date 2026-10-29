# Ompig architecture

This page maps Ompig's parity core, governed additions, extension runtime, package and Ompiglet surfaces, local state, and product boundary. Read the [ecosystem ontology](ompiglet-resource-ontology.md) first so entity, carrier, component-realization, materialization, and environment axes stay separate. Return to the [maintainer docs router](README.md) for focused references.

## Layer map

```text
ompig binary
├── parity core
│   ├── CLI/TUI/session/agent loop
│   ├── providers and message conversion
│   ├── builtin tools
│   └── Pi-compatible package management
├── generic Ompig additions
│   ├── Ompiglets: compose, scope, strip, launch, build, inventory, publish
│   ├── extension SDK, host, and runtime-cell support
│   ├── Ompiglet frontend members (D91)
│   └── embedded reference documentation
├── extension host
│   ├── subprocess wire
│   ├── isolated and packed runtime cells
│   └── neutral Ompiglet runtime: fused registrations, subprocess assets, component closure
├── ompiglet runtime
│   ├── schema/discovery/inventory
│   ├── agentEnv image/Dev Container/package-source resolution
│   ├── native Docker/Podman simple runtime + external full-standard adapter
│   ├── Ompig runtime injection and effective permission policy
│   └── explicit YAML validation/registration and secure whole-process launch
├── Ompig Standard (explicit Ompiglet source; never active in Stock Ompig)
│   └── product identity and optional extension Resources
└── external product packages
    ├── authentication and provider adapters
    ├── catalog and publication transport
    ├── workflow, tracing, MCP, and remote-agent integration
    └── product runtime bindings
```

## Public package map

Ompig keeps Go package boundaries close to the packages of the Pi monorepo:

| Pi package | Ompig package | Purpose |
|---|---|---|
| `packages/agent` | `agent` | Agent loop, messages, tools, and Session primitives |
| `packages/ai` | `ai` | Models, providers, authentication, and stream types |
| `packages/coding-agent` | `coding` and `cmd/ompig` | Embedding API and command-line application |
| `packages/tui` | `tui` | Terminal UI components and renderer |
| `packages/chord` | `chord` | Application-composition runtime (`chord/delta`, `chord/context`) |
| `packages/durable` | `durable` | Detached durable Session runtime |
| `packages/env` | `env` | Durable execution environment on another machine over SSH; the daemon `env/cmd/pi-env` is a Go program (D96) |
| `packages/codemode` | `codemode` | Sandboxed code-execution runtime |
| `packages/mcp` | `mcp` | Model Context Protocol client library |
| `packages/telemetry` | `telemetry` | Standalone telemetry library |
| `packages/client`, `packages/protocol`, `packages/server` | `internal/experimental/...` | Experimental remote Session stack, built only with the `ompig_experimental` build tag |
| `packages/evals` | `internal/evals` and `cmd/ompig-evals` | Evaluation tooling |

Private command-line implementation stays under `internal/codingagent`. Ompig does not copy TypeScript `src` directories or barrel modules because those are npm package mechanics.

`docs/parity/PORT_MAP.md` defines the exact package scope and the port status of each file.

## User entity map

| Pole | Role |
|---|---|
| Resource | one extension, skill, prompt, theme, hook, MCP definition, cross-tool agent, or agent environment |
| Plugin | cross-harness capability product with Agent Plugins/vendor protocol metadata |
| Package | optionally group/distribute Ompig/Pi resources; never contain ompiglets |
| Ompiglet | consume direct, Plugin, and/or Package resources; compose, launch, build, and ship an agent |
| Ompiglet release | pin source/component closure and group optional binary/image artifacts |
| Ompiglet Binary | target-native Ompig executable for one immutable Ompiglet composition; components may be fused, subprocess, or external |
| Ompiglet Image | OCI artifact materializing the same Ompiglet/component plan and environment closure; reserved, `ompig ompiglet build --format image` reports `Ompiglet Image build is not implemented` |

See [packages versus ompiglets](ompig-packages-vs-ompiglets.md) and [ompiglet design](ompig-ompiglet-spec.md).

## Extension runtime

Normal third-party extensions use subprocess hosting. A Ompiglet Binary is still Ompig and may host the same subprocess extensions while additionally fusing compatible Go factories. Ompiglet planning records realization (`fused|subprocess|external`) and materialization (`binary|release|agentEnv|external`) per component. These are execution and delivery facts, not new authoring entities or binary kinds.

| Runtime responsibility | Correct home |
|---|---|
| Ompiglet build/component-plan orchestration | Ompiglet-owned artifact code |
| cell build/cache/planning | generic extension host/runtime-cell packages |
| subprocess component registration/materialization | neutral extension runtime plus registered Ompiglet release store |
| fused Go registration | neutral extension runtime, dormant in stock Ompig unless selected by a Ompiglet plan |
| closure verification/pull remediation | shared Ompiglet run and direct-binary bootstrap path |
| product HTTP or remote-agent transport | external product extension |

For details read [extension runtime cells](extension-runtime-cells.md).

## Local and deployed roots

Ompig separates writable product state from the agent resource/configuration root. Local defaults nest them; deployed products may mount them separately:

```text
$OMPIG_HOME/                         # default ~/.ompig; writable
├── ompiglets/                     # editable user ompiglet YAML
├── artifacts/ompiglets/           # content-addressed managed Binary outputs; Image outputs are reserved
├── receipts/ompiglets/            # resolution/artifact records and locks
├── receipts/ompiglet-publications/ # npm source publications made from this machine
├── state/<extension-id>/         # namespaced additive capability state
├── state/ompiglet-verify/          # Ompiglet Binary startup verification cache
└── docs/                         # managed embedded user manual

$OMPIG_CODING_AGENT_DIR/            # default $OMPIG_HOME/agent
├── settings.json
├── sessions/                     # created owner-only, mode 0700 (D132)
├── npm/                          # upstream npm package materialization
├── git/                          # upstream Git package materialization
├── catalog/                      # downloaded catalog bundle roots
├── extensions/
├── skills/
├── prompts/
├── themes/
├── agents/
├── hooks/
├── mcp/
├── models.json
├── SYSTEM.md
├── crashes.json                  # crash records
├── exit.log                      # interactive exits no user action asked for (D102)
└── crash-output/                 # goroutine panic output, read at the next start (D102)
```

Project-local resources mirror the agent root under the workspace:

```text
<workspace>/.ompig/
├── settings.json
├── npm/
├── git/
├── ompiglets/
├── extensions/
├── skills/
├── prompts/
├── agents/
├── hooks/
├── mcp/
└── state/<extension-id>/
```

Materialize npm and Git sources once in their upstream-compatible directories. Catalog-only archives use `catalog/`. Package and Plugin records may point to the same materialized source root while retaining separate protocol identities; Ompig does not copy that root into parallel caches.

A managed deployment can mount an immutable release bundle and point `OMPIG_OMPIGLET_PATH` at its Ompiglet. It keeps `OMPIG_HOME` and `OMPIG_CODING_AGENT_DIR` writable. Ompiglet origins and the dependency lock refer to immutable Package or Plugin roots. Sessions, settings, dynamic installs, and caches stay in the writable agent tree. This uses existing configuration roles and does not require one directory to be both immutable product input and writable runtime state.

Ompiglet records do not live in source discovery directories. Cache and materialization paths never become source or reproducibility identity.

## Boundary invariants

- Upstream-compatible behavior stays in parity core unless a numbered divergence says otherwise.
- Generic Ompig-only mechanics live under `coding/`, `internal/`, and the public extension SDKs. Stock Ompig activates no product Resource.
- Ompig Standard Resources live under `ompiglets/standard/` and load only when the user explicitly selects that Ompiglet.
- Ompig emits facts, plans, Ompiglet records, and generic artifacts; products add policy, authentication, transport, scanning, signing, and curation.
- Core command paths win. Products may contribute only unclaimed exact paths.
- A Ompiglet runtime and its records contain no literal secrets.
- Building a source entry script writes only the explicit output; Binary/Image builds never mutate shell configuration or PATH.
- Direct Ompiglet Binary/Image execution verifies component closure before model/session/tools; no `ompiglet run` indirection exists.
- The extension wire has no independent version. Wire changes land atomically across the running binary, staged SDKs, generated cells, validation, and conformance.
- Contract changes land atomically across code, schemas, clients, tests, and docs.
