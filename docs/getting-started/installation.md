# Installation

Genifest can be installed using several methods. Choose the one that works best for your environment.

## Quick Install (Recommended)

The fastest way to install Genifest is using the install script:

```bash
curl -L https://raw.githubusercontent.com/zostay/genifest/master/tools/install.sh | sh
```

This script will:

- Detect your operating system and architecture
- Download the latest release binary
- Install it to `/usr/local/bin/genifest`
- Make it executable

## Manual Installation

### Download Pre-built Binaries

Download the latest release for your platform from the [GitHub releases page](https://github.com/zostay/genifest/releases):

=== "Linux (amd64)"

    ```bash
    curl -L https://github.com/zostay/genifest/releases/latest/download/genifest-linux-amd64 -o genifest
    chmod +x genifest
    sudo mv genifest /usr/local/bin/
    ```

=== "Linux (arm64)"

    ```bash
    curl -L https://github.com/zostay/genifest/releases/latest/download/genifest-linux-arm64 -o genifest
    chmod +x genifest
    sudo mv genifest /usr/local/bin/
    ```

=== "macOS (Intel)"

    ```bash
    curl -L https://github.com/zostay/genifest/releases/latest/download/genifest-darwin-amd64 -o genifest
    chmod +x genifest
    sudo mv genifest /usr/local/bin/
    ```

=== "macOS (Apple Silicon)"

    ```bash
    curl -L https://github.com/zostay/genifest/releases/latest/download/genifest-darwin-arm64 -o genifest
    chmod +x genifest
    sudo mv genifest /usr/local/bin/
    ```

### Install from Source

If you have Go 1.22+ installed, you can install from source:

```bash
go install github.com/zostay/genifest/cmd/genifest@latest
```

This will install the latest version to your `$GOPATH/bin` directory.

### Build from Source

For development or to build a specific version:

```bash
# Clone the repository
git clone https://github.com/zostay/genifest.git
cd genifest

# Build using Make (recommended)
make build

# Or build manually
go build -o genifest .
```

## Verification

After installation, verify that Genifest is working correctly:

```bash
# Check version
genifest version

# Show help
genifest --help
```

## System Requirements

- **Operating System**: Linux or macOS
- **Architecture**: amd64 (x86_64) or arm64 (aarch64)
- **Disk Space**: ~20MB for the binary
- **Memory**: Minimal (typically <100MB during operation)

_Windows is not currently supported, but support could be added if there's interest._

## Troubleshooting

### Permission Denied

If you get a "permission denied" error on macOS:

```bash
# Remove quarantine attribute
sudo xattr -d com.apple.quarantine /usr/local/bin/genifest
```

### Command Not Found

Ensure `/usr/local/bin` is in your PATH:

```bash
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

## Updating

To update Genifest to the latest version, simply re-run the installation command:

```bash
curl -L https://raw.githubusercontent.com/zostay/genifest/master/tools/install.sh | sh
```

Or manually download and replace the binary from the releases page.

## Uninstalling

To remove Genifest:

```bash
sudo rm /usr/local/bin/genifest
```

---

Next: [Quick Start Guide →](quickstart.md)