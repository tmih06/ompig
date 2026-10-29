#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PROFILE="$ROOT/ompiglets/porter/ompig-porter.yaml"
OMPIG_BIN="${OMPIG_BIN:-$ROOT/bin/ompig-parity}"

[ -x "$OMPIG_BIN" ] || {
  echo "ompig-porter: Ompig binary is not executable: $OMPIG_BIN" >&2
  echo "build it with: cd $ROOT && make parity-bin" >&2
  exit 1
}

# shellcheck source=automation/porter/hermetic-agent.sh
. "$ROOT/automation/porter/hermetic-agent.sh"
home="$(mktemp -d "${TMPDIR:-/tmp}/ompig-porter-home.XXXXXXXX")"
trap 'chmod -R u+rwX "$home" 2>/dev/null || true; rm -rf "$home"' EXIT
porter_export_hermetic_agent "$home/agent"

"$OMPIG_BIN" ompiglet validate "$PROFILE"
"$OMPIG_BIN" --ompiglet "$PROFILE" "$@"
