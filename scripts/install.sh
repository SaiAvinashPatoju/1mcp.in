#!/usr/bin/env bash
set -euo pipefail

# 1. Check Node.js
if ! command -v node >/dev/null 2>&1; then
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    if [ "$OS" = "darwin" ]; then
        echo "Node.js not found. Installing via Homebrew..."
        if ! command -v brew >/dev/null 2>&1; then
            echo "Homebrew not found. Please install Homebrew or Node manually."
            exit 1
        fi
        brew install node
    else
        echo "Node.js not found. Please install Node.js manually, or via nvm/apt/yum etc."
    fi
fi

# 2. Check uv
if ! command -v uv >/dev/null 2>&1; then
    echo "uv not found. Installing via astral.sh..."
    curl -LsSf https://astral.sh/uv/install.sh | sh
    export PATH="$HOME/.local/bin:$PATH"
fi

OWNER="${MACH1_GITHUB_OWNER:-SaiAvinashPatoju}"
REPO="${MACH1_GITHUB_REPO:-1mcp.in}"
VERSION="${MACH1_VERSION:-latest}"
INSTALL_DIR="${MACH1_INSTALL_DIR:-$HOME/.mach1/bin}"

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
ASSET="mach1-${OS}-${ARCH}.tar.gz"
URL="$(curl -fsSL "${API_URL}" | sed -n "s/.*\"browser_download_url\": \"\([^\"]*${ASSET}\)\".*/\1/p" | head -1)"
if [ -z "${URL}" ]; then
  echo "Could not find release asset ${ASSET}" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT
curl -fsSL "${URL}" -o "${TMP_DIR}/${ASSET}"
tar -xzf "${TMP_DIR}/${ASSET}" -C "${INSTALL_DIR}"
chmod +x "${INSTALL_DIR}/mach1" "${INSTALL_DIR}/mach1ctl" 2>/dev/null || true

case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    SHELL_RC="${HOME}/.profile"
    if [ -n "${SHELL:-}" ] && [ "$(basename "${SHELL}")" = "zsh" ]; then SHELL_RC="${HOME}/.zshrc"; fi
    printf '\nexport PATH="%s:$PATH"\n' "${INSTALL_DIR}" >> "${SHELL_RC}"
    echo "Added ${INSTALL_DIR} to PATH in ${SHELL_RC}"
    ;;
esac

export PATH="${INSTALL_DIR}:$PATH"

# 4. Inject Rules
echo "Injecting rule files for AI clients..."
if command -v mach1ctl >/dev/null 2>&1; then
    mach1ctl inject-rules vscode || true
    mach1ctl inject-rules cursor || true
    mach1ctl inject-rules windsurf || true
else
    "${INSTALL_DIR}/mach1ctl" inject-rules vscode || true
    "${INSTALL_DIR}/mach1ctl" inject-rules cursor || true
    "${INSTALL_DIR}/mach1ctl" inject-rules windsurf || true
fi

echo "1mcp.in installed in ${INSTALL_DIR}"
echo "Run \`mach1ctl start\` to launch 1mcp.in"