# Packages versus Ompiglets

Read the [ecosystem ontology](ompiglet-resource-ontology.md) first.

Ompig has two authored management poles:

- a **Package** distributes available Ompig/Pi Resources;
- a **Ompiglet** selects, scopes, launches, and builds one agent. `ompig ompiglet publish` publishes its source to npm and its signed Binaries to GitHub Releases.

Ompiglet Binary, Ompiglet Image, and Ompiglet release are generated Ompiglet facets, not peer authored entities.

Return to the [maintainer docs router](README.md). Read [Package structure](ompig-package-spec.md) for layout and [Ompiglet design](ompig-ompiglet-spec.md) for schema, component closure, and carriers.

## Decision table

| Question | Package | Ompiglet |
|---|---|---|
| What is it? | versioned Resource distribution envelope | independent named agent declaration |
| Primary job | make Resources available | select/scope Resources and define runtime behavior |
| Upstream status | Pi-compatible Package model with governed Ompig additions | Ompig-only governed addition |
| Contains/references | owned extensions, skills, prompts, themes, hooks, MCP definitions, agents, environments | Package dependencies, direct origins, model/prompt settings, scopes, `agentEnv`, release/build defaults |
| Main action | install/update/remove; `ompig package publish --to npm` publishes it | create/edit/run/build/publish/remove |
| Runs by itself | no | yes through Host Ompiglet or Ompiglet Binary; Image is planned |
| Direct `ompig install` | records Package in settings; Pi discovery/filtering exposes enabled members | not applicable |
| Ompiglet dependency | materialized without settings mutation | only explicitly selected Package members activate |
| Marketplace kind | `package` | independent Ompiglet source. Available: source to npm with `ompig ompiglet publish --to npm` (catalogs list it by the `ompig-ompiglet` keyword) and signed Binaries to GitHub Releases with `--to github`. Planned: pull by installed Ompiglet name. |

## One-way relationship

```text
Resources ── optionally grouped by Package ──┐
Plugin Resources ────────────────────────────┼──> Ompiglet ──> Ompiglet release
standalone direct origins ───────────────────┘                   ├── Binary
                                                                └── Image
```

Packages do not contain or own Ompiglets. Validation rejects Ompiglet membership, avoiding cyclic ownership, implicit activation, and ambiguous removal.

## Availability versus activation

A Package may contain many Resources. Direct `ompig install` preserves upstream Pi behavior by adding the Package to settings; discovery exposes members, defaulting to all unless Package filters narrow them.

A Ompiglet dependency does not mutate settings. It declares the Package source once and selects members explicitly:

```yaml
packages:
  base: npm:@example/base-coding@^1

extensions:
  - name: trace
    origins:
      - package:base
      - npm:@example/fallback-trace@^1
```

The section kind plus `trace` selects the Package member. Package filesystem paths remain implementation details. Ordered origins remain independent. The resolution graph de-duplicates identical canonical sources.

## Package origin feeds component closure

When a Ompiglet is built, the selected Package/Resource origin becomes a component record:

```text
Package member identity
  -> exact source/version/content digest
  -> Ompiglet capability scope
  -> realization: fused | subprocess | external
  -> materialization: binary | release | agentEnv | external
  -> Ompiglet record and managed release closure
```

This is how a Ompiglet Binary can safely use ordinary subprocess extensions. The binary does not accept whatever happens to be installed. It verifies the exact origin-pinned component from its embedded assets, registered Ompiglet release, required environment, or explicit external mapping.

Fusing a compatible Go factory is an optimization. It does not create a different Ompiglet Binary ontology.

## Ompiglet source stays explicit

Package and Resource inventory can inform the author, but Ompig does not infer a Ompiglet from installed state. The user, editor, or agent writes explicit Package aliases and member origins in Ompiglet YAML. Installing a Package alone never creates or activates a Ompiglet.

## Publication flow

```text
install/reference Resources
          |
          v
available direct/Plugin/Package inventories
          |
          v
author independent Ompiglet YAML
          |
          +---- publish source + exact resolution/component closure
          |
          v
build optional Ompiglet Binary and/or Ompiglet Image
          |
          v
publish one Ompiglet release with selected artifacts and records
```

| Intent | Catalog entity/action |
|---|---|
| cross-harness capability product | Plugin |
| grouped Ompig/Pi Resources | Package |
| editable whole-agent composition | Ompiglet source/release |
| target-native agent executable | Ompiglet Binary facet |
| environment-complete runnable agent | Ompiglet Image facet |

Do not wrap a Ompiglet in a Package to publish it. Do not merge Plugin and Package schemas when both point to one source root.

Current state:

- `ompig ompiglet publish <name|path> --to npm` publishes Ompiglet source. Each local Package becomes its `npm:<name>@^<version>` reference, and `package.json` gets the `ompig-ompiglet` keyword and `ompig.ompiglet`.
- `ompig ompiglet publish <name|path> --to github --repo <owner/repo> --sign-key <key>` publishes signed per-target Ompiglet Binaries to GitHub Releases. A `--to npm` publication records that release in `package.json` under `ompig.binaries` when the GitHub publication ran on the same machine or `--binaries github:<owner/repo>` names it.
- `ompig package publish [<dir>] --to npm` publishes a Package. Plain `npm publish` also works.
- One published Ompiglet release that groups source and Binary facets does not exist yet. Ompiglet Image publication stays reserved until an Image producer exists.

## Authoring safety

Package source is authored directly as ordinary `package.json`; Ompig provides side-effect-free `package validate` rather than a second membership editor. A running session is never an implicit Package source because it may contain third-party Resources whose licenses, provenance, or layout prohibit rebundling. Ompiglets may use session/repository discovery because they write references and selections rather than copying source.

## Source identity

Install, Package, Plugin, and Ompiglet origins use `coding/source`. Exact version, commit, digest, membership, and content identity belong in Ompiglet resolution and artifact records. Materialization/cache paths never become public identity.
