# syntax=docker/dockerfile:1.7
# The Ompig benchmark image for bench/k8s: Ompig's Durable builds (TinyGo and Go Wasm cores, the native Go Durable
# benchmark), Ompig itself, Node, pi-durable and Pi's coding agent at Ompig's Pi pin (the build context pi, which
# `k8sbench pi-packages` writes: the published packages for a release pin, the pinned source's packs with its frozen
# closures for a commit pin; `k8sbench pi-packages -check` checks the installed versions and the coding agent against
# k8sbench's pins), and the Pi
# reference checkout pinned in durable/contract/PIN. Build it with bench/k8s/image/build.sh, which passes the commit and prints the digest-pinned
# reference the runner needs. Every download is pinned by version and sha256; the base images are pinned by digest.
ARG GO_IMAGE=golang:1.27.3-bookworm@sha256:5c3e9b43cfd3bb95c7937462e29d87ada57b481a96fa23ad0eb61778d7a00110
ARG RUNTIME_IMAGE=debian:bookworm-slim@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587

FROM ${GO_IMAGE} AS build
ARG TARGETARCH
ARG NODE_VERSION=24.19.0
ARG NODE_SHA256_AMD64=f625d97cd707df4ff96254916fbc5ff014f09c09effe5a1e0ca8f6d41a8789d4
ARG NODE_SHA256_ARM64=d28c8a5bf0a808f0ed434a1dce8c54ae98f0371c0bd86ac58abc613f73e6643f
ARG TINYGO_VERSION=0.42.0
ARG TINYGO_SHA256_AMD64=b87688fa2e19cee7d813cad7fd7dadb71dff3198e47125aba66ba4af5e490438
ARG TINYGO_SHA256_ARM64=f2f3f863e63728b87772e7306bd6030df6e6c89773a7082eb0122113cf9e453d
ARG CPYTHON_RELEASE=20261001
ARG CPYTHON_VERSION=3.12.15
ARG CPYTHON_SHA256_AMD64=7bb1659e3235077b7f63d5b6eb6ce653c6fcd6c5041e9d5f73b42ce10421464d
ARG CPYTHON_SHA256_ARM64=0b35f4dc08d58534eb82e024989e2db9873885dccd4f2a316ff55c0cec146123
RUN set -eu; \
	case "${TARGETARCH:-amd64}" in \
		amd64) nodearch=x64; nodesum=$NODE_SHA256_AMD64; tgsum=$TINYGO_SHA256_AMD64; pyarch=x86_64; pysum=$CPYTHON_SHA256_AMD64 ;; \
		arm64) nodearch=arm64; nodesum=$NODE_SHA256_ARM64; tgsum=$TINYGO_SHA256_ARM64; pyarch=aarch64; pysum=$CPYTHON_SHA256_ARM64 ;; \
		*) echo "unsupported architecture ${TARGETARCH}" >&2; exit 1 ;; \
	esac; \
	curl -fsSLo /tmp/node.tar.gz "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-${nodearch}.tar.gz"; \
	echo "${nodesum}  /tmp/node.tar.gz" | sha256sum -c -; \
	mkdir -p /usr/local/node; tar -xzf /tmp/node.tar.gz -C /usr/local/node --strip-components=1; \
	curl -fsSLo /tmp/tinygo.tar.gz "https://github.com/tinygo-org/tinygo/releases/download/v${TINYGO_VERSION}/tinygo${TINYGO_VERSION}.linux-${TARGETARCH:-amd64}.tar.gz"; \
	echo "${tgsum}  /tmp/tinygo.tar.gz" | sha256sum -c -; \
	tar -xzf /tmp/tinygo.tar.gz -C /usr/local; \
	curl -fsSLo /tmp/cpython.tar.gz "https://github.com/astral-sh/python-build-standalone/releases/download/${CPYTHON_RELEASE}/cpython-${CPYTHON_VERSION}%2B${CPYTHON_RELEASE}-${pyarch}-unknown-linux-gnu-install_only_stripped.tar.gz"; \
	echo "${pysum}  /tmp/cpython.tar.gz" | sha256sum -c -; \
	mkdir -p /opt/ompig; tar -xzf /tmp/cpython.tar.gz -C /opt/ompig --no-same-owner; \
	rm /tmp/node.tar.gz /tmp/tinygo.tar.gz /tmp/cpython.tar.gz
ENV PATH=/usr/local/node/bin:/usr/local/tinygo/bin:$PATH \
	GOTOOLCHAIN=local GOWORK=/src/go.work GOFLAGS=-buildvcs=false CGO_ENABLED=0
WORKDIR /src
COPY . .
ARG OMPIG_COMMIT
# ARMS: comma-separated ids of bench/evals/arms.lock (such as codex) installed under arms/<id>, as k8sbench bundle -arms does.
ARG ARMS=
# Built outputs go under /opt/ompig: bin/ (k8sbench, durableperf, ompig, sessionscale, the core's native bench when the source has one),
# dist/ (core-go.wasm, core-tinygo.wasm), pi-durable/ (durable/interop/bench.mjs beside the locked reference
# packages), pi-agent/ (Pi's coding agent at the pin), python/ (the pinned standalone CPython), pigbench.py and
# evals/pigbench/ (bench/evals' harness, which the agent profiles run), arms/<id>/ (the ARMS), COMMIT,
# VERSIONS and ARTIFACTS.sha256.
RUN --mount=type=cache,target=/root/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build --mount=type=bind,from=pi,target=/pi set -eu; \
	test -n "${OMPIG_COMMIT}" || { echo "OMPIG_COMMIT is required (bench/k8s/image/build.sh sets it)" >&2; exit 1; }; \
	mkdir -p /opt/ompig/bin /opt/ompig/dist /opt/ompig/pi-durable /opt/ompig/pi-agent /opt/ompig/evals/pigbench; \
	cp bench/k8s/image/pigbench.py /opt/ompig/pigbench.py; cp bench/evals/pigbench/*.py /opt/ompig/evals/pigbench/; cp bench/evals/arms.toml bench/evals/arms.lock /opt/ompig/evals/; \
	go build -trimpath -o /opt/ompig/bin/k8sbench ./bench/k8s/cmd/k8sbench; \
	go build -trimpath -o /opt/ompig/bin/durableperf ./bench/durableperf; \
	go build -trimpath -o /opt/ompig/bin/ompig ./cmd/ompig; \
	go build -trimpath -o /opt/ompig/bin/sessionscale ./bench/sessionscale/cmd/sessionscale; \
	if [ -f durable/core/Makefile ]; then \
		make -C durable/core build-go build-tinygo; \
	else \
		sh durable/core/budget/build-wasm.sh go durable/core/cmd/corewasm durable/core/host/cf/dist/core-go.wasm; \
		sh durable/core/budget/build-wasm.sh tinygo durable/core/cmd/corewasm durable/core/host/cf/dist/core-tinygo.wasm; \
	fi; \
	cp durable/core/host/cf/dist/core-go.wasm durable/core/host/cf/dist/core-tinygo.wasm /opt/ompig/dist/; \
	if [ -d durable/core/sqlhost/cmd/bench ]; then go build -trimpath -o /opt/ompig/bin/sqlhost-bench ./durable/core/sqlhost/cmd/bench; fi; \
	cp /pi/pi-durable/package.json /pi/pi-durable/package-lock.json durable/interop/bench.mjs /opt/ompig/pi-durable/; \
	if [ -d /pi/pi-durable/packs ]; then cp -r /pi/pi-durable/packs /opt/ompig/pi-durable/; fi; \
	(cd /opt/ompig/pi-durable && npm ci --ignore-scripts --no-audit --no-fund --loglevel=error); \
	cp /pi/pi-agent/package.json /pi/pi-agent/package-lock.json /opt/ompig/pi-agent/; \
	if [ -d /pi/pi-agent/packs ]; then cp -r /pi/pi-agent/packs /opt/ompig/pi-agent/; fi; \
	(cd /opt/ompig/pi-agent && npm ci --ignore-scripts --no-audit --no-fund --loglevel=error); \
	/opt/ompig/bin/k8sbench pi-packages -check /opt/ompig; \
	: > /tmp/arm-versions; \
	if [ -n "${ARMS}" ]; then /opt/ompig/bin/k8sbench arms -root /src -out /opt/ompig -arch "${TARGETARCH:-amd64}" $(echo "${ARMS}" | tr , ' ') > /tmp/arm-versions; fi; \
	echo "${OMPIG_COMMIT}" > /opt/ompig/COMMIT; \
	{ echo "go=$(go env GOVERSION)"; echo "tinygo=$(tinygo version | awk '{print $3}')"; echo "node=$(node --version)"; \
	  echo "pi-durable=$(cat /pi/pi-durable/VERSION)"; \
	  echo "pi-reference=$(awk '$1 == "main" { print $2 }' durable/contract/PIN)"; echo "arch=${TARGETARCH:-amd64}"; \
	  echo "builder=bench/k8s/image/Dockerfile"; \
	  echo "pi-agent=$(cat /pi/pi-agent/VERSION)"; \
	  echo "python=${CPYTHON_VERSION}+${CPYTHON_RELEASE}"; cat /tmp/arm-versions; } > /opt/ompig/VERSIONS; \
	(cd /opt/ompig && export LC_ALL=C && sha256sum bin/* dist/* pi-durable/bench.mjs pi-durable/package-lock.json $(ls pi-durable/packs/*.tgz 2>/dev/null) \
		pi-agent/package-lock.json $(ls pi-agent/packs/*.tgz 2>/dev/null) pi-agent/node_modules/@earendil-works/pi-coding-agent/dist/bundle/cli.js python/bin/python3.12 \
		pigbench.py evals/pigbench/*.py evals/arms.toml evals/arms.lock \
		$(for d in arms/*/; do [ -f "${d}BIN" ] || continue; [ ! -f "${d}package-lock.json" ] || printf '%s ' "${d}package-lock.json"; printf '%s ' "${d}$(cat "${d}BIN")"; done)) > /opt/ompig/ARTIFACTS.sha256

# The Pi reference at the commit durable/contract/PIN pins, with its dependencies, for profiles that run it.
# PI_REFERENCE=0 leaves it out.
ARG PI_REFERENCE=1
RUN set -eu; mkdir -p /opt/pi; \
	if [ "${PI_REFERENCE}" = 1 ]; then DURABLE_CONTRACT_CACHE=/opt/pi sh durable/contract/tools/setup.sh; fi

# The artifact bundle for k8sbench -bundle (build.sh with BUNDLE=DIR exports it to a local directory): the same files
# as /opt/ompig, run on the official Node image, so a cluster without a registry needs no custom image.
FROM scratch AS bundle
COPY --from=build /opt/ompig /

FROM ${RUNTIME_IMAGE}
COPY --from=build /usr/local/node /usr/local/node
COPY --from=build /opt/ompig /opt/ompig
COPY --from=build /opt/pi /opt/pi
# taskset comes from util-linux, which the Debian base includes.
RUN command -v taskset >/dev/null
ENV PATH=/opt/ompig/bin:/usr/local/node/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
USER 65532:65532
WORKDIR /data
