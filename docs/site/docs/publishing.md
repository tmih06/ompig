# Publishing your Package or Ompiglet

Ompig lists Packages and Ompiglets that authors publish to npm, the way Pi's gallery lists Pi packages. Publishing is ordinary npm publishing plus one npm keyword. This page covers both commands, signed Ompiglet Binaries, how the ompig.dev catalog finds what you publish, and npm authentication, including CI.

You need an account on https://www.npmjs.com and the `npm` command, which Node.js installs. Signed Ompiglet Binaries also need the GitHub CLI (`gh`), signed in to GitHub. `ompig package publish` and `ompig ompiglet publish` are dry runs until you add `--yes`, so you can try them first. Plain `npm publish` has no such step.

## What gets listed

| You publish | npm keyword | Install command shown | Publish with |
|---|---|---|---|
| a Package | `ompig-package` | `ompig install npm:<name>` | `npm publish`, or `ompig package publish --to npm` |
| a Ompiglet's source | `ompig-ompiglet` | `ompig ompiglet add npm:<name>` | `ompig ompiglet publish --to npm` |

Pi packages carry the keyword `pi-package`. A Package that works in both Pi and Ompig may carry both keywords; a package that has several keywords is listed under the most specific one (`ompig-ompiglet`, then `ompig-package`, then `pi-package`).

The ompig.dev catalog finds packages by searching npm for these keywords when its site builds each day, so a package appears after the next build. The catalog has no upload step, and a listing is not a review: community listings are unreviewed.

## Publish a Package

A Package is an ordinary npm package. Add the keyword and a description to `package.json`, then publish as you do for any npm package:

```json
{
  "name": "@acme/review",
  "version": "1.2.0",
  "description": "Review helpers for Ompig",
  "keywords": ["ompig-package"],
  "pi": { "extensions": ["extensions/*"] }
}
```

```bash
npm publish --access public
```

`ompig package publish` is optional sugar over that command. It validates the Package, checks that `package.json` has the fields the catalog reads (`name`, `version`, `description`, and the `ompig-package` keyword), checks that `name@version` is free on npm, and shows what npm would pack:

```bash
ompig package publish ./base-coding --to npm
```

Without `--yes` it is a dry run: it runs `npm publish --dry-run`, prints npm's tarball listing and the exact command, and publishes nothing. With `--yes` it runs `npm publish` in the Package directory, so your `.npmrc` and scripts such as `prepublishOnly` apply as they do for plain `npm publish`. Ompig never edits `package.json`; a missing keyword is an error that tells you what to add.

## Publish a Ompiglet

A Ompiglet publishes as source. Set `release.version` in the Ompiglet, then run:

```bash
ompig ompiglet publish ./agents/reviewer.yaml --to npm
```

Ompig validates the Ompiglet as `ompig ompiglet validate` does, writes a temporary npm package, checks the registry, and shows what npm would publish. Add `--yes` to publish:

```bash
ompig ompiglet publish ./agents/reviewer.yaml --to npm --yes --access public
```

The generated package contains `ompiglet.yaml`, `package.json`, your `README.md` and `LICENSE` beside the Ompiglet (a short README is written when there is none), and the system prompt file the Ompiglet names. In `package.json`:

- `name` is the `name` in a `package.json` beside the Ompiglet, else the Ompiglet's own name. `--npm-name @acme/reviewer` sets it. Choose a scoped name: a short Ompiglet name such as `reviewer` is probably taken.
- `version` is `release.version`.
- `keywords` includes `ompig-ompiglet`, with any keywords from your own `package.json`.
- `ompig.ompiglet` is `ompiglet.yaml`, the path `ompig ompiglet add` reads. `license`, `author`, `repository`, `homepage`, `bugs`, `funding`, and `publishConfig` come from your `package.json` when you have one beside the Ompiglet. Scripts and dependencies never do.

A remote Ompiglet cannot name local paths. Ompig replaces each Package with `local:` source by `npm:<name>@^<version>`, using the `name` and `version` in that Package's own `package.json`. The authored Ompiglet:

```yaml
packages:
  review: local:./packages/review
```

The published Ompiglet:

```yaml
packages:
  review: npm:@acme/review@^1.2.0
```

When that `package.json` sets `publishConfig.registry`, the reference names that registry, for example `npm:@acme/review@^1.2.0?registry=https%3A%2F%2Fnpm.pkg.github.com`, so `ompig ompiglet add` installs the Package from it. The registry must be an HTTPS URL without credentials; otherwise publication stops and asks for `--package-map`.

Publish the Package first. Ompig asks npm whether a published version satisfies each `npm:` Package of the published Ompiglet that names no registry of its own. The dry run lists every missing Package as a warning, and `--yes` refuses to publish a Ompiglet that nobody could add. When a local Package has no npm name and version, the error says so; publish it first, or give its npm reference yourself:

```bash
ompig ompiglet publish ./agents/reviewer.yaml --to npm --package-map review=npm:@acme/review@^1.2.0
```

`--package-map <alias>=npm:<name>@<range>` can be repeated. Ompig refuses what `ompig ompiglet add` would refuse: `extends`, `agentEnv.devContainer`, a local `agentEnv.source`, file-based secrets, and extensions or skills with a `local:` origin. Move a local extension into a Package and publish that Package.

Publishing is safe to repeat. Ompig asks npm whether `name@version` exists (`npm view`, on the registry in `publishConfig.registry` when `package.json` names one) and refuses to publish over it. Raise `release.version` and publish again.

### Options

| Option | Meaning |
|---|---|
| `--yes` | Run `npm publish`. Without it the command is a dry run. |
| `--tag <dist-tag>` | npm dist-tag. A prerelease version such as `2.0.0-rc.1` needs one, for example `--tag next`. |
| `--access public\|restricted` | Package access. npm's default applies when omitted, and scoped packages default to restricted. |
| `--otp <code>` | One-time password from your authenticator. |
| `--npm-name <name>` | Published package name (Ompiglets only). |
| `--package-map <alias>=<ref>` | npm reference for a Package alias (Ompiglets only). |
| `--binaries github:<owner/repo>` | Record the signed Binary release (Ompiglets only). |
| `--no-binaries` | Record no signed Binary release (Ompiglets only). |

Ompig runs the `npm` on your `PATH`, or the command in the global `npmCommand` setting. Ompig never runs an `npm` from the directory it publishes: it looks for npm only in the absolute directories on your `PATH`, never in the current directory or a relative entry such as `.`, and resolves an `npmCommand` with a relative path against the directory you run `ompig` in. On Windows, the first element of `npmCommand` must be an `.exe`, `.com`, `.cmd` or `.bat` file; to run an npm script such as `npm-cli.js`, name its interpreter first, as in `["node", "C:\\path\\to\\npm-cli.js"]`.

## Signed Binaries next to the source

The npm package is the Ompiglet's source. A signed per-platform Binary is a separate artifact that you publish to GitHub Releases:

```bash
ompig ompiglet publish ./agents/reviewer.yaml --to github --repo acme/reviewer --sign-key ./reviewer-signing.key --yes
```

See [Ompiglet Binaries](/docs/latest/ompiglet-binaries#publish-to-github-releases) for the release layout. When you then publish the source to npm, Ompig records the signed release in `package.json`, so a catalog can show how to pull the Binary beside how to add the source:

```json
"ompig": {
  "ompiglet": "ompiglet.yaml",
  "binaries": { "ref": "github:acme/reviewer/reviewer@1.2.3", "signer": "ed25519:..." }
}
```

Ompig takes the release from the publication this machine made with `--to github`. To name a release you published elsewhere, pass `--binaries github:acme/reviewer`: Ompig downloads and verifies the signed release index, checks that it is for this Ompiglet and version, and records the release reference and the signer key id from it. `--no-binaries` records nothing. Anyone can pull the Binary with `ompig ompiglet pull <ref>`, which verifies the signatures and pins the signer.

## Add it back

`ompig ompiglet add npm:<name>` accepts exactly what publication produces. It installs the package, reads `ompig.ompiglet`, validates the Ompiglet, resolves its Packages from npm, and registers the source with an origin record that holds the exact npm version and integrity:

```bash
ompig ompiglet add npm:@acme/reviewer
ompig --ompiglet reviewer
```

## npm authentication

npm does the authentication. Ompig never reads, stores, or forwards an npm token: `npm publish` runs with your environment and your npm configuration.

- **First publish.** Log in with `npm login`. If your account requires two-factor authentication for writes, npm asks for a one-time password or a passkey on each publish, or pass `--otp`. A scoped package needs the scope to be yours, and `--access public` for a public first publish.
- **First publish is manual.** An npm Trusted Publisher can only be added to a package that already exists, so publish the first version from your own machine.
- **Trusted publishing in CI.** After the first publish, open the package's Settings on npmjs.com, add a Trusted Publisher (GitHub Actions) with your repository and workflow file name, and publish from that workflow with no stored token. The workflow needs `id-token: write` and npm 11.5.1 or newer. GitHub Actions then sets `ACTIONS_ID_TOKEN_REQUEST_URL`, and Ompig adds `--provenance` to `npm publish` so the package carries a provenance statement. Your `package.json` must name the repository.

```yaml
permissions:
  contents: read
  id-token: write
steps:
  - uses: actions/checkout@v4
  - uses: actions/setup-node@v4
    with: { node-version: 24 }
  - run: ompig package publish ./packages/review --to npm --yes
  - run: ompig ompiglet publish ./ompiglets/reviewer.yaml --to npm --yes
```

Publish the Packages before the Ompiglet that names them, as above. In the Ompiglet example, `ompig` comes from your install step.

## Additive

Publishing to npm has no Pi counterpart. Pi's package workflow is plain `npm publish` with the `pi-package` keyword ([Pi's packages reference](https://github.com/earendil-works/pi/blob/v1.2.0/packages/coding-agent/docs/packages.md)), and Ompig keeps that working unchanged. `ompig package publish --to npm` and `ompig ompiglet publish --to npm` are Ompig additions (D18). The Ompig repository records them in `docs/additive-features.md`.
