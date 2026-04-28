#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
OUT_DIR="${1:-${REPO_ROOT}/bin}"
SERVICE_DIR="${REPO_ROOT}/services/mach1"
WEB_DIR="${REPO_ROOT}/services/web-ui"

mkdir -p "${OUT_DIR}"

if ! command -v go >/dev/null 2>&1; then
  echo "Go toolchain not found in PATH. Install Go 1.22+ and re-run." >&2
  exit 1
fi

echo "Syncing registry-index into API embed dir..."
mkdir -p "${SERVICE_DIR}/cmd/mcpapiserver/data"
cp "${REPO_ROOT}/packages/registry-index/index.json" "${SERVICE_DIR}/cmd/mcpapiserver/data/registry-index.json"

echo "Building mach1 and CLI binaries..."
pushd "${SERVICE_DIR}" >/dev/null
go mod tidy
go run ./cmd/mach1signregistry --check --catalog "${REPO_ROOT}/packages/registry-index/index.json"
for cmd in mach1 mach1ctl mach1e2e stubmcp mcpapiserver; do
  go build -trimpath -ldflags "-s -w" -o "${OUT_DIR}/${cmd}" "./cmd/${cmd}"
done
go vet ./...
go test ./...
popd >/dev/null

if command -v npm >/dev/null 2>&1; then
  echo "Building Hub UI..."
  pushd "${WEB_DIR}" >/dev/null
  npm install
  npm run build
  if npm run | grep -q " tauri"; then
    npm run tauri build
  fi
  popd >/dev/null
else
  echo "npm not found; skipping Hub UI build." >&2
fi

echo "Done. Binaries in ${OUT_DIR}"