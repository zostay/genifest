# Genifest Documentation

This directory contains the source files for the Genifest documentation site, built with [Material for MkDocs](https://squidfunk.github.io/mkdocs-material/).

## Quick Start

### Prerequisites

- Python 3.11 or later
- pip package manager

### Setup

1. **Install dependencies:**
   ```bash
   make docs-install
   # or manually:
   pip install -r requirements.txt
   ```

2. **Serve locally:**
   ```bash
   make docs-serve
   # or manually:
   mkdocs serve
   ```

3. **Open browser:** http://localhost:8000

### Building

```bash
# Build static site
make docs-build

# Clean build artifacts
make docs-clean
```

## Structure

```
docs/
├── index.md                    # Homepage
├── getting-started/            # Installation & quickstart
├── user-guide/                 # Core documentation
├── examples/                   # Tutorials & examples
├── reference/                  # API & configuration reference
├── development/                # Contributor documentation
└── assets/                     # Images, logos, etc.
```

## Deployment

The documentation is automatically deployed to GitHub Pages at [genifest.qubling.com](https://genifest.qubling.com) when changes are pushed to the master branch.

### Manual Deployment

```bash
make docs-deploy
```

## Custom Domain Setup

To complete the custom domain setup for `genifest.qubling.com`:

1. **DNS Configuration** (external):
   ```
   genifest.qubling.com. IN CNAME zostay.github.io.
   ```

2. **GitHub Pages Settings**:
   - Go to repository Settings → Pages
   - Set custom domain to `genifest.qubling.com`
   - Enable "Enforce HTTPS"

The `docs/CNAME` file ensures the custom domain is preserved during deployments.

## Features

- **Material Design 3** theme with dark/light mode
- **Search** with highlighting and suggestions
- **Mermaid diagrams** for architecture illustrations
- **Code syntax highlighting** with copy buttons
- **Responsive design** for all devices
- **Git integration** for edit links and revision dates

## Contributing

1. **Edit content** in the `docs/` directory
2. **Test locally** with `make docs-serve`
3. **Submit PR** - documentation builds are tested automatically
4. **Deploy** happens automatically on merge to master

## Troubleshooting

### Dependencies Not Installing

```bash
# Upgrade pip
python -m pip install --upgrade pip

# Install from requirements.txt
pip install -r requirements.txt
```

### Build Failures

```bash
# Check for syntax errors
mkdocs build --strict

# Common issues:
# - Missing files referenced in nav
# - Invalid YAML front matter
# - Broken internal links
```

### Local Server Issues

```bash
# Kill existing process
pkill -f "mkdocs serve"

# Serve on different port
mkdocs serve --dev-addr=127.0.0.1:8001
```

## Resources

- [MkDocs Documentation](https://www.mkdocs.org/)
- [Material for MkDocs](https://squidfunk.github.io/mkdocs-material/)
- [Markdown Guide](https://www.markdownguide.org/)
- [Mermaid Diagrams](https://mermaid-js.github.io/mermaid/)