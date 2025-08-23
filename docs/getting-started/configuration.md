# Configuration

This page covers the basics of configuring Genifest for your project.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Overview

Genifest uses YAML configuration files to define how to generate and modify Kubernetes manifests. The configuration system is hierarchical and metadata-driven.

## Basic Configuration Structure

```yaml
metadata:
  cloudHome: "."
  paths:
    - path: "scripts"
      scripts: true
      depth: 0
    - path: "manifests"
      files: true
      depth: 1
    - path: "files"
      files: true
      depth: 0

functions:
  - name: "example-function"
    params:
      - name: "param1"
        required: true
    valueFrom:
      default:
        value: "example"

changes:
  - fileSelector: "*.yaml"
    keySelector: ".metadata.name"
    valueFrom:
      call:
        function: "example-function"
        args:
          - name: "param1"
            valueFrom:
              default:
                value: "value"
```

## Next Steps

- [Quick Start Guide](quickstart.md) - Hands-on tutorial
- [Core Concepts](../user-guide/concepts.md) - Detailed concepts
- [CLI Reference](../user-guide/cli-reference.md) - Command documentation