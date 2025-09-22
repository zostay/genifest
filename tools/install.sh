#!/bin/sh

# Genifest Installation Script
# This script downloads and installs the latest release of genifest to /usr/local/bin

echo "Installing genifest..."

INSTALL_DIR=/usr/local/bin
echo "Installation directory: $INSTALL_DIR"

# Get the latest release version from GitHub API
echo "Fetching latest release information..."
CURRENT_VERSION=$(curl -s https://api.github.com/repos/zostay/genifest/releases/latest | grep tag_name | cut -d '"' -f 4 | cut -c 2-)
echo "Latest version: $CURRENT_VERSION"

# Detect the operating system and architecture
OS_NAME=$(uname -s | tr '[:upper:]' '[:lower:]')
OS_ARCH=$(uname -m)
echo "Detected platform: $OS_NAME-$OS_ARCH"

# Construct the download URL for the appropriate binary
DOWNLOAD_URL="https://github.com/zostay/genifest/releases/download/v$CURRENT_VERSION/genifest-$CURRENT_VERSION-$OS_NAME-$OS_ARCH"
echo "Download URL: $DOWNLOAD_URL"

# Download and install the binary
# Note: sudo is required to write to /usr/local/bin (system directory)
echo "Downloading and installing genifest (sudo required for system directory access)..."
sudo sh -c "curl -L \"$DOWNLOAD_URL\" -o \"$INSTALL_DIR/genifest\" && chmod +x \"$INSTALL_DIR/genifest\""

echo "Installation complete! You can now run 'genifest' from anywhere."
