#!/bin/sh
# SPDX-License-Identifier: MIT
#
# Ompig installer for macOS, Linux and Android (Termux).
#
#   curl -fsSL https://ompig.dev/install.sh | sh
#
# Downloads the Ompig release archive for this machine from GitHub Releases,
# verifies it against the release's SHA256SUMS, and installs the ompig binary.
# It fails closed: no published release, a missing checksum, or a checksum
# mismatch stops the install before anything is written to the install
# directory.
#
# Environment:
#   OMPIG_VERSION        install this version (for example 0.2.0) instead of the latest
#   OMPIG_INSTALL_DIR    install directory (default: $HOME/.local/bin, or $PREFIX/bin in Termux)
#   OMPIG_API_BASE       API that names the latest release (default: https://ompig.dev/api)
#   OMPIG_LATEST_RELEASE_URL  page that redirects to the latest release, used when the API does not answer (default: https://github.com/tmih06/ompig/releases/latest)
#   OMPIG_DOWNLOAD_BASE  release download root (default: https://github.com/tmih06/ompig/releases/download)
#   OMPIG_UPDATE_URL     update manifest `ompig update` should use (default: the latest release's update.json)
#   OMPIG_HOME           Ompig's settings directory, where the install receipt goes (default: ~/.ompig)
#
# On Android (Termux) it installs the android-arm64 release, never linux-arm64,
# into $PREFIX/bin. Other Android CPUs have no release.
#
# After installing, it records an owner-only receipt (<settings>/install-receipt)
# naming the installed executable, its release, its SHA-256, and its update
# source, so `ompig update` can later replace the binary in place.
#
# The whole script is one function called on its last line, so a truncated
# download runs nothing.

set -eu

main() {
  api_base=${OMPIG_API_BASE:-https://ompig.dev/api}
  latest_release_url=${OMPIG_LATEST_RELEASE_URL:-https://github.com/tmih06/ompig/releases/latest}
  download_base=${OMPIG_DOWNLOAD_BASE:-https://github.com/tmih06/ompig/releases/download}

  need uname
  need tar
  need mktemp
  downloader=$(pick_downloader)
  platform=$(detect_platform)
  install_dir=${OMPIG_INSTALL_DIR:-$(default_install_dir)}

  version=${OMPIG_VERSION:-}
  if [ -z "$version" ]; then
    version=$(latest_version "$downloader" "$api_base" "$latest_release_url")
  fi
  version=${version#v}
  valid_version "$version" || fail "not a release version: $version"

  name="ompig-${version}-${platform}"
  archive="${name}.tar.gz"
  release_url="${download_base%/}/v${version}"

  work=$(mktemp -d 2>/dev/null || mktemp -d -t ompig-install)
  trap 'rm -rf "$work"' EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM

  say "Downloading Ompig ${version} for ${platform}"
  fetch "$downloader" "${release_url}/SHA256SUMS" "$work/SHA256SUMS" ||
    fail "Ompig ${version} has no SHA256SUMS at ${release_url}; refusing to install an unverified archive"
  fetch "$downloader" "${release_url}/${archive}" "$work/$archive" ||
    fail "could not download ${release_url}/${archive}"

  expected=$(expected_sha256 "$work/SHA256SUMS" "$archive")
  [ -n "$expected" ] || fail "SHA256SUMS has no single valid entry for ${archive}"
  actual=$(sha256_of "$work/$archive")
  [ "$actual" = "$expected" ] || fail "checksum mismatch for ${archive}: expected ${expected}, got ${actual}"
  say "Verified SHA-256 ${actual}"

  mkdir "$work/extract"
  tar -xzf "$work/$archive" -C "$work/extract"
  binary="$work/extract/${name}/ompig"
  if [ ! -f "$binary" ] || [ -L "$binary" ]; then
    fail "${archive} does not contain ${name}/ompig"
  fi

  # Stage beside the destination (a noexec /tmp cannot run the smoke test),
  # then replace any existing ompig in one rename.
  mkdir -p "$install_dir"
  staged="${install_dir}/.ompig.install.$$"
  trap 'rm -rf "$work" "$staged"' EXIT
  cp "$binary" "$staged"
  chmod 0755 "$staged"
  "$staged" --version >/dev/null 2>&1 || fail "the downloaded ompig binary does not run on this machine"
  mv -f "$staged" "${install_dir}/ompig"

  say "Installed $("${install_dir}/ompig" --version 2>/dev/null || printf 'ompig %s' "$version") to ${install_dir}/ompig"
  write_receipt "$install_dir" "$version" ||
    say "Could not record the install receipt; to update later, run this installer again"
  case ":${PATH:-}:" in
    *":${install_dir}:"*) ;;
    *) say "Add ${install_dir} to PATH to run ompig, for example: export PATH=\"${install_dir}:\$PATH\"" ;;
  esac
}

# write_receipt DIR VERSION records the standalone install for `ompig update`,
# in the format internal/codingagent/selfupdate_receipt.go reads: owner-only,
# one key=value per line.
write_receipt() {
  exe="$(cd "$1" && pwd -P)/ompig"
  source_url=${OMPIG_UPDATE_URL:-https://github.com/tmih06/ompig/releases/latest/download/update.json}
  if [ -n "${OMPIG_HOME:-}" ]; then
    case "$OMPIG_HOME" in
      \~) root=$HOME ;;
      \~/*) root="$HOME/${OMPIG_HOME#\~/}" ;;
      *) root=$OMPIG_HOME ;;
    esac
  elif [ -n "${XDG_CONFIG_HOME:-}" ]; then
    root="${XDG_CONFIG_HOME}/ompig"
  else
    root="$HOME/.ompig"
  fi
  case "${exe}${source_url}" in
    *"
"*) return 1 ;;
  esac
  sum=$(sha256_of "$exe") || return 1
  (
    umask 077
    mkdir -p "$root" &&
      printf 'kind=standalone\nexecutable=%s\npig-version=%s\nsha256=%s\nupdate-source=%s\n' \
        "$exe" "$2" "$sum" "$source_url" > "$root/.install-receipt.$$" &&
      mv -f "$root/.install-receipt.$$" "$root/install-receipt"
  ) 2>/dev/null || {
    rm -f "$root/.install-receipt.$$"
    return 1
  }
}

say() {
  printf 'ompig-install: %s\n' "$*"
}

fail() {
  printf 'ompig-install: error: %s\n' "$*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || fail "this installer needs '$1'"
}

pick_downloader() {
  if command -v curl >/dev/null 2>&1; then
    echo curl
  elif command -v wget >/dev/null 2>&1; then
    echo wget
  else
    fail "this installer needs curl or wget"
  fi
}

# fetch DOWNLOADER URL FILE: HTTPS only, failing on any HTTP error.
fetch() {
  case "$1" in
    curl) curl --proto '=https' --tlsv1.2 -fsSL --retry 3 -o "$3" "$2" ;;
    wget) wget --https-only -q -O "$3" "$2" ;;
  esac
}

# default_install_dir prints where ompig goes without OMPIG_INSTALL_DIR: Termux's
# bin directory, which is on PATH and has no /usr/local, on Android, and
# ~/.local/bin elsewhere.
default_install_dir() {
  if is_android && [ -n "${PREFIX:-}" ]; then
    echo "${PREFIX}/bin"
    return
  fi
  echo "${HOME:?HOME is not set}/.local/bin"
}

# is_android reports Android's userland (Termux): uname -s says Linux and
# uname -o says Android. BSD and macOS uname has no -o.
is_android() {
  [ "$(uname -o 2>/dev/null)" = Android ]
}

# detect_platform prints OS-ARCH as release archives name them. Termux reports
# uname -s Linux, but it needs the android build: that one is a position
# independent executable linked to bionic, which Termux's loader runs and which
# resolves DNS and trusts certificates through Android.
detect_platform() {
  case "$(uname -s)" in
    Linux)
      if is_android; then os=android; else os=linux; fi
      ;;
    Darwin) os=darwin ;;
    *) fail "unsupported operating system $(uname -s); download a release archive from GitHub instead" ;;
  esac
  case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    arm64 | aarch64) arch=arm64 ;;
    *) fail "unsupported CPU architecture $(uname -m)" ;;
  esac
  if is_android && [ "$arch" != arm64 ]; then
    fail "Ompig publishes Android binaries for arm64 only, not $(uname -m)"
  fi
  # A Rosetta 2 shell on Apple silicon reports x86_64; install the native binary.
  if [ "$os" = darwin ] && [ "$arch" = amd64 ] &&
    [ "$(sysctl -n sysctl.proc_translated 2>/dev/null || echo 0)" = 1 ]; then
    arch=arm64
  fi
  echo "${os}-${arch}"
}

# latest_version DOWNLOADER API_BASE LATEST_RELEASE_URL asks the site API, then GitHub's latest-release redirect, which is not subject to the API's rate limit. The archive is verified against SHA256SUMS either way.
latest_version() {
  latest=
  if response=$(fetch_stdout "$1" "${2%/}/latest-version"); then
    latest=$(printf '%s' "$response" | tr -d '\n\r' | sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
  fi
  if [ -z "$latest" ]; then
    latest=$(redirect_tag "$1" "$3")
    [ -n "$latest" ] ||
      fail "no Ompig release is published yet (${2%/}/latest-version did not answer and $3 named no release); build from source or set OMPIG_VERSION"
    say "${2%/}/latest-version did not name a release; using the latest on GitHub (${latest})" >&2
  fi
  echo "$latest"
}

# redirect_tag DOWNLOADER URL prints the tag URL redirects to (.../releases/tag/<tag>) without following the redirect, or nothing.
redirect_tag() {
  case "$1" in
    curl) location=$(curl --proto '=https' --tlsv1.2 -sS --retry 3 -o /dev/null -w '%{redirect_url}' "$2" 2>/dev/null) || location= ;;
    # Wget indents response headers; its unindented Location diagnostic appends "[following]" even when redirects are disabled.
    wget) location=$(wget --https-only --max-redirect=0 -S -O /dev/null "$2" 2>&1 | sed -n 's/^  *[Ll]ocation: *//p' | tr -d '\r' | tail -n 1) ;;
  esac
  printf '%s\n' "$location" | sed -n 's|.*/releases/tag/\([^/?#]*\)$|\1|p' | sed 's/%2[Bb]/+/g'
}

fetch_stdout() {
  case "$1" in
    curl) curl --proto '=https' --tlsv1.2 -fsSL --retry 3 "$2" ;;
    wget) wget --https-only -q -O - "$2" ;;
  esac
}

# valid_version VERSION: SemVer core with optional pre-release and build parts.
valid_version() {
  printf '%s\n' "$1" | grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'
}

# expected_sha256 SUMS NAME prints the digest of NAME (listed as NAME or ./NAME)
# only when exactly one well-formed line names it.
expected_sha256() {
  awk -v name="$2" '
    ($2 == name || $2 == "./" name || $2 == "*" name || $2 == "*./" name) && NF == 2 {
      count++
      digest = tolower($1)
    }
    END {
      if (count == 1 && digest ~ /^[0-9a-f]+$/ && length(digest) == 64) print digest
    }
  ' "$1"
}

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print tolower($1)}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print tolower($1)}'
  elif command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 -r "$1" | awk '{print tolower($1)}'
  else
    fail "this installer needs sha256sum, shasum, or openssl to verify the download"
  fi
}

main "$@"
