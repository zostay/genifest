# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Genifest is a Kubernetes manifest generation tool that creates deployment manifests from templates for GitOps workflows. It's a Go CLI application built with Cobra that processes configuration files to generate Kubernetes resources.

IMPORTANT NOTE: The project is currently in the process of rewrite.

## Architecture

The codebase follows a typical Go CLI structure:

- `main.go` - Entry point that calls `cmd.Execute()`
- `internal/cmd/` - Cobra command definitions (root.go, generate.go)
- `internal/config/` - Configuration parsing and management

### Configuration System

The configuration system uses a **metadata-driven loading approach**:

#### Core Types
- `Config` - Main configuration structure containing metadata, files, changes, and functions
- `MetaConfig` - Metadata including cloudHome, scripts, manifests, and files paths
- `PathContext` - Represents paths with context about where they were defined
- `ChangeOrder` - Defines modifications to be applied to managed files
- `FunctionDefinition` - Defines reusable functions for value generation
- `ValueFrom` - Union type for different value sources (functions, templates, scripts, etc.)

#### Loading Behavior
1. **Starts with root `genifest.yaml`** - Loads the top-level configuration first
2. **Metadata-driven discovery** - Uses `scripts`, `manifests`, and `files` paths to discover additional directories
3. **Depth-limited recursion**:
   - Scripts directories: single level only
   - Manifests/Files directories: two levels deep
4. **Synthetic configs** - Creates configurations for directories without `genifest.yaml` containing all `.yaml`/`.yml` files
5. **CloudHome scoping** - Respects cloudHome boundaries and local overrides

#### Validation System
- **Context-aware validation** - Function references validated against available functions
- **Path security** - All paths validated to stay within cloudHome boundaries
- **Function scoping** - Functions only accessible from same path or child paths
- **Compile-time checks** - Function calls validated before execution

#### PathContext System
- **Tracks origin** - Each path remembers where it was defined
- **Custom YAML marshalling** - Appears as simple strings in YAML files
- **Runtime context** - Available for path resolution during execution

## Commands

### Build and Development
```bash
# Build the application
go build -o genifest

# Install from source
go install github.com/zostay/genifest/cmd/genifest@latest

# Run tests
go test ./...

# Lint (using golangci-lint)
golangci-lint run --timeout=5m
```

## Configuration File Structure

### Basic genifest.yaml
```yaml
metadata:
  cloudHome: "."           # Optional: override cloudHome for this scope
  scripts: ["scripts"]     # Directories containing scripts (single level)
  manifests: ["k8s"]       # Directories containing manifests (two levels)
  files: ["files"]         # Directories containing template files (two levels)

files:
  - "deployment.yaml"      # Files managed by genifest
  - "service.yaml"

changes:
  - tag: "production"      # Optional tag for conditional application
    fileSelector: "*.yaml"
    keySelector: ".spec.replicas"
    valueFrom:
      default:
        value: "3"

functions:
  - name: "get-replicas"   # Reusable function definition
    params:
      - name: "environment"
        required: true
    valueFrom:
      template:
        string: "replicas-${environment}"
        variables:
          - name: "environment"
            valueFrom:
              argRef:
                name: "environment"
```

### Loading Rules
- **Root discovery**: Starts by loading root `genifest.yaml` or creates synthetic config
- **Metadata processing**: Follows `scripts`, `manifests`, `files` paths to discover additional directories
- **Automatic inclusion**: Directories without `genifest.yaml` get synthetic configs with all YAML files
- **Scope respect**: Each config's metadata only affects its subdirectories
- **Security**: All paths must stay within cloudHome boundaries