#!/bin/bash
set -euo pipefail

VERSION="0.12.5"
TAG="v${VERSION}"

sudo apt-get update
sudo apt-get install -y curl ninja-build gettext cmake unzip build-essential ripgrep

curl -fsL "https://github.com/neovim/neovim/archive/refs/tags/${TAG}.tar.gz" -o "/tmp/neovim-${TAG}.tar.gz"
tar -xzf "/tmp/neovim-${TAG}.tar.gz" -C /tmp
rm "/tmp/neovim-${TAG}.tar.gz"

cd "/tmp/neovim-${VERSION}"

make CMAKE_BUILD_TYPE=Release
sudo make CMAKE_INSTALL_PREFIX=/usr/local/nvim install

sudo ln -sf /usr/local/nvim/bin/nvim /usr/local/bin/nvim

cd /
sudo rm -rf "/tmp/neovim-${VERSION}"
