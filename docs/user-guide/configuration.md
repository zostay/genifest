# Configuration Files

Detailed reference for Genifest configuration file format and options.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Configuration File Format

Genifest uses YAML configuration files named `genifest.yaml`.

## Configuration Schema

### Metadata Section

```yaml
metadata:
  cloudHome: "."              # Security boundary for file operations
  paths:                      # Unified directory configuration
    - path: "scripts"         # Directory path
      scripts: true           # Enable script execution access
      depth: 0                # Directory depth (0-based, 0=single level)
    - path: "manifests" 
      files: true             # Enable file inclusion access
      depth: 1                # Go one level deep into subdirectories
    - path: "files"
      files: true
      depth: 0
```

### Functions Section

```yaml
functions:
  - name: "function-name"
    params:
      - name: "param1"
        required: true
      - name: "param2"
        required: false
    valueFrom:
      # ValueFrom expression
```

### Changes Section

```yaml
changes:
  - tag: "optional-tag"
    fileSelector: "*.yaml"
    keySelector: ".path.to.field"
    valueFrom:
      # ValueFrom expression
```

## See Also

- [Core Concepts](concepts.md) - Understanding the configuration system
- [Value Generation](value-generation.md) - ValueFrom expressions
- [Quick Start](../getting-started/quickstart.md) - Practical examples