# Functions

Reference for defining and using functions in Genifest.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Overview

Functions provide reusable value generation logic that can be called from changes or other functions.

## Function Definition

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

## Function Scoping

Functions are scoped to their definition location:
- Root functions are available everywhere
- Directory functions are available to that directory and children
- This prevents naming conflicts

## Built-in Functions

Currently, Genifest does not provide built-in functions. All functions must be defined in configuration files.

## Examples

### Environment-based Replica Count
```yaml
functions:
  - name: "get-replicas"
    params:
      - name: "environment"
        required: true
    valueFrom:
      template:
        string: '{{ if eq .environment "production" }}5{{ else }}2{{ end }}'
```

### Dynamic Image Tags
```yaml
functions:
  - name: "get-image-tag"
    params:
      - name: "service"
        required: true
      - name: "environment"
        required: true
    valueFrom:
      template:
        string: "${service}:${environment}-latest"
        variables:
          - name: "service"
            valueFrom:
              argRef:
                name: "service"
          - name: "environment"
            valueFrom:
              argRef:
                name: "environment"
```

## Calling Functions

```yaml
changes:
  - fileSelector: "*.yaml"
    keySelector: ".spec.replicas"
    valueFrom:
      call:
        function: "get-replicas"
        args:
          - name: "environment"
            valueFrom:
              default:
                value: "production"
```

## See Also

- [Value Generation](../user-guide/value-generation.md) - ValueFrom expressions
- [Configuration Schema](schema.md) - Complete schema