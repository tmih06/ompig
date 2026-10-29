# SPDX-License-Identifier: MIT

ARG GO_BUILDER=cgr.dev/chainguard/go:latest-dev@sha256:72f9b391f511f88e7154aaaa5ee1e27fb6594575d9a406940f49786b30172803
ARG WOLFI_BASE=cgr.dev/chainguard/wolfi-base:latest@sha256:81211e15a86ea1d5357f284482af45730c51de45eaf10ad9ce205d6bfc2df323

# GO_VERSION must equal .go-version; make compliance checks it.
ARG GO_VERSION=1.27.3

FROM ${GO_BUILDER} AS go-toolchain
ARG TARGETARCH
ARG GO_VERSION
ARG GO_SHA256_AMD64=65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40
ARG GO_SHA256_ARM64=a06e99309dbcbdca1a8ccd2282c3ee146e55930886e26b4f1193269587165f49
RUN case "${TARGETARCH}" in \
      amd64) checksum="${GO_SHA256_AMD64}" ;; \
      arm64) checksum="${GO_SHA256_ARM64}" ;; \
      *) echo "unsupported Go target architecture: ${TARGETARCH}" >&2; exit 1 ;; \
    esac \
    && wget -q "https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz" -O /tmp/go.tar.gz \
    && echo "${checksum}  /tmp/go.tar.gz" | sha256sum -c - \
    && mkdir -p /out \
    && tar -C /out -xzf /tmp/go.tar.gz \
    && test "$(/out/go/bin/go env GOVERSION)" = "go${GO_VERSION}"

# The package index's current versions of the locked packages. Only the scheduled refresh builds this stage
# (RESOLVE_APK_LOCK=1 automation/ci/build-ci-image.sh); it is not part of the image, and the build below installs the lock exactly.
FROM ${WOLFI_BASE} AS apk-resolve
USER root
COPY automation/images/ci-parity/packages.lock /tmp/packages.lock
RUN apk update \
  && cut -d= -f1 /tmp/packages.lock | xargs apk add --no-cache --upgrade \
  && awk 'BEGIN { RS=""; FS="\n" } { p=""; v=""; for (i=1; i<=NF; i++) { if ($i ~ /^P:/) p=substr($i,3); if ($i ~ /^V:/) v=substr($i,3) } if (p != "") print p "=" v }' /lib/apk/db/installed | LC_ALL=C sort > /tmp/resolved.lock

FROM scratch AS apk-resolved
COPY --from=apk-resolve /tmp/resolved.lock /packages.lock

FROM ${WOLFI_BASE} AS toolchain-base
COPY automation/images/ci-parity/packages.lock /tmp/packages.lock
RUN apk update \
  && xargs apk add --no-cache < /tmp/packages.lock \
  && awk 'BEGIN { RS=""; FS="\n" } { p=""; v=""; for (i=1; i<=NF; i++) { if ($i ~ /^P:/) p=substr($i,3); if ($i ~ /^V:/) v=substr($i,3) } if (p != "") print p "=" v }' /lib/apk/db/installed | LC_ALL=C sort > /tmp/installed.lock \
  && cmp /tmp/packages.lock /tmp/installed.lock \
  && rm /tmp/packages.lock /tmp/installed.lock

FROM toolchain-base AS pi-oracle
# PI_VERSION must equal the pin's package version and PI_COMMIT the pin's commit when the pin is a commit
# (empty for a release tag); `make pin-sync` writes both and make compliance checks them.
ARG PI_VERSION=1.4.0
ARG PI_COMMIT=6c384aa30f619d7b28963f2bab24d8dacbe73cb6
COPY automation/images/ci-parity/package.json automation/images/ci-parity/package-lock.json /opt/ompig/pi/
RUN test -z "${PI_COMMIT}" || { echo "this image installs the published Pi ${PI_VERSION}; it cannot be the build of Pi commit ${PI_COMMIT} yet (a commit pin needs the oracle built from source, automation/pin/oracle.py)" >&2; exit 1; } \
  && npm ci --prefix /opt/ompig/pi --omit=dev --ignore-scripts --no-audit --no-fund \
  && mkdir -p /opt/ompig/pi/bin \
  && ln -s ../node_modules/.bin/pi /opt/ompig/pi/bin/pi \
  && test "$(/opt/ompig/pi/bin/pi --version)" = "${PI_VERSION}" \
  && node -e 'const [want, ...packages] = process.argv.slice(1); for (const p of packages) { if (require(`/opt/ompig/pi/node_modules/@earendil-works/${p}/package.json`).version !== want) process.exit(1) }' "${PI_VERSION}" pi-agent-core pi-ai pi-coding-agent pi-tui \
  && coding_root=/opt/ompig/pi/node_modules/@earendil-works/pi-coding-agent \
  && rm -rf /opt/ompig/pi/node_modules/@earendil-works/pi-ai/node_modules/@types/node \
  && mkdir -p "${coding_root}/node_modules/@earendil-works" "${coding_root}/node_modules/@types" \
  && ln -s /opt/ompig/pi/node_modules/@types/node "${coding_root}/node_modules/@types/node" \
  && for package in pi-agent-core pi-ai pi-tui; do \
       ln -s "/opt/ompig/pi/node_modules/@earendil-works/${package}" "${coding_root}/node_modules/@earendil-works/${package}"; \
     done

FROM toolchain-base AS npm-runtime
COPY automation/images/npm-runtime/package.json automation/images/npm-runtime/package-lock.json /opt/ompig/npm/
RUN npm ci --prefix /opt/ompig/npm --omit=dev --ignore-scripts --no-audit --no-fund \
  && rm -rf \
    /opt/ompig/npm/node_modules/npm/node_modules/brace-expansion \
    /opt/ompig/npm/node_modules/npm/node_modules/http-cache-semantics \
    /opt/ompig/npm/node_modules/npm/node_modules/ip-address \
    /opt/ompig/npm/node_modules/npm/node_modules/postcss-selector-parser \
    /opt/ompig/npm/node_modules/npm/node_modules/tar \
    /opt/ompig/npm/node_modules/npm/node_modules/undici \
  && test "$(node /opt/ompig/npm/node_modules/npm/bin/npm-cli.js --version)" = "12.1.0"

FROM toolchain-base
ARG GO_VERSION
USER root
RUN mkdir -p /usr/lib /opt/ompig /workspace \
  && ln -sf /usr/bin/fd /usr/local/bin/fdfind \
  && chown -R nonroot:nonroot /opt/ompig /workspace
COPY --from=go-toolchain /out/go /usr/lib/go
COPY --from=pi-oracle /opt/ompig/pi /opt/ompig/pi
COPY --from=npm-runtime /opt/ompig/npm /opt/ompig/npm
RUN ln -s /opt/ompig/npm/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
  && ln -s /opt/ompig/npm/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx \
  && test "$(/usr/lib/go/bin/go env GOVERSION)" = "go${GO_VERSION}"

ENV GOCACHE=/tmp/ompig-go-build
ENV GOMODCACHE=/tmp/ompig-go-mod
ENV GOPATH=/tmp/ompig-go
ENV CARGO_HOME=/tmp/ompig-cargo
ENV PATH="/usr/lib/go/bin:/opt/ompig/pi/bin:${PATH}"
ENV OMPIG_PARITY_PI_BIN=/opt/ompig/pi/bin/pi
# This image installs the PUBLISHED Pi, not a build of the pinned source, so it opts into the published oracle explicitly (a release-tag pin only; the runner refuses it for a commit pin). Building Pi from source in the image is the follow-up that removes this line.
ENV OMPIG_PARITY_PI_PUBLISHED=1
ENV PI_PACKAGE_ROOT=/opt/ompig/pi/node_modules/@earendil-works/pi-coding-agent

USER nonroot
WORKDIR /workspace
ENTRYPOINT []
CMD ["bash", "-lc", "go version && node --version && npm --version && python3 --version && rustc --version && cargo --version && pi --version && tmux -V"]
