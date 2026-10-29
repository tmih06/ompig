#!/usr/bin/env bash
# Render Pi's own --help with Ompig's identity (Pi's piConfig name "ompig" and
# configDir ".ompig"), from the pinned package that extensions/sdk-ts installs.
# Lines that do not apply to a single native binary are removed below.
set -euo pipefail
root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
pinned="$root/extensions/sdk-ts/node_modules/@earendil-works/pi-coding-agent"
# Resolve past version-manager shims before changing HOME.
node_bin=$(node -p 'process.execPath')
work=$(mktemp -d "${TMPDIR:-/tmp}/ompig.XXXXXXXX")
output=""
trap 'rm -rf "$work"; if [ -n "$output" ]; then rm -f "$output"; fi' EXIT
output=$(mktemp "$root/coding/cli/.help_upstream.XXXXXXXX")
mkdir -p "$work/node_modules/@earendil-works"
cp -R "$pinned" "$work/node_modules/@earendil-works/pi-coding-agent"
for dependency in "$pinned"/node_modules/*; do
  ln -s "$dependency" "$work/node_modules/$(basename "$dependency")" 2>/dev/null || true
done
"$node_bin" -e '
const fs = require("fs");
const file = process.argv[1];
const pkg = JSON.parse(fs.readFileSync(file, "utf8"));
pkg.piConfig = { ...pkg.piConfig, name: "ompig", configDir: ".ompig" };
fs.writeFileSync(file, JSON.stringify(pkg, null, 2));
' "$work/node_modules/@earendil-works/pi-coding-agent/package.json"
# Ompig reads OMPIG_PACKAGE_DIR (config.ts getPackageDir), so Pi's PI_PACKAGE_DIR line is renamed and keeps its description column.
# Pi lists its own app name as the third self-update target ('update [source|self|pi]'); ompig accepts 'ompig' there.
# Ompig's --tools has no +name/-name form yet (K1 in coding/cli/help_oracle_test.go), so those Pi lines are not printed.
# ompig divergence (D64): /share prints the Ompig gateway's URL, so PI_SHARE_VIEWER_URL is unused.
# The patched app name selects OMPIG_CODING_AGENT_DIR, not PI_CODING_AGENT_DIR.
# Never load the invoking agent's state or let terminal color enter the artifact.
# Pi exits immediately after printing help. Node writes regular files synchronously, so capture before filtering rather than giving it a pipe that can lose queued bytes.
(
  cd "$work"
  HOME="$work/home" OMPIG_CODING_AGENT_DIR="$work/agent" PI_CODING_AGENT_DIR="$work/agent" \
    PI_PACKAGE_DIR="$work/node_modules/@earendil-works/pi-coding-agent" FORCE_COLOR=0 \
    "$node_bin" "$work/node_modules/@earendil-works/pi-coding-agent/dist/cli.js" --help
) >"$work/help.raw"
grep -v 'PI_SHARE_VIEWER_URL' "$work/help.raw" \
  | sed -e 's/^  PI_PACKAGE_DIR /  OMPIG_PACKAGE_DIR/' \
    -e 's/^  ompig update \[source|self|pi\]   Update pi, extensions, or model catalogs$/  ompig update [source|self|ompig]   Update ompig, extensions, or model catalogs/' \
  | awk '
      /^ +Only \+name\/-name entries add to or remove from the defaults$/ { next }
      /^  # Add codemode to the default tools$/ { skip = 1; next }
      skip && /^  ompig --tools \+codemode$/ { next }
      skip && /^$/ { skip = 0; next }
      { skip = 0; print }
    ' \
  >"$output"
chmod 644 "$output"
mv "$output" "$root/coding/cli/help_upstream.txt"
echo "rendered coding/cli/help_upstream.txt from pi-coding-agent $("$node_bin" -p "require('$pinned/package.json').version")"
