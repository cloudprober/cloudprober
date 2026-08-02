#!/bin/sh
#
# Copyright 2026 The Cloudprober Authors.
# Licensed under the Apache License, Version 2.0:
# https://github.com/cloudprober/cloudprober/blob/main/LICENSE
#
# Install script for cloudprober (https://cloudprober.org).
#
# Usage:
#   curl -fsSL https://cloudprober.org/install.sh | sh
#
# Environment variables:
#   VERSION      Release to install, e.g. v0.14.5. Defaults to the latest
#                release. Use "tip" for the build from the main branch.
#   INSTALL_DIR  Where to put the binary. Defaults to /usr/local/bin, falling
#                back to $HOME/.local/bin if that's not writable.

set -eu

REPO_URL="https://github.com/cloudprober/cloudprober"

err() {
  echo "install.sh: $*" >&2
  exit 1
}

# Map uname output to the platform names used in release archives.
detect_platform() {
  os=$(uname -s)
  case "$os" in
    Linux) os="linux" ;;
    Darwin) os="macos" ;;
    *) err "unsupported OS '$os'; see ${REPO_URL}/releases for other options" ;;
  esac

  arch=$(uname -m)
  case "$arch" in
    x86_64 | amd64) arch="x86_64" ;;
    aarch64 | arm64) arch="arm64" ;;
    armv7l | armv7) arch="armv7" ;;
    *) err "unsupported architecture '$arch'; see ${REPO_URL}/releases" ;;
  esac

  platform="${os}-${arch}"
}

# /releases/latest redirects to /releases/tag/<version>. Reading that redirect
# avoids the GitHub API's unauthenticated rate limit.
latest_version() {
  url=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "${REPO_URL}/releases/latest") ||
    err "couldn't reach GitHub to find the latest version"
  version="${url##*/}"
  case "$version" in
    v*) ;;
    *) err "couldn't parse a version out of '$url'" ;;
  esac
}

verify_checksum() {
  got=$($sha256_cmd "${tmpdir}/${archive}" | cut -d' ' -f1)

  curl -fsSL -o "${tmpdir}/checksums.txt" "${base_url}/cloudprober-${version}-checksums.txt" ||
    err "couldn't download checksums for ${version}; refusing to install unverified"

  # Exact string compare on the filename field: $archive embeds $version, which
  # comes from the environment, so a regex match could pick the wrong line.
  want=$(awk -v a="$archive" '$2 == a { print $1; exit }' "${tmpdir}/checksums.txt")
  [ -n "$want" ] || err "${archive} is not listed in checksums.txt"
  [ "$got" = "$want" ] || err "checksum mismatch for ${archive}: got ${got}, want ${want}"
}

pick_install_dir() {
  install_dir="${INSTALL_DIR:-}"
  if [ -z "$install_dir" ]; then
    if [ -w /usr/local/bin ]; then
      install_dir="/usr/local/bin"
    elif [ -n "${HOME:-}" ]; then
      install_dir="${HOME}/.local/bin"
    else
      err "/usr/local/bin isn't writable and HOME isn't set; set INSTALL_DIR"
    fi
  fi
  mkdir -p "$install_dir" || err "couldn't create ${install_dir}"
  [ -w "$install_dir" ] || err "${install_dir} is not writable; set INSTALL_DIR or re-run with sudo"
}

command -v curl >/dev/null 2>&1 || err "curl is required"
command -v tar >/dev/null 2>&1 || err "tar is required"

# Resolve the hashing tool up front rather than after the download: macOS ships
# shasum, not sha256sum.
if command -v sha256sum >/dev/null 2>&1; then
  sha256_cmd="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  sha256_cmd="shasum -a 256"
else
  err "need sha256sum or shasum to verify the download; install either one, or grab a release manually from ${REPO_URL}/releases"
fi

detect_platform

version="${VERSION:-}"
[ -n "$version" ] || latest_version

base_url="${REPO_URL}/releases/download/${version}"
archive="cloudprober-${version}-${platform}.tar.gz"

tmpdir=$(mktemp -d)
tmpbin=""
cleanup() {
  rm -rf "$tmpdir"
  [ -z "$tmpbin" ] || rm -f "$tmpbin"
}
trap cleanup EXIT
# dash doesn't run the EXIT trap when the shell is killed by a signal, so Ctrl-C
# would otherwise leave the temp dir (and possibly a half-copied binary) behind.
trap 'cleanup; exit 130' INT
trap 'cleanup; exit 143' TERM HUP

echo "Downloading cloudprober ${version} (${platform})..."
curl -fsSL -o "${tmpdir}/${archive}" "${base_url}/${archive}" ||
  err "couldn't download ${base_url}/${archive}"

verify_checksum

tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"
# The archive holds a single directory, but its name tracks the build version
# rather than $version ("tip" archives are renamed copies of versioned ones),
# so find the binary instead of assuming the directory name.
binary=""
for f in "$tmpdir"/*/cloudprober; do
  if [ -f "$f" ]; then
    binary="$f"
    break
  fi
done
[ -n "$binary" ] || err "${archive} doesn't contain a cloudprober binary"

pick_install_dir
# Install by rename: an interrupted copy can't leave a half-written binary in
# place, and replacing a cloudprober that's currently running won't fail with
# ETXTBSY.
tmpbin="${install_dir}/.cloudprober.$$"
cp "$binary" "$tmpbin"
chmod 755 "$tmpbin"
mv "$tmpbin" "${install_dir}/cloudprober"
tmpbin=""

echo "Installed cloudprober ${version} to ${install_dir}/cloudprober"
case ":${PATH}:" in
  *":${install_dir}:"*) ;;
  *) echo "Note: ${install_dir} is not in your PATH." ;;
esac
echo "Run 'cloudprober' to start, or see https://cloudprober.org/docs/overview/getting-started/"
