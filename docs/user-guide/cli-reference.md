# CLI Reference

Complete reference for the Genifest command-line interface.

## Commands Overview

Genifest uses a subcommand-based architecture:

```bash
genifest <command> [options] [directory]
```

## Available Commands

### `genifest run`

Apply configuration changes to manifest files using the groups-based tag selection system.

```bash
genifest run [group] [directory] [flags]
```

**Argument Structure:**

Genifest uses intelligent argument parsing to determine your intent:

- **Zero arguments**: `genifest run`
  - Uses "all" group in current directory

- **One argument**: `genifest run <arg>`
  - If `<arg>` is a defined group name: Uses that group in current directory
  - If `<arg>` is a directory path: Uses "all" group in that directory

- **Two arguments**: `genifest run <group> <directory>`
  - Uses specified group in specified directory

**Flags:**

- `--tag string` - Add additional tag expression to the selected group
- `--output string` - Output mode: color, plain, markdown, or auto (default "auto")

**Groups-Based Selection:**

Genifest uses named groups defined in your configuration:

```yaml
groups:
  all: ["*"]                           # Default group (all changes)
  config-only: ["config"]              # Only configuration changes
  no-secrets: ["*", "!secret-*"]       # Everything except secrets
  dev: ["config", "image", "!production"] # Development environment
  prod: ["*", "!dev-*", "!test-*"]     # Production with exclusions
```

**Tag Expression Syntax:**

- `"*"` - All tags (wildcard)
- `"config"` - Literal tag match
- `"!secret-*"` - Negation with glob pattern
- `"prod-*"` - Glob pattern matching
- `"manifests:secret-*"` - Directory-scoped expression

**Enhanced Output:**

The run command provides detailed progress reporting:

- Configuration summary with group selection and total changes
- Individual change tracking: `file -> document[index] -> keySelector: old → new`
- Clear distinction between changes applied vs actual file modifications
- Final summary with file modification counts and emoji indicators

**Examples:**

```bash
# Apply all changes in current directory (uses "all" group)
genifest run

# Apply specific group in current directory
genifest run config-only

# Apply all changes in specific directory
genifest run examples/guestbook

# Apply specific group in specific directory
genifest run dev examples/guestbook

# Add additional tag expression to group selection
genifest run --tag "!secret" prod

# Apply changes without secrets in development environment
genifest run --tag "!secret-*" dev examples/app
```

### `genifest validate`

Validate configuration without applying changes.

```bash
genifest validate [directory] [flags]
```

**Flags:**

- `--strict` - Enable strict schema validation (fail on unknown fields)
- `--warn` - Enable schema validation warnings (warn on unknown fields)
- `--output string` - Output mode: color, plain, markdown, or auto (default "auto")

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

## Output Modes

Both `run` and `validate` commands support flexible output formatting with the `--output` flag:

### Available Output Modes

- **`color`** - ANSI color codes and emojis for terminal display
- **`plain`** - Clean text without colors, suitable for logging and automation
- **`markdown`** - Formatted markdown output for documentation workflows
- **`auto`** - Automatically detects TTY and uses color for terminals, plain for redirected output (default)

### Output Mode Examples

```bash
# Colored output with emojis (for terminals)
genifest validate --output=color
genifest run dev --output=color

# Plain text output (for scripts/logs)
genifest validate --output=plain
genifest run prod --output=plain

# Markdown formatted output (for documentation)
genifest validate --output=markdown
genifest run staging --output=markdown

# Auto-detect based on TTY (default behavior)
genifest validate --output=auto
genifest validate  # same as above
```

### Use Cases

- **`color`**: Interactive terminal sessions, development work
- **`plain`**: CI/CD pipelines, log files, automation scripts
- **`markdown`**: Documentation generation, integration with documentation workflows
- **`auto`**: General use, automatically adapts to context

### TTY Detection

When using `--output=auto` (the default), genifest automatically:
- Uses `color` mode when output goes to a terminal (TTY)
- Uses `plain` mode when output is redirected to files or pipes

```bash
genifest validate              # Color output in terminal
genifest validate > log.txt    # Plain output when redirected
genifest validate | less       # Plain output when piped
```

## Global Options

- `--help` - Show help for any command
- `--version` - Show version information

## Examples

See the [Quick Start Guide](../getting-started/quickstart.md) for practical examples.