#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$ROOT"

OMPIG_BIN=${OMPIG_BIN:-$ROOT/bin/ompig-parity}
if [[ ! -x "$OMPIG_BIN" ]]; then
  echo "ERROR: OMPIG_BIN is not executable: $OMPIG_BIN" >&2
  echo "Run: make parity-bin" >&2
  exit 2
fi

echo "qc-smoke: ompig=$OMPIG_BIN"
out=$(mktemp -d "${TMPDIR:-/tmp}/ompig-qc.XXXXXXXX")
trap 'rm -rf "$out"' EXIT
"$OMPIG_BIN" --version >"$out"/version.txt
OMPIG_HOME=$(mktemp -d "${TMPDIR:-/tmp}/ompig.XXXXXXXX") "$OMPIG_BIN" docs sync >"$out"/docs-sync.txt
OMPIG_HOME=$(mktemp -d "${TMPDIR:-/tmp}/ompig.XXXXXXXX") "$OMPIG_BIN" docs list >"$out"/docs-list.txt
grep -q 'providers.md' "$out"/docs-list.txt
"$OMPIG_BIN" install examples/extensions/go-factory --validate-only --json >"$out"/go-factory.json
grep -q '"valid": true' "$out"/go-factory.json
grep -q '"hello"' "$out"/go-factory.json

OMPIG_PARITY_OMPIG_BIN="$OMPIG_BIN" go test -tags=parity ./test/parity/runner \
  -run 'TestParity/(01-version-flag|10-login-subscription-providers|01-register-handshake)$' \
  -count=1 -v -timeout 5m -args -ompig-parity.results="$out"/parity-results.json
ran=$(go run ./test/parity/cmd/checkparityran -results "$out"/parity-results.json)
if [[ "$ran" -eq 0 ]]; then
  echo "ERROR: qc-smoke compared 0 parity scenarios: the runner skipped before running any" >&2
  exit 1
fi

echo "qc-smoke: ok"
