# Site patch: show the signed Binary of a published Ompiglet

`ompiglet-npm-publish-site.patch` is for the Ompig-platform repository (ompig.dev). It applies with `git apply` to `main` at `91553f7`. Do not push it from here.

## What it does

`ompig ompiglet publish <name> --to npm` records the signed Binary release of a Ompiglet in the published `package.json`:

```json
"ompig": {
  "ompiglet": "ompiglet.yaml",
  "binaries": { "ref": "github:acme/reviewer/reviewer@1.2.3", "signer": "ed25519:<32 hex digits>" }
}
```

The patch makes `scripts/build-npm-catalog.mjs` read that value for the Ompiglets source and `app/package-catalog.tsx` show `ompig ompiglet pull <ref>` beside `ompig ompiglet add npm:<name>`.

- npm search results do not include `ompig`, so the build fetches `https://registry.npmjs.org/<name>/<version>` for each listed Ompiglet, one request per 250 ms.
- The reference is printed inside a shell command. `pigletBinaries` accepts only `github:<owner>/<repo>[/<ompiglet>]@<semver>` with Ompig's character set, and a signer that is `ed25519:` plus 32 hex digits. Anything else is dropped.
- A Ompiglet whose `package.json` cannot be read keeps the value from the committed snapshot when its version is unchanged. The build does not fail, and a new version does not inherit an old release.
- The published value is the publisher's claim. The catalog does not verify signatures; `ompig ompiglet pull` does.
- `docs/catalog-sync.md` and the "Publish a Ompiglet" panel describe the new field and the `--to npm` command.

## Where the field lives

`ompig.ompiglet` stays a string, the path `ompig ompiglet add` reads, because `ompig ompiglet add npm:<package>` decodes `ompig.ompiglet` as a string and a Ompiglet published with plain `npm publish` and that field keeps working. On 2026-10-05 no npm package carried the `ompig-ompiglet` keyword, and the 25 `@ompig/pigpen-*` packages with `ompig-package` are Packages that carry a `pi` manifest and no `ompig` field, so neither field affects them. The Binary release is `ompig.binaries`, a sibling of `ompig.ompiglet`, not `ompig.ompiglet.binaries`.

## Tests

`test/npm-catalog.test.mjs` gains five tests: the reference grammar (including shell metacharacters and a malformed signer), the pull command and manifest URL, adding releases to Ompiglet entries only, keeping the saved release for an unreadable manifest of the same version, and the snapshot round trip. `node --test test/npm-catalog.test.mjs` passes 24 tests. The full `node --test test/*.test.mjs` run has the same three failures with and without the patch (they need a Ompig checkout and generated data that the working copy lacked).

## Ordering

The cards show the pull command only for Ompiglets published with a Ompig that writes `ompig.binaries` (Ompig 0.5.0). Older listings are unchanged. The `/docs/latest/publishing` page ships with Ompig 0.5.0, so link to it after the docs sync.
