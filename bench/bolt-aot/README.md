# bench/bolt-aot: TypeScript extension runtimes, Node vs Bun vs the compiled aot host

Full-path measurement of Ompig's three TypeScript extension runtimes on one set of five extensions: `@tintinweb/pi-subagents` (pinned in `exts/package-lock.json`), Pi's `todo.ts` and `permission-gate.ts` examples (from `.upstream/current`), a heavy `tool_call` handler (`exts/heavy-toolcall.ts`) and Pi-Bolt's `word-count` example (`exts/word-count.ts`, copied unchanged under its MIT notice). The spike's set (`@juicesharp/rpiv-todo`, `pi-mcp-adapter`) is not here because the Ompiglet refuses it: those packages read files next to `import.meta.url`, which a compiled executable cannot reproduce, so they run on Bun. `bench.py` starts `ompig --mode rpc` with the set and a scripted local model (`fake_model.py`), interleaves the arms and rotates their order each round, and reports medians with ranges (`summarize.py`): launch to ready, extension memory at ready and after 5 prompts, the `/words` hot loop on a 16 MB file, the tool turn gap and the prompt wall time. Each run checks that the extension process is the tier the arm names, so an arm that fell back cannot report a number.

The harness derives from the Pi-Bolt hosting spike's harness (its `bench/bolt-host/` directory), reduced to the tiers Ompig ships. Every number it produces comes from a loaded shared host: report the host load beside it and do not publish one.

## Run

```sh
(cd bench/bolt-aot/exts && npm ci --ignore-scripts)
go build -o /tmp/ompig ./cmd/ompig
mkdir -p /tmp/slot/bun && cp "$BUN_1_4_2" /tmp/slot/bun/bun   # stock Bun 1.4.2 and its runtime.json, as /bun-runtime install writes them
python3 bench/bolt-aot/bench.py --out /tmp/bench --rounds 7 --arms node,bun,aot \
  --ompig /tmp/ompig --slot /tmp/slot --exts bench/bolt-aot/exts \
  --build-cmd 'cd ompiglets/aot && GOWORK=off OMPIG_AOT_BUILD_SLOT=/tmp/slot go test ./extensions/aotruntime -run TestBuildSlotRequests -count=1'
python3 bench/bolt-aot/summarize.py /tmp/bench/runs.jsonl
```

`--build-cmd` runs the Ompiglet's worker over the slot: first to install the aot manifest and download and verify the compiler (from `OMPIG_AOT_RELEASE_URL` when set), then, after a launch on Bun recorded the set's build request, to build it. The harness then launches on the compiled host.

## Compile a listed set ahead of time (compiled-in shape)

`build_set.py` compiles a listed set of extension files into the aot host before any measurement, from a slot that holds stock Bun (`<slot>/bun/{bun,runtime.json}`):

```sh
OMPIG_AOT_RELEASE_URL=<mirror, optional> python3 bench/bolt-aot/build_set.py --ompig /tmp/ompig --slot /tmp/slot /abs/path/bench-ext-01.ts ... /abs/path/bench-ext-10.ts
```

It runs the Ompiglet's worker (installs the manifest, downloads and verifies the Pi-Bolt compiler), records the set with one launch, builds it, and checks a second launch runs `ompig-ts-host`. Afterwards `OMPIG_TS_RUNTIME=aot OMPIG_TS_RUNTIMES_DIR=/tmp/slot ompig ... -e <the same files>` runs `<slot>/aot/images/<key>-<id>/ompig-ts-host` with the factories compiled in: no Bun or Node starts the extensions and the sources are not loaded (a same-size edit with the mtime restored still runs the compiled code, Node runs the edit). A set's key is its members' names and entry paths, so run from the paths it was built from. `ext10_bench.py` measures this arm (`aot`) against the dynamic host (`aot-dyn`, built by `TestWriteConformanceSlot`) and Node and Bun.

### One Ompiglet Binary with Go-fused and compiled TypeScript extensions

A Ompiglet Binary can carry Go-fused members, TypeScript extensions (Node members whose sources the Binary embeds and extracts under its `OMPIG_HOME`) and the aot-runtime extension together. List the TypeScript extensions as directories (`dir/index.ts`; a single-file origin fails `ompig ompiglet build` at the lock step, coding/pigletbuild/records.go `localGoReplacementDirs`), and give every fused member one name for its yaml entry, its `sdk.New` registration and its directory. Then:

```sh
OMPIG_SOURCE_ROOT=<Ompig checkout> ompig ompiglet build combo.yaml --format binary --out ./ompig-combo
mkdir -p /tmp/slot/bun && cp "$BUN_1_4_2" /tmp/slot/bun/bun   # and its runtime.json
python3 bench/bolt-aot/build_set.py --binary ./ompig-combo --slot /tmp/slot --home /tmp/combo-home
OMPIG_TS_RUNTIME=aot OMPIG_TS_RUNTIMES_DIR=/tmp/slot OMPIG_HOME=/tmp/combo-home/.ompig ./ompig-combo ...
```

The Binary's own fused worker builds the image (about 20 s for ten extensions); the image lives in the slot, not in the Binary, and its set key hashes the extracted source paths, so measure with the same `--home`. Needs the network (or `OMPIG_AOT_RELEASE_URL`) for the compiler: leave `OMPIG_OFFLINE` unset.

## Evaluation arms

`bench/compare` and `ompig-evals` pass their environment to the Ompig arms (`bench/evals/pigbench/arms.py` starts each arm with `dict(os.environ, …)`), so the aot tier is an environment setting, not a new arm:

```sh
export OMPIG_TS_RUNTIME=aot OMPIG_TS_RUNTIMES_DIR=/tmp/slot   # a slot built as above, for the extension set the arm loads
make bench-compare                                        # or the ompig-evals command of bench/evals/README.md
```

Compiled images are per extension set, so an arm that loads another set first launches on Bun and records a build request; run the Ompiglet's worker over the slot (`--build-cmd` above) and launch again. An arm whose set the Ompiglet refuses (files read next to `import.meta.url`) runs on Bun, and `ompig extensions runtime status` names the reason. Check `ext_exes` in a run's memory snapshot, as `bench.py` does, before reading a number as an aot number.
