#!/usr/bin/env bash
# Runs Ompig's extension conformance suite on the compiled aot host.
#
# The suite writes its extensions per test, so it runs on a dynamic host: the aot runtime's executable with jiti compiled in and no extension compiled in. That
# exercises the runtime tree, the harness and the wire on the compiled executable; extensions compiled in are covered by ompiglets/aot's end-to-end tests.
#
# Needs OMPIG_AOT_TEST_COMPILER (a Pi-Bolt 0.7.4 runtime executable). Extra arguments go to `go test` after the package, for example -run Node.
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
: "${OMPIG_AOT_TEST_COMPILER:?set OMPIG_AOT_TEST_COMPILER to the Pi-Bolt runtime executable}"
work="$(mktemp -d "${TMPDIR:-/tmp}/aot-conformance.XXXXXX")"
trap 'rm -rf "$work"' EXIT
(
	cd "$root/ompiglets/aot"
	GOWORK=off OMPIG_AOT_CONFORMANCE_SLOT="$work/slot" OMPIG_AOT_TEST_RUNTIME_TREE="$root/coding/extension/host/subprocess/runtime-node" \
		go test ./extensions/aotruntime -run '^TestWriteConformanceSlot$' -count=1
)
test -x "$work/slot/aot/ompig-ts-host" || { echo "the conformance slot has no compiled host" >&2; exit 1; }
cd "$root"
# The suite falls back to Node without a word when the slot's host does not verify or start, so check that the request resolves to the compiled host.
go build -o "$work/ompig" ./cmd/ompig
mkdir -p "$work/home"
selected="$(HOME="$work/home" OMPIG_CODING_AGENT_DIR="$work/home/agent" OMPIG_TS_RUNTIME=aot OMPIG_TS_RUNTIMES_DIR="$work/slot" "$work/ompig" extensions runtime status | grep '^Selected:')"
case "$selected" in
*"aot (executable"*) ;;
*) echo "the conformance slot does not select the compiled host: $selected" >&2; exit 1 ;;
esac
OMPIG_TS_RUNTIME=aot OMPIG_TS_RUNTIMES_DIR="$work/slot" go test ./test/extension-conformance -count=1 -timeout 50m "$@"
