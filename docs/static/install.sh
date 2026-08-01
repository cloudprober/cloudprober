#!/bin/sh
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
  if command -v sha256sum >/dev/null 2>&1; then
    got=$(sha256sum "${tmpdir}/${archive}" | cut -d' ' -f1)
  elif command -v shasum >/dev/null 2>&1; then
    got=$(shasum -a 256 "${tmpdir}/${archive}" | cut -d' ' -f1)
  else
    echo "Warning: no sha256sum or shasum; skipping checksum verification" >&2
    return
  fi

  # No -S here: a 404 is an expected case that we report ourselves.
  if ! curl -fsL -o "${tmpdir}/checksums.txt" "${base_url}/cloudprober-${version}-checksums.txt"; then
    echo "Warning: ${version} has no published checksums; skipping verification" >&2
    return
  fi

  want=$(grep " ${archive}$" "${tmpdir}/checksums.txt" | cut -d' ' -f1)
  [ -n "$want" ] || err "${archive} is not listed in checksums.txt"
  [ "$got" = "$want" ] || err "checksum mismatch for ${archive}: got ${got}, want ${want}"
}

pick_install_dir() {
  install_dir="${INSTALL_DIR:-}"
  if [ -z "$install_dir" ]; then
    if [ -w /usr/local/bin ]; then
      install_dir="/usr/local/bin"
    else
      install_dir="${HOME}/.local/bin"
    fi
  fi
  mkdir -p "$install_dir" || err "couldn't create ${install_dir}"
  [ -w "$install_dir" ] || err "${install_dir} is not writable; set INSTALL_DIR or re-run with sudo"
}

command -v curl >/dev/null 2>&1 || err "curl is required"

detect_platform

version="${VERSION:-}"
[ -n "$version" ] || latest_version

base_url="${REPO_URL}/releases/download/${version}"
archive="cloudprober-${version}-${platform}.tar.gz"

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

echo "Downloading cloudprober ${version} (${platform})..."
curl -fsSL -o "${tmpdir}/${archive}" "${base_url}/${archive}" ||
  err "couldn't download ${base_url}/${archive}"

verify_checksum

tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"
binary="${tmpdir}/cloudprober-${version}-${platform}/cloudprober"
[ -f "$binary" ] || err "${archive} doesn't contain a cloudprober binary"

pick_install_dir
cp "$binary" "${install_dir}/cloudprober"
chmod 755 "${install_dir}/cloudprober"

echo "Installed cloudprober ${version} to ${install_dir}/cloudprober"
case ":${PATH}:" in
  *":${install_dir}:"*) ;;
  *) echo "Note: ${install_dir} is not in your PATH." ;;
esac
echo "Run 'cloudprober' to start, or see https://cloudprober.org/docs/overview/getting-started/"
