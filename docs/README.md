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

## Documentation Structure

The documentation is organized into logical sections to guide users from initial installation through advanced usage:

### Getting Started (`getting-started/`)

Entry-level documentation for new users:

- **`installation.md`** - Installation methods including Go install, binary downloads, and from source
- **`quickstart.md`** - Basic 10-minute tutorial using the guestbook example
- **`configuration.md`** - Overview of configuration concepts and file structure

### User Guide (`user-guide/`)

Comprehensive documentation for daily usage:

- **`concepts.md`** - Core concepts: configurations, changes, ValueFrom expressions, functions, and tag filtering
- **`cli-reference.md`** - Complete command-line interface documentation with all subcommands and options
- **`configuration.md`** - Detailed configuration file structure and advanced features
- **`value-generation.md`** - In-depth guide to ValueFrom expressions and all supported types
- **`tag-filtering.md`** - Tag-based change filtering system with glob patterns and logical operators

### Examples (`examples/`)

Practical tutorials and real-world scenarios:

- **`guestbook.md`** - Complete step-by-step tutorial using the Kubernetes guestbook application
- **`patterns.md`** - Common usage patterns, best practices, and configuration recipes
- **`gitops.md`** - Integration with GitOps workflows, CI/CD pipelines, and deployment strategies

### Reference (`reference/`)

Technical specifications and API documentation:

- **`schema.md`** - Complete YAML configuration schema with all fields and validation rules
- **`keyselectors.md`** - YAML path selection syntax for targeting specific values in manifests
- **`valuefrom.md`** - Comprehensive reference for all ValueFrom types with examples
- **`functions.md`** - Function definition syntax, parameter handling, and scoping rules
- **`troubleshooting.md`** - Common issues, error messages, and debugging techniques

### Development (`development/`)

Contributor-focused technical documentation:

- **`contributing.md`** - How to contribute: setup, testing, pull requests, and coding standards
- **`architecture.md`** - Technical architecture, design decisions, and internal APIs
- **`testing.md`** - Testing strategies, integration tests, and validation approaches
- **`releases.md`** - Release process, versioning strategy, and deployment procedures

### Additional Files

- **`index.md`** - Documentation site homepage with project overview and navigation
- **`changelog.md`** - Version history, breaking changes, and release notes
- **`includes/mkdocs.md`** - Shared content snippets included across multiple pages
- **`assets/`** - Images, logos, diagrams, and other static resources

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

## Documentation Guidelines

When contributing to the documentation:

### Content Organization

- **Getting Started**: Focus on user onboarding and quick wins. Keep content beginner-friendly.
- **User Guide**: Provide comprehensive how-to information for daily usage. Include practical examples.
- **Examples**: Create step-by-step tutorials with real-world scenarios. Use the guestbook example as a foundation.
- **Reference**: Maintain complete technical specifications. Keep API documentation up-to-date.
- **Development**: Document technical architecture and contributor workflows.

### Writing Style

- Use clear, concise language appropriate for technical audiences
- Include code examples with proper syntax highlighting (yaml, bash, go)
- Provide both conceptual explanations and practical examples
- Cross-reference related concepts using internal links
- Use consistent terminology throughout all documentation

### Material for MkDocs Features

The documentation site uses advanced MkDocs features:

- **Admonitions**: Use `!!! note`, `!!! warning`, `!!! tip` for callouts
- **Code blocks**: Include language hints for syntax highlighting
- **Tabbed content**: Use `=== "Tab Name"` for multi-option examples
- **Snippets**: Reference shared content with `--8<-- "includes/mkdocs.md"`
- **Mermaid diagrams**: Include architecture and flow diagrams
- **Search**: All content is automatically indexed and searchable

### Testing Documentation

Always test documentation changes locally:

```bash
# Install dependencies
make docs-install

# Serve with live reload
make docs-serve

# Build and check for errors
make docs-build
```

### Cross-References and Navigation

- Link to related sections using relative paths: `[Configuration](../user-guide/configuration.md)`
- Update navigation in `mkdocs.yml` when adding new pages
- Ensure all internal links work correctly
- Use descriptive link text that provides context

## Contributing

1. **Edit content** in the `docs/` directory
2. **Test locally** with `make docs-serve` to verify formatting and links
3. **Check build** with `make docs-build` to catch any errors
4. **Submit PR** - documentation builds are tested automatically
5. **Deploy** happens automatically on merge to master

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