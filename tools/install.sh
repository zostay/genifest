#!/bin/sh

INSTALL_DIR=/usr/local/bin
CURRENT_VERSION=$(curl -s https://api.github.com/repos/zostay/genifest/releases/latest | grep tag_name | cut -d '"' -f 4)
OS_NAME=$(uname -s | tr '[:upper:]' '[:lower:]')
OS_ARCH=$(uname -m)
DOWNLOAD_URL="https://github.com/zostay/genifest/releases/download/$CURRENT_VERSION/genifest-$CURRENT_VERSION-$OS_NAME-$OS_ARCH"

curl -L "$DOWNLOAD_URL" -o "$INSTALL_DIR/genifest"
