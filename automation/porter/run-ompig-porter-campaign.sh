#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OMPIG_ROOT="$ROOT"
OMPIG_BIN="${OMPIG_BIN:-$OMPIG_ROOT/bin/ompig-parity}"
MAX_JOBS="${OMPIG_PORTER_MAX_JOBS:-3}"

usage() {
  cat <<'EOF'
Usage: automation/porter/run-ompig-porter-campaign.sh <inventory|verify> <family> [family...]

Runs bounded ompig-porter jobs concurrently against disposable snapshots of the
current commit. Workers cannot modify the integration workspace. Their only
output is one report and status file per family under tmp/porter-campaigns.

Environment:
  OMPIG_PORTER_MAX_JOBS   Maximum concurrent workers (default: 3)
  OMPIG_PORTER_MODEL      Explicit model passed to every worker
  OMPIG_PORTER_REPORT_DIR Override report directory
  OMPIG_BIN               Ompig executable (default: bin/ompig-parity)
EOF
}

if [ "$#" -lt 2 ]; then
  usage >&2
  exit 2
fi
mode="$1"
shift
case "$mode" in
  inventory|verify) ;;
  *) echo "ompig-porter campaign: mode must be inventory or verify" >&2; exit 2 ;;
esac
case "$MAX_JOBS" in
  ''|*[!0-9]*) echo "ompig-porter campaign: OMPIG_PORTER_MAX_JOBS must be a positive integer" >&2; exit 2 ;;
esac
[ "$MAX_JOBS" -gt 0 ] || { echo "ompig-porter campaign: OMPIG_PORTER_MAX_JOBS must be positive" >&2; exit 2; }
[ -x "$OMPIG_BIN" ] || { echo "ompig-porter campaign: Ompig binary is not executable: $OMPIG_BIN" >&2; exit 1; }

# shellcheck source=automation/porter/hermetic-agent.sh
. "$ROOT/automation/porter/hermetic-agent.sh"

cd "$ROOT"
if [ -n "$(git status --porcelain --untracked-files=no)" ]; then
  echo "ompig-porter campaign: tracked workspace changes exist; commit them before creating an immutable campaign" >&2
  exit 1
fi
commit="$(git rev-parse HEAD)"
short_commit="$(git rev-parse --short HEAD)"
version="$(python3 "$OMPIG_ROOT/automation/pin/pin.py" get version)"
[ -n "$version" ] || { echo "ompig-porter campaign: cannot resolve coding.UpstreamVersion" >&2; exit 1; }
upstream="$OMPIG_ROOT/.upstream/v$version"
[ -d "$upstream" ] || { echo "ompig-porter campaign: exact upstream mirror is missing: $upstream" >&2; exit 1; }

families=()
declare -A seen=()
for family in "$@"; do
  [ -z "${seen[$family]:-}" ] || { echo "ompig-porter campaign: duplicate family $family" >&2; exit 2; }
  seen[$family]=1
  (cd "$OMPIG_ROOT" && go run ./test/parity/cmd/familygaps -family "$family" >/dev/null)
  families+=("$family")
done

stamp="$(date -u +%Y%m%dT%H%M%SZ)"
report_dir="${OMPIG_PORTER_REPORT_DIR:-$OMPIG_ROOT/tmp/porter-campaigns/$stamp-$short_commit}"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/ompig-porter-campaign.XXXXXX")"
mkdir -p "$report_dir"
chmod 700 "$work_dir" "$report_dir"
cleanup() {
  chmod -R u+rwX "$work_dir" 2>/dev/null || true
  rm -rf "$work_dir"
}
trap cleanup EXIT

prepare_snapshot() {
  local family="$1"
  local snapshot="$work_dir/$family/source"
  mkdir -p "$snapshot"
  git archive "$commit" | tar -x -C "$snapshot"
  mkdir -p "$snapshot/.upstream"
  cp -R "$upstream" "$snapshot/.upstream/v$version"
  ln -s "v$version" "$snapshot/.upstream/current"
  # Campaign workers are read-only at the filesystem boundary. Keep only
  # disposable build/test output writable; source, tests, evidence, ledgers,
  # and Ompiglet inputs cannot be changed even if a worker ignores its prompt.
  mkdir -p "$snapshot/tmp"
  chmod -R a-w "$snapshot"
  chmod -R u+rwX "$snapshot/tmp"
}

prepare_home() {
  local family="$1"
  local home="$work_dir/$family/home"
  mkdir -p "$home"
  chmod 700 "$home"
  porter_hermetic_agent "$home/agent"
}

run_worker() {
  local family="$1"
  local snapshot="$work_dir/$family/source"
  local home="$work_dir/$family/home"
  local report="$report_dir/$family.log"
  local status="$report_dir/$family.status"
  local profile="$snapshot/ompiglets/porter/ompig-porter.yaml"
  local campaign_families
  campaign_families="$(IFS=,; printf '%s' "${families[*]}")"
  local model_args=()
  if [ -n "${OMPIG_PORTER_MODEL:-}" ]; then
    model_args=(--model "$OMPIG_PORTER_MODEL")
  fi
  {
    printf 'campaign_commit=%s\nupstream_version=%s\nmode=%s\nfamily=%s\n\n' "$commit" "$version" "$mode" "$family"
    cd "$snapshot"
    OMPIG_HOME="$home" OMPIG_CODING_AGENT_DIR="$home/agent" PI_CODING_AGENT_DIR="$home/agent" "$OMPIG_BIN" --ompiglet "$profile" --no-session "${model_args[@]}" -p "$(cat <<EOF
You are the read-only $family worker in a parallel Ompig Porter campaign.
Campaign commit: $commit
Exact upstream: $version
Mode: $mode
All sibling assignments: $campaign_families
Maximum concurrent workers: $MAX_JOBS

Other workers are active in isolated snapshots. Do not edit repository source,
tests, evidence, generated inventories, or git state. Do not kill or signal any
process or tmux session, remove temporary state, use broad process matching, or
run destructive git commands. Build tools may create their normal ephemeral
outputs inside this disposable snapshot.

Apply AGENTS.md's "Faithful, general implementations" rule and the Ompig Porter Skill: Pi's shared data-driven paths, no uncited provider/model special cases, and evidence across applicable OAuth, API-key, custom-base-URL OpenAI-compatible, and no-default-model shapes. Copilot and test-faux alone are insufficient. Report sibling defects without editing this read-only snapshot. Refuse completion for a provider-specific branch without a Pi citation or a shared path verified with a single provider; report missing proof as a blocker.

Fix root causes, not failures hidden by timeouts, retries, sleeps, skips, or normalization. Retries and timeouts are only for surfaced faults outside our control.

Be patient with every build, test, extraction, and oracle probe. Wait for its
declared timeout. Do not duplicate, prematurely terminate, or repeatedly retry
a quiet or slow command. If an operation times out or is externally blocked,
record the exact command and blocker once, then continue only with independent
read-only analysis.

/skill:ompig-porter $mode $family
EOF
)"
  } >"$report" 2>&1
  printf 'pass\n' >"$status"
}

pids=()
pid_families=()
failures=0
wait_worker() {
  local pid="$1"
  local family="$2"
  if wait "$pid"; then
    printf 'ompig-porter campaign: PASS %s\n' "$family"
  else
    printf 'fail\n' >"$report_dir/$family.status"
    printf 'ompig-porter campaign: FAIL %s (see %s)\n' "$family" "$report_dir/$family.log" >&2
    failures=$((failures + 1))
  fi
}

for family in "${families[@]}"; do
  prepare_snapshot "$family"
  prepare_home "$family"
  run_worker "$family" &
  pids+=("$!")
  pid_families+=("$family")
  if [ "${#pids[@]}" -ge "$MAX_JOBS" ]; then
    wait_worker "${pids[0]}" "${pid_families[0]}"
    pids=("${pids[@]:1}")
    pid_families=("${pid_families[@]:1}")
  fi
done
for index in "${!pids[@]}"; do
  wait_worker "${pids[$index]}" "${pid_families[$index]}"
done

printf 'ompig-porter campaign: reports=%s commit=%s workers=%d failures=%d\n' "$report_dir" "$short_commit" "${#families[@]}" "$failures"
[ "$failures" -eq 0 ]
