# Tag Filtering

Advanced techniques for using tags to control which changes are applied.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Overview

Tags provide a powerful way to selectively apply changes based on environment, change type, or any other criteria you define.

## Basic Tag Usage

```yaml
changes:
  - tag: "production"
    fileSelector: "*.yaml"
    keySelector: ".spec.replicas"
    valueFrom:
      default:
        value: "5"
        
  - tag: "development"
    fileSelector: "*.yaml"
    keySelector: ".spec.replicas"
    valueFrom:
      default:
        value: "1"
```

## Tag Filtering Commands

```bash
# Apply only production changes
genifest run --include-tags production

# Apply all except development changes
genifest run --exclude-tags development

# Complex filtering with globs
genifest run --include-tags "prod*" --exclude-tags "test-*"
```

## Tag Logic

- **No flags**: All changes applied (tagged and untagged)
- **Include only**: Only changes matching include patterns
- **Exclude only**: All changes except those matching exclude patterns
- **Both flags**: Changes matching include but not exclude patterns

## Glob Patterns

Tags support glob pattern matching:
- `*` - Matches any characters
- `prod*` - Matches tags starting with "prod"
- `*-test` - Matches tags ending with "-test"

## See Also

- [CLI Reference](cli-reference.md) - Command syntax
- [Core Concepts](concepts.md) - Understanding tags