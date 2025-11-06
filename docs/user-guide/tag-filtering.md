# Tag Filtering

Advanced techniques for organizing and controlling which changes are applied using the groups-based tag selection system.

## Overview

Genifest uses a **groups-based tag system** that revolutionizes how you organize and select changes. Instead of using simple include/exclude flags, you define named groups with sophisticated tag expression patterns.

## Groups-Based Tag Selection

### Basic Groups Configuration

Groups are defined in your `genifest.yaml` configuration:

```yaml
groups:
  all: ["*"]                           # All changes (default)
  config-only: ["config"]              # Only configuration changes
  no-secrets: ["*", "!secret-*"]       # Everything except secrets
  dev: ["config", "image", "!production"] # Development environment
  prod: ["*", "!dev-*", "!test-*"]     # Production with exclusions
```

### Tag Expression Syntax

Tag expressions support powerful pattern matching:

- **Wildcards**: `"*"` matches all tags
- **Literal tags**: `"config"` matches exactly "config"
- **Negations**: `"!secret-*"` excludes tags matching "secret-*"
- **Glob patterns**: `"prod-*"` matches tags starting with "prod-"
- **Directory scoping**: `"manifests:prod-*"` matches "prod-*" only in manifests directory

### Expression Evaluation

Tag expressions are evaluated sequentially, with later expressions overriding earlier ones:

```yaml
groups:
  flexible: ["*", "!secret-*", "secret-dev"]  # All except secrets, but include secret-dev
  staged: ["dev-*", "test-*", "!test-broken"] # Dev and test, but exclude test-broken
```

## Using Groups

### Command-Line Usage

The CLI uses intelligent argument parsing:

```bash
# Zero arguments: Uses "all" group in current directory
genifest run

# One argument: Group name OR directory path
genifest run config-only           # Group "config-only" in current directory
genifest run examples/guestbook    # Group "all" in specified directory

# Two arguments: Group name in specified directory
genifest run dev examples/guestbook    # Group "dev" in examples/guestbook directory

# Additional tag expressions
genifest run --tag "!secret" prod     # Add "!secret" to "prod" group selection
```

### Automatic Defaults

If no groups are defined in your configuration, Genifest automatically provides:

```yaml
groups:
  all: ["*"]  # Default group for backward compatibility
```

## Directory-Scoped Expressions

Advanced scoping allows different rules for different directories:

```yaml
groups:
  production:
    - "*"                    # All changes
    - "!dev-*"              # Except development changes
    - "!test-*"             # Except test changes
    - "manifests:secret-*"  # But include secrets only in manifests directory
```

When configurations are merged from subdirectories, directory prefixes are automatically added to maintain proper scoping.

## Real-World Examples

### Environment-Based Groups

```yaml
groups:
  development:
    - "config"
    - "dev-*"
    - "!production"

  staging:
    - "config"
    - "staging-*"
    - "!dev-*"
    - "!production"

  production:
    - "*"
    - "!dev-*"
    - "!staging-*"
    - "!test-*"
```

### Feature-Based Groups

```yaml
groups:
  database-only: ["db-*", "migration-*"]
  frontend-only: ["web-*", "ui-*", "!backend-*"]
  security: ["auth-*", "cert-*", "secret-*"]
  monitoring: ["metrics-*", "logging-*", "alerts-*"]
```

### Complex Multi-Environment

```yaml
groups:
  qa-testing:
    - "test-*"
    - "staging-*"
    - "mock-*"
    - "!production"
    - "!real-secrets"

  prod-deployment:
    - "*"
    - "!test-*"
    - "!mock-*"
    - "!dev-*"
    - "manifests:real-secrets"  # Real secrets only in manifests
```

## Best Practices

### Group Naming

- Use descriptive names that indicate purpose: `prod-deploy`, `dev-testing`
- Include environment context: `staging-with-mocks`, `prod-no-secrets`
- Use hyphens for readability: `database-migration` rather than `databasemigration`

### Expression Design

- Start broad, then exclude: `["*", "!unwanted-*"]`
- Use positive inclusion for critical changes: `["essential", "security-*"]`
- Leverage directory scoping for complex projects: `["*", "scripts:dev-*"]`

### Configuration Organization

```yaml
# Clear, purpose-driven groups
groups:
  # Standard environments
  dev: ["*", "!production", "!real-secrets"]
  prod: ["*", "!dev-*", "!test-*", "!mock-*"]

  # Specific purposes
  config-only: ["config", "settings"]
  secrets-update: ["secret-*", "cert-*"]
  database-migrate: ["db-*", "migration-*"]
```

## Troubleshooting

### Group Not Found

If you specify a group that doesn't exist, Genifest will show available groups:

```bash
$ genifest run invalid-group
Error: Group "invalid-group" not found. Available groups: all, dev, prod, config-only
```

### No Changes Selected

If your group expressions don't match any changes:

```bash
$ genifest run empty-group
No changes selected by group "empty-group" - check your tag expressions
```

### Debugging Group Selection

Use the `tags` command to see what tags are available:

```bash
genifest tags                    # Show all tags
genifest tags examples/app       # Show tags in specific directory
```

## Output Modes

When working with groups, you can control the output format for different use cases:

```bash
# Interactive development with colors and emojis
genifest run dev --output=color

# CI/CD pipelines with clean text output
genifest run prod --output=plain

# Documentation generation with markdown formatting
genifest run config-only --output=markdown

# Auto-detect based on terminal context (default)
genifest run staging --output=auto
```

The output mode is particularly useful when:

- **Development**: Use `color` mode for rich visual feedback
- **Automation**: Use `plain` mode for clean logs and scripts
- **Documentation**: Use `markdown` mode for generating documentation
- **General use**: Use `auto` mode for adaptive behavior

Use the `config` command to verify your groups configuration:

```bash
genifest config | grep -A 10 groups:
```

## See Also

- [CLI Reference](cli-reference.md) - Complete command syntax
- [Core Concepts](concepts.md) - Understanding the tag system
- [Configuration](configuration.md) - Groups configuration reference