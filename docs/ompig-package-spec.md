# Package structure and discovery

`coding/packagecontent` is the normative discovery engine for Pi/Plugin Resources plus Ompig hooks, MCP definitions, and agent environments.

A Package is one self-contained source root. It owns the Resources nested under that root and exposes them through the upstream `package.json` `pi` block, conventional directories, or governed Ompig additions. It never owns a Ompiglet.
Package distribution is independent from Ompiglet component realization: a selected Package member may be fused, subprocess-hosted, or external according to the Ompiglet build plan.

Return to the [maintainer docs router](README.md). Read the [ecosystem ontology](ompiglet-resource-ontology.md) first, then [packages versus ompiglets](ompig-packages-vs-ompiglets.md) for the entity boundary and [marketplace boundary](ompig-plugin-marketplace-spec.md) for Plugin versus Package protocols.

## Source layout

The package root contains its resource directories directly:

```text
base-coding/
├── package.json
├── README.md
├── LICENSE
├── extensions/
│   ├── trace/
│   │   ├── go.mod
│   │   └── extension.go
│   └── workflow/
│       └── index.js
├── skills/
│   ├── commit/SKILL.md
│   └── review/SKILL.md
├── prompts/
│   └── review.md
├── themes/
│   └── acme.json
├── agents/
│   └── reviewer.agent.md
├── mcp/
│   └── source-control.json
├── hooks/
│   └── validate.json
└── .devcontainer/
    └── go-development/
        ├── devcontainer.json
        ├── Dockerfile
        └── docker-compose.yml
```

Do not add an intermediate `resources/` directory inside a package. The package root already provides the ownership boundary, so `packages/base-coding/resources/extensions/trace` adds no information.

A repository may also carry resources that no package owns:

```text
resources/
├── resources/
│   ├── extensions/
│   ├── skills/
│   ├── prompts/
│   ├── hooks/
│   ├── mcp/
│   └── agent-environments/
├── packages/
├── plugins/
└── ompiglets/
```

Ompiglets may reference those standalone resources directly. Do not create a one-member package only to make a direct resource usable.

## Manifest ownership

Use the upstream `pi` block for resource kinds Pi understands:

```json
{
  "name": "@example/base-coding",
  "version": "1.0.0",
  "pi": {
    "extensions": ["extensions/*"],
    "skills": ["skills/*"],
    "prompts": ["prompts/*.md"],
    "themes": ["themes/*.json"]
  }
}
```

Ompig must not reinterpret these fields. Explicit Pi entries override plugin metadata for the same kind; absent entries use plugin metadata or conventional directories, matching the existing installer.

Ompig reads and validates the Ompig membership block for hooks, MCP definitions, and agent environments:

```json
{
  "ompig": {
    "hooks": ["hooks/*.json"],
    "mcpServers": ["mcp/*.json"],
    "agentEnvironments": [
      ".devcontainer/go-development/devcontainer.json"
    ]
  }
}
```

This block declares membership only. Hook, MCP, extension, and Dev Container files remain authoritative for their schemas.

## Authoring and validation

Package source is ordinary `package.json`; edit it directly with the same `pi` and optional `ompig` fields shown above. Ompig adds no second membership editor.
Validate the complete source tree without installing it:

```text
ompig package validate <dir> [--json] [--no-input]
```

Validation checks manifests, member identity, JSON Resource syntax, real-path containment, and portable Dev Container/Compose closure. `ompig package list [--json] [--no-input]` is a side-effect-free view of configured Package settings and materialized paths.

Publish a Package to npm with `npm publish`. `ompig package publish [<dir>] --to npm` is an optional wrapper: it validates the Package, checks the `package.json` fields the ompig.dev catalog reads (`name`, `version`, `description`, and the `ompig-package` keyword), checks that `name@version` is free on npm, and shows npm's dry run. It publishes only with `--yes`. It also accepts `--tag`, `--access public|restricted`, and `--otp`.

## Agent environments

A package-contained agent environment is a standard Dev Container closure, not a running container or copied OCI image. The package includes the selected `devcontainer.json` and every portable file it references inside the package: Dockerfiles, Compose files, lifecycle scripts, lockfiles, and build-context files.

The validator checks JSONC syntax, lexical and symlink-resolved Package containment, Dockerfile/build-context references, selected Compose services, Compose env/build/volume references, and absolute or escaping host bind mounts. A `features` object with entries fails with `uses unsupported features`. A lifecycle command field (`initializeCommand`, `onCreateCommand`, `updateContentCommand`, `postCreateCommand`, `postStartCommand`, `postAttachCommand`) fails with `uses unsupported lifecycle command <field>`. Ompig does not implement Feature locks, lifecycle execution, metadata merge, or engine conformance.

OCI images and Dev Container Features remain external dependencies. Resolution pins their digests in the dependency lock and Ompiglet record. Publication rejects or reports references that escape the package root, use unportable absolute host paths, or require unrecorded files. Secret values never enter the package.

Member identity comes from the Dev Container `name` when present, then the named `.devcontainer/<name>/` directory. A root `.devcontainer/devcontainer.json` may use the package name as its default identity. Duplicate names fail validation.

## Member identity and activation

A ompiglet declares a package source once and selects members by kind and name:

```yaml
packages:
  base: npm:@example/base-coding@^1

extensions:
  - name: trace
    origins:
      - package:base
      - local:./fallback/trace
```

The Package inventory resolves `extensions/trace`; the filesystem path does not become public identity. A Ompiglet dependency does not add the Package to global or project settings, so its member selections are explicit in that Ompiglet. Normal `ompig install`, by contrast, preserves upstream direct-use behavior: it adds the Package to settings and Ompig discovers its enabled members, defaulting to all members unless Package filters narrow them.

When a Ompiglet is built, this exact Package/member origin feeds the exact Ompiglet record. Startup may accept the component only from the binary, registered Ompiglet release closure, required `agentEnv`, or explicit external mapping named by that record; ambient Package discovery cannot substitute another version.

A ompiglet may address the same physical resource directly during local development. The dependency graph de-duplicates package and direct routes by canonical source and content identity.

## Shared resources

Give each resource one authoritative source location. Do not copy the same extension into several packages. Let ompiglets compose multiple packages or reference a standalone resource directly. A package dependency may group a shared resource only when the package protocol can preserve its independent identity and provenance; otherwise the ompiglet remains the aggregation layer.

## Plugin facets

Plugin and Package are separate protocol entities. One source root may expose both facets:

```text
review-tools/
├── package.json
├── plugin.json
├── .claude-plugin/plugin.json
├── .ompig-plugin/plugin.json
├── extensions/
└── skills/
```

The package record and plugin record may point to this one materialized root. They retain distinct names, manifests, compatibility rules, and catalog kinds. Neither installation path copies the root into a second cache.

## Ompiglets

Packages do not contain Ompiglet YAML. Ompiglets are independent catalog entries and may reference Packages, Plugins, or direct Resources through explicit YAML. Ompig does not generate a Ompiglet from Package inventory. Package validation rejects `ompiglets` membership.

## Discovery implementation

`coding/packagecontent` is the single discovery engine for package installation, resource configuration, and ompiglet package-member resolution. It owns:

- Pi `package.json` resource entries;
- cross-harness plugin manifest merging and `.ompig-plugin` augmentation;
- conventional resource directories;
- glob, include, exclude, force-include, and force-exclude behavior;
- ordered de-duplication;
- kind/name member selection using extension source paths and Skill frontmatter;
- missing and ambiguous member diagnostics.

Callers consume its typed `Resources` inventory. They must not reproduce manifest parsing or infer package members through a second directory walker.
