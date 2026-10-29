#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OMPIG_BIN="${OMPIG_PORTER_SMOKE_BIN:-$ROOT/bin/ompig-porter-smoke}"
cd "$ROOT"

mkdir -p "$(dirname "$OMPIG_BIN")"
CGO_ENABLED=0 go build -buildvcs=false -trimpath -o "$OMPIG_BIN" ./cmd/ompig
command -v tmux >/dev/null || { echo "ompig-porter: smoke test requires tmux" >&2; exit 1; }

session="ompig-porter-smoke-$$"
home="$(mktemp -d "${TMPDIR:-/tmp}/ompig.XXXXXXXX")"
pane="$(mktemp "${TMPDIR:-/tmp}/ompig.XXXXXXXX")"
# A tmux server of its own, so the run never touches the caller's sessions.
tm() { tmux -L "$session" -f /dev/null "$@"; }
ompig_pid=""
cleanup() {
  tm kill-session -t "$session" 2>/dev/null || true
  # Killing the session ends ompig through its terminal-loss exit, which still writes exit.log and the session under $home: wait for ompig first.
  if [ -n "$ompig_pid" ]; then
    for _ in $(seq 1 100); do
      kill -0 "$ompig_pid" 2>/dev/null || break
      sleep 0.1
    done
  fi
  rm -rf "$home" "$pane" || echo "ompig-porter: could not remove $home" >&2
}
trap cleanup EXIT

mkdir -p "$home/agent"
printf '%s\n' '{"defaultProjectTrust":"always"}' >"$home/agent/settings.json"

# The isolated HOME must not move the caller's Go caches and settings: the Ompiglet's Go extension is built with them, and a
# module cache under $home is read-only, so the cleanup could not remove it.
go_env="GOCACHE='$(go env GOCACHE)' GOMODCACHE='$(go env GOMODCACHE)' GOPATH='$(go env GOPATH)' GOENV='$(go env GOENV)'"
# exec at each step makes the pane process ompig itself (run-ompig-porter.sh execs it), so the cleanup can wait for it.
tm new-session -d -s "$session" -x 140 -y 50 \
  "cd '$ROOT' && exec env -u HTTPS_PROXY -u HTTP_PROXY -u https_proxy -u http_proxy $go_env HOME='$home' OMPIG_HOME='$home' OMPIG_CODING_AGENT_DIR='$home/agent' PI_CODING_AGENT_DIR='$home/agent' OMPIG_BIN='$OMPIG_BIN' '$ROOT/automation/porter/run-ompig-porter.sh' --verbose"
ompig_pid="$(tm display-message -p -t "$session" '#{pane_pid}' 2>/dev/null || true)"

ready=false
for _ in $(seq 1 240); do
  tm capture-pane -pt "$session" -S - >"$pane" 2>/dev/null || true
  if grep -q 'Trust project folder?' "$pane"; then
    cat "$pane" >&2
    echo "ompig-porter: isolated smoke unexpectedly requested project trust" >&2
    exit 1
  fi
  if grep -q 'warning: extension "ompig-porter"' "$pane"; then
    cat "$pane" >&2
    echo "ompig-porter: extension failed to load" >&2
    exit 1
  fi
  # The startup header lists each loaded resource under [Extensions] and [Skills] once the TUI is up.
  if grep -q '^\[Extensions\]' "$pane" && grep -q 'ompiglets/porter/extensions/ompig-porter$' "$pane"; then
    ready=true
    break
  fi
  sleep 0.25
done
[ "$ready" = true ] || { cat "$pane" >&2; echo "ompig-porter: did not reach the TUI" >&2; exit 1; }

grep -q 'ompiglets/porter/extensions/ompig-porter$' "$pane" || {
  cat "$pane" >&2
  echo "ompig-porter: selected extension is not active" >&2
  exit 1
}
grep -q 'ompiglets/porter/skills/ompig-porter/SKILL.md$' "$pane" || {
  cat "$pane" >&2
  echo "ompig-porter: selected skill is not active" >&2
  exit 1
}

tm send-keys -t "$session" -l -- '/ompig-porter {"operation":"disposition-plan"}'
tm send-keys -t "$session" Enter
operation_ok=false
for _ in $(seq 1 240); do
  tm capture-pane -pt "$session" -S - >"$pane" 2>/dev/null || true
  if grep -q 'No model selected' "$pane"; then
    cat "$pane" >&2
    echo "ompig-porter: extension command was handled as a model prompt" >&2
    exit 1
  fi
  # The plan is thousands of lines and the TUI clears its scrollback when it redraws, so only the tail of the JSON is on the
  # pane: every group ends with these three members, which no error message prints.
  if grep -q '"portMapStatus"' "$pane" && grep -q '"recommendation"' "$pane" && grep -q '"decision"' "$pane"; then
    operation_ok=true
    break
  fi
  sleep 0.25
done
[ "$operation_ok" = true ] || {
  cat "$pane" >&2
  echo "ompig-porter: disposition-plan did not return its deterministic report" >&2
  exit 1
}

echo "ompig-porter smoke: Ompiglet extension, skill, and read-only disposition plan verified"
