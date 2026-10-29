# Plugin, package, ompiglet, and marketplace boundary

Stock Ompig owns generic Package and resolver registration contracts. A consuming product owns contributed catalog hosting, authentication, transport, and policy. `ompig ompiglet publish` publishes Ompiglet source to npm and signed Ompiglet Binaries to GitHub Releases. `ompig package publish` publishes a Package to npm.

Return to the [maintainer docs router](README.md). Read the [ecosystem ontology](ompiglet-resource-ontology.md) first, then [packages versus ompiglets](ompig-packages-vs-ompiglets.md) for the entity model and [package structure](ompig-package-spec.md) for source layout.

## Vocabulary

| Term | Owner | Purpose |
|---|---|---|
| Resource | Ompig/harness | extension, skill, prompt, theme, hook, MCP definition, agent environment, or cross-tool agent declaration |
| Plugin | cross-harness protocol | Agent Plugins 1.0.0 package plus Claude, VS Code/Copilot, or compatible vendor overlays |
| Package | Pi/Ompig package layer | versioned Ompig/Pi distribution envelope for resources |
| Ompiglet | Ompig | independent named agent composition and build source |
| Ompiglet Binary | Ompig Ompiglet lifecycle | target-native Ompig executable for one immutable Ompiglet composition; component plan may be fused/subprocess/external |
| Ompiglet Image | Ompig Ompiglet lifecycle | OCI materialization of Ompiglet component/environment closure; reserved, no producer exists |
| Resolver | generic Ompig contract | normalize a supported source/index format to a typed, materialized package root plus stable source identity |
| Ompiglet source resolver | installed product contribution | resolve one contributed scheme to a local Ompiglet YAML path; Stock Ompig registers none |
| Publisher | stock `ompig ompiglet publish` and `ompig package publish` commands | publish Ompiglet source and Packages through npm, and signed per-target Ompiglet Binaries through GitHub Releases |
| Marketplace | product/discovery layer | multi-kind catalog, authentication, validation, curation, and distribution policy |

## Generic Ompig boundary

Raw Ompig owns:

- Pi-compatible local, git, npm, and HTTP package management;
- package manifests and resource discovery;
- `ompig install` and package inventory;
- generic resolver registration for additional source schemes;
- generic Ompiglet record/artifact/component-closure validation.

Raw Ompig does not own:

- product host configuration, authentication, or API endpoints;
- authenticated catalog search, download, or publication transport;
- product validation and approval policy;
- product-specific catalog types, CRDs, or UI.

## Extension source and Package manifests

Ompig extensions use a conventional factory or exact standalone source form. Runtime registration declares identity and capabilities. Cross-tool Package metadata uses Agent Plugins `plugin.json`, vendor files such as `.plugin/plugin.json` and `.claude-plugin/plugin.json`, `.ompig-plugin/plugin.json`, or the upstream-compatible `package.json` `pi` section. Package metadata selects exact extension members but does not define their runtime capabilities.

A package may group:

| Resource | Typical path/manifest entry |
|---|---|
| extensions | `extensions/` or explicit `pi.extensions` entries |
| skills | `skills/` or explicit `pi.skills` entries |
| prompts | `prompts/` or explicit `pi.prompts` entries |
| themes | `themes/` or explicit `pi.themes` entries |
| cross-tool agents | `agents/` or plugin metadata |
| MCP definitions | `.mcp.json`, `mcp/`, plugin metadata, or governed Ompig membership |
| hooks | `hooks/` through governed Ompig membership |
| agent environments | standard Dev Container definition/closure through governed Ompig membership |

Resources sit directly under the package root; do not add a nested `resources/` hop. A Package does not carry a Ompiglet, Binary, or Image manifest; validation rejects Ompiglet membership.

## npm catalog and public pages

The ompig.dev catalog indexes npm packages carrying `ompig-ompiglet` for Ompiglets and `ompig-package` for Ompig Packages. It searches npm by keyword when the site builds each day. It labels community listings as unreviewed. It defines no `marketplace:` transport and accepts no uploads. The website code lives in a separate hosting repository (see [Provenance](project/PROVENANCE.md#public-documentation-website)); `site/docs/publishing.md` is the user guide.

Ompiglet Images remain reserved until an Image producer exists. Cross-harness Plugins remain separate catalog entities.

## Resolver contract

A contributed Ompiglet resolver may handle an additional source scheme after installed product code registers it. Stock Ompig currently registers no Ompiglet source resolver, so `marketplace:` and `catalog:` Ompiglet refs fail clearly.

Package installation independently supports local paths, npm, Git, and HTTP. Plugin manifest discovery runs only after one of those sources has been materialized. Stock Ompig does not read a marketplace index.

Requirements:

- core Package source forms win;
- duplicate contributed schemes fail deterministically;
- product authentication and API details stay inside the contributing product;
- a contributed Ompiglet resolver returns a local YAML path;
- Ompiglet acquisition rejects non-portable dependencies;
- resolution and ordinary listing do not initialize a model or Session.

## Publisher contract

Signed per-target Ompiglet Binaries publish through GitHub Releases with SHA256SUMS and a signed release index. The reusable workflow `docs/examples/ompiglet-release.yml` adds build provenance.

```text
ompig ompiglet publish <name> --to github --repo <owner/repo> --sign-key <key> [--yes]
```

Publication defaults to a dry run unless `--yes` is present. Ompig invokes `gh`, so it never handles registry tokens. It does not contact pi.dev or upload third-party bytes to ompig.dev.

### Source publication to npm

Ompiglet source publishes to npm with the `ompig-ompiglet` keyword, and a Package publishes with `ompig-package`. Catalogs discover both by npm keyword search.

```text
ompig ompiglet publish <name|path> --to npm
ompig package publish [<dir>] --to npm
```

Source publication defaults to a dry run and invokes `npm`, so Ompig never handles the npm token. `ompig ompiglet add npm:<name>` registers exactly what `publish --to npm` produces. The generated package replaces each local Package by `npm:<name>@^<version>` and records `ompig.ompiglet` (the Ompiglet path) and, when a signed Binary release exists, `ompig.binaries` (`ref` and `signer`). `ompig ompiglet publish --to npm` also accepts `--tag`, `--access`, `--otp`, `--npm-name`, `--package-map`, and `--binaries github:<owner/repo>`. A Git ref remains a source form that a user shares directly.

A live remote agent is created or updated through the consuming platform. Ompiglet publication does not create or mutate remote-agent state.

## Product implementation boundary

A product extension can contribute catalog authentication and a Ompiglet source resolver. Product API URLs, policy resources, and deployment state remain outside Stock Ompig. The stock publisher uses the GitHub and npm transports directly rather than a product-contributed nested command.

## Implementation invariants

- core Package source parsing remains independent from contributed schemes;
- contributed Ompiglet resolvers are unavailable until installed code registers them;
- Ompiglet acquisition rejects non-portable dependencies and never routes Ompiglet entries through `ompig install`;
- Package-member origins use the product-neutral source materializer and `coding/packagecontent`;
- product credentials, authentication, and API transport remain outside Stock Ompig;
- Stock Ompig reads no marketplace index. Its only publishers are `ompig ompiglet publish` and `ompig package publish`, which call `gh` and `npm`;
- provenance and license facts remain in Ompiglet records.

## Validation commands

Generic local validation remains:

```bash
ompig install ./extension --validate-only --json
ompig install --validate-only --set ./ext-a,./ext-b --json
```

Package manifests and Ompiglet YAML validate through their focused management surfaces. A product can invoke the same generic validation against materialized source, then apply product policy outside Ompig.
