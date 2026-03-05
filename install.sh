#!/usr/bin/env bash

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

BIN_NAME="lazycider"
PROJECT_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
BUILD_TARGET="./cmd/lazycider"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

echo -e "${BLUE}lazycider installer${NC}"
echo "-----------------------------------"

if ! command -v go >/dev/null 2>&1; then
  echo -e "${RED}Error: Go is not installed.${NC}"
  echo "Please install Go 1.25+ and try again."
  exit 1
fi

GO_VERSION_RAW="$(go env GOVERSION 2>/dev/null || true)"
if [[ -n "${GO_VERSION_RAW}" ]]; then
  GO_VERSION_NUM="${GO_VERSION_RAW#go}"
  GO_MAJOR="${GO_VERSION_NUM%%.*}"
  GO_REST="${GO_VERSION_NUM#*.}"
  GO_MINOR="${GO_REST%%.*}"
  if [[ "${GO_MAJOR}" =~ ^[0-9]+$ ]] && [[ "${GO_MINOR}" =~ ^[0-9]+$ ]]; then
    if (( GO_MAJOR < 1 || (GO_MAJOR == 1 && GO_MINOR < 25) )); then
      echo -e "${RED}Error: Go ${GO_VERSION_RAW} detected.${NC}"
      echo "lazycider requires Go 1.25 or newer."
      exit 1
    fi
  fi
fi

cd "${PROJECT_ROOT}"

echo "Building ${BIN_NAME}..."
go build -o "${BIN_NAME}" "${BUILD_TARGET}"

if [[ ! -f "${BIN_NAME}" ]]; then
  echo -e "${RED}Build failed: binary not found.${NC}"
  exit 1
fi

echo -e "${GREEN}Build completed.${NC}"
echo "Installing to ${INSTALL_DIR}..."

if [[ ! -d "${INSTALL_DIR}" ]]; then
  echo -e "${YELLOW}Install directory does not exist. Creating ${INSTALL_DIR}.${NC}"
  if mkdir -p "${INSTALL_DIR}" 2>/dev/null; then
    :
  else
    sudo mkdir -p "${INSTALL_DIR}"
  fi
fi

if [[ -w "${INSTALL_DIR}" ]]; then
  mv "${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"
  chmod 755 "${INSTALL_DIR}/${BIN_NAME}"
else
  echo -e "${YELLOW}Administrator permissions are required for ${INSTALL_DIR}.${NC}"
  sudo mv "${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"
  sudo chmod 755 "${INSTALL_DIR}/${BIN_NAME}"
fi

echo "-----------------------------------"
echo -e "${GREEN}Installation completed successfully.${NC}"
echo "Run: ${BLUE}${BIN_NAME}${NC}"
