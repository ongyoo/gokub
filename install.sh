#!/usr/bin/env bash

set -e

REPO="${GOKUB_REPOSITORY:-ongyoo/gokub}"
VERSION="${GOKUB_VERSION:-latest}"

echo "Installing GOKUB from $REPO"

OS=$(uname | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH=amd64 ;;
    arm64|aarch64) ARCH=arm64 ;;
esac

DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/gokub-${OS}-${ARCH}.tar.gz"

curl -L "$DOWNLOAD_URL" -o gokub.tar.gz

tar -xzf gokub.tar.gz

sudo mv gokub /usr/local/bin/

echo "Installed!"