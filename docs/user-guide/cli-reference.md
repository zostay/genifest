# CLI Reference

Complete reference for the Genifest command-line interface.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Commands Overview

Genifest uses a subcommand-based architecture:

```bash
genifest <command> [options] [directory]
```

## Available Commands

### `genifest run`

Apply configuration changes to manifest files.

```bash
genifest run [directory] [flags]
```

**Flags:**

- `-i, --include-tags` - Include only changes with these tags
- `-x, --exclude-tags` - Exclude changes with these tags

### `genifest validate`

Validate configuration without applying changes.

```bash
genifest validate [directory]
```

### `genifest tags`

List all available tags in the configuration.

```bash
genifest tags [directory]
```

### `genifest config`

Display the merged configuration in YAML format.

```bash
genifest config [directory]
```

### `genifest version`

Show version information.

```bash
genifest version
```

## Global Options

- `--help` - Show help for any command
- `--version` - Show version information

## Examples

See the [Quick Start Guide](../getting-started/quickstart.md) for practical examples.