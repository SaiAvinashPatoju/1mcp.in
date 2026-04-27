#!/usr/bin/env bash
set -euo pipefail

OWNER="${ONEMCP_GITHUB_OWNER:-SaiAvinashPatoju}"
REPO="${ONEMCP_GITHUB_REPO:-1mcp.in}"
VERSION="${ONEMCP_VERSION:-latest}"
INSTALL_DIR="${ONEMCP_INSTALL_DIR:-$HOME/.onemcp/bin}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "${OS}" in
  darwin) OS="darwin" ;;
  linux) OS="linux" ;;
  *) echo "Unsupported OS: ${OS}" >&2; exit 1 ;;
esac
case "${ARCH}" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: ${ARCH}" >&2; exit 1 ;;
esac

if [ "${VERSION}" = "latest" ]; then
  API_URL="https://api.github.com/repos/${OWNER}/${REPO}/releases/latest"
else
  API_URL="https://api.github.com/repos/${OWNER}/${REPO}/releases/tags/${VERSION}"
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi

mkdir -p "${INSTALL_DIR}"
ASSET="onemcp-${OS}-${ARCH}.tar.gz"
URL="$(curl -fsSL "${API_URL}" | sed -n "s/.*\"browser_download_url\": \"\([^\"]*${ASSET}\)\".*/\1/p" | head -1)"
if [ -z "${URL}" ]; then
  echo "Could not find release asset ${ASSET}" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT
curl -fsSL "${URL}" -o "${TMP_DIR}/${ASSET}"
tar -xzf "${TMP_DIR}/${ASSET}" -C "${INSTALL_DIR}"
chmod +x "${INSTALL_DIR}/centralmcpd" "${INSTALL_DIR}/onemcpctl" 2>/dev/null || true

case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    SHELL_RC="${HOME}/.profile"
    if [ -n "${SHELL:-}" ] && [ "$(basename "${SHELL}")" = "zsh" ]; then SHELL_RC="${HOME}/.zshrc"; fi
    printf '\nexport PATH="%s:$PATH"\n' "${INSTALL_DIR}" >> "${SHELL_RC}"
    echo "Added ${INSTALL_DIR} to PATH in ${SHELL_RC}"
    ;;
esac

echo "1mcp installed in ${INSTALL_DIR}"
echo "Run \`onemcpctl start\` to launch 1mcp"