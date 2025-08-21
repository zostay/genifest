# CLI Reference

Complete reference for the Genifest command-line interface.

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

- `-i, --include-tags strings` - Include only changes with these tags (supports glob patterns)
- `-x, --exclude-tags strings` - Exclude changes with these tags (supports glob patterns)

**Tag Filtering Logic:**

- **No flags**: All changes applied (tagged and untagged)  
- **Include only**: Only changes matching include patterns applied
- **Exclude only**: All changes except those matching exclude patterns applied
- **Both flags**: Changes matching include but not exclude patterns applied
- **Glob patterns**: Supports patterns like `prod*`, `test-*`, `*-staging`

**Enhanced Output:**

The run command provides detailed progress reporting:

- Configuration summary with total change definitions and tag filtering results
- Individual change tracking: `file -> document[index] -> keySelector: old → new`
- Clear distinction between changes applied vs actual file modifications
- Final summary with file modification counts and emoji indicators

**Examples:**

```bash
# Apply all changes in current directory
genifest run

# Apply changes from specific directory
genifest run path/to/project

# Apply only production-tagged changes
genifest run --include-tags production

# Apply all changes except staging
genifest run --exclude-tags staging

# Apply changes matching multiple patterns  
genifest run --include-tags prod*,release-*

# Apply production changes but exclude secrets
genifest run --include-tags production --exclude-tags secrets
```

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