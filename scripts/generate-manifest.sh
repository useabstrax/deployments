#!/usr/bin/env bash
set -euo pipefail

BINARY="abstrax-deploy"
PLUGIN_NAME="deploy"
PUBLISHER="useabstrax"
TRUST_LEVEL="official"
DESCRIPTION="Zero-downtime GitHub deployments for Abstrax projects"
DISPLAY_NAME="Deploy Plugin"
REQUIRES_ABSTRAX=">=0.1.0"
PROTOCOL_VERSION="1"
CHANNEL="stable"
REPO="${GITHUB_REPOSITORY:-useabstrax/abstrax}"

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')}"
TAG="${TAG:-v${VERSION}}"
RELEASE_BASE="${RELEASE_BASE:-https://github.com/${REPO}/releases/download/${TAG}}"

DIST_DIR="${DIST_DIR:-dist}"
mkdir -p "${DIST_DIR}"

checksums_file="${DIST_DIR}/${BINARY}_${VERSION}_checksums.txt"
: > "${checksums_file}"

platforms=("linux-amd64:linux_amd64" "linux-arm64:linux_arm64")
declare -A PLATFORM_SIZES=()

for entry in "${platforms[@]}"; do
  platform="${entry%%:*}"
  archive_suffix="${entry##*:}"
  src="${DIST_DIR}/${BINARY}-${platform}"
  archive="${DIST_DIR}/${BINARY}_${VERSION}_${archive_suffix}.tar.gz"

  if [[ ! -f "${src}" ]]; then
    echo "missing binary: ${src}" >&2
    exit 1
  fi

  tmpdir=$(mktemp -d)
  cp "${src}" "${tmpdir}/${BINARY}"
  tar -C "${tmpdir}" -czf "${archive}" "${BINARY}"
  rm -rf "${tmpdir}"

  sha=$(shasum -a 256 "${archive}" | awk '{print $1}')
  size=$(wc -c < "${src}" | tr -d ' ')
  PLATFORM_SIZES["${platform}"]="${size}"
  echo "${sha}  $(basename "${archive}")" >> "${checksums_file}"
done

amd64_sha=$(grep "linux_amd64" "${checksums_file}" | awk '{print $1}')
arm64_sha=$(grep "linux_arm64" "${checksums_file}" | awk '{print $1}')

cat > plugin-manifest.json <<EOF
{
  "name": "${PLUGIN_NAME}",
  "version": "${VERSION}",
  "protocol_version": ${PROTOCOL_VERSION},
  "requires_abstrax": "${REQUIRES_ABSTRAX}",
  "channel": "${CHANNEL}",
  "publisher": "${PUBLISHER}",
  "trust_level": "${TRUST_LEVEL}",
  "description": "${DESCRIPTION}",
  "display_name": "${DISPLAY_NAME}",
  "platforms": {
    "linux-amd64": {
      "url": "${RELEASE_BASE}/${BINARY}_${VERSION}_linux_amd64.tar.gz",
      "sha256": "${amd64_sha}",
      "size": ${PLATFORM_SIZES["linux-amd64"]}
    },
    "linux-arm64": {
      "url": "${RELEASE_BASE}/${BINARY}_${VERSION}_linux_arm64.tar.gz",
      "sha256": "${arm64_sha}",
      "size": ${PLATFORM_SIZES["linux-arm64"]}
    }
  }
}
EOF

echo "Wrote plugin-manifest.json"
