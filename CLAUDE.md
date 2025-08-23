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
- `MetaConfig` - Metadata including cloudHome and unified paths configuration
- `PathConfig` - Unified path configuration with capabilities (scripts/files) and configurable depth
- `PathContext` - Represents paths with context about where they were defined
- `ChangeOrder` - Defines modifications to be applied to managed files
- `FunctionDefinition` - Defines reusable functions for value generation
- `ValueFrom` - Union type for different value sources (functions, templates, scripts, etc.)

#### Loading Behavior
1. **Starts with root `genifest.yaml`** - Loads the top-level configuration first
2. **Metadata-driven discovery** - Uses unified `paths` configuration to discover additional directories
3. **Configurable depth recursion**:
   - Each path can specify its own depth (0-based indexing, default: 0)
   - Depth 0: single level only, Depth 1: one level deep, etc.
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

**Using Make (Recommended):**
```bash
# Show all available targets
make help

# Development workflow
make build                               # Build the application
make test                                # Run all tests
make lint                                # Run linters
make check                               # Run all checks (fmt, vet, lint, test)

# Example testing
make run-example                         # Run guestbook example
make validate-example                    # Validate guestbook example
make config-example                      # Show merged config
make tags-example                        # Show available tags

# Release
make release                             # Build release binaries for all platforms
make install                             # Install to GOPATH/bin
```

**Manual Commands:**
```bash
# Build the application
go build -o genifest

# Install from source
go install github.com/zostay/genifest/cmd/genifest@latest

# Run tests
go test ./...

# Lint (using golangci-lint)
golangci-lint run --timeout=5m

# Run the CLI tool from project root
./genifest run                           # Apply all changes
./genifest run --include-tags production # Apply only production changes
./genifest run --exclude-tags staging    # Apply all except staging changes
./genifest tags                          # List available tags
./genifest validate                      # Validate configuration
./genifest config                        # Display merged configuration
./genifest version                       # Show version information
```

### CLI Usage

The genifest CLI uses a subcommand-based architecture with optional directory arguments:

```bash
# Core commands (can be run from any directory)
genifest run [directory]                    # Apply changes
genifest tags [directory]                   # List available tags
genifest validate [directory]               # Validate configuration
genifest config [directory]                 # Display merged configuration
genifest version                            # Show version information

# Tag filtering options for 'run' command:
-i, --include-tags strings   include only changes with these tags (supports glob patterns)
-x, --exclude-tags strings   exclude changes with these tags (supports glob patterns)

# Examples:
genifest run                                # Apply all changes in current directory
genifest run path/to/project               # Apply changes in specified directory
genifest run --include-tags production     # Apply only production-tagged changes
```

**Tag filtering logic:**
- No flags: All changes applied (tagged and untagged)
- Include only: Only changes matching include patterns
- Exclude only: All changes except those matching exclude patterns  
- Both flags: Changes matching include but not exclude patterns
- Glob patterns supported: `prod*`, `test-*`, etc.

**Enhanced Output:**
- Detailed progress reporting with emoji indicators
- Change tracking: `file -> document[index] -> key: old → new`
- Distinguishes between changes applied vs actual modifications
- Clear summary of files modified and change counts

## Configuration File Structure

### Basic genifest.yaml
```yaml
metadata:
  cloudHome: "."           # Optional: override cloudHome for this scope
  paths:
    - path: "scripts"      # Directory path
      scripts: true        # Enable script execution access
      depth: 0             # Single level only (0-based)
    - path: "k8s"          # Directory path  
      files: true          # Enable file inclusion access
      depth: 1             # One level deep (0-based)
    - path: "files"        # Directory path
      files: true          # Enable file inclusion access
      depth: 0             # Single level only (0-based)

files:
  include:
    - "deployment.yaml"    # Files managed by genifest
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
- **Metadata processing**: Follows unified `paths` configuration to discover additional directories
- **Automatic inclusion**: Directories without `genifest.yaml` get synthetic configs with all YAML files
- **Scope respect**: Each config's metadata only affects its subdirectories
- **Security**: All paths must stay within cloudHome boundaries

## Implementation Architecture

### ValueFrom Evaluation System (`internal/changes/`)

The system implements a sophisticated value evaluation architecture:

- **EvalContext**: Immutable context carrying current file, document, variables, and functions
- **Evaluators**: Separate functions for each ValueFrom type (DefaultValue, ArgumentRef, BasicTemplate, FunctionCall, ScriptExec, FileInclusion, DocumentRef, CallPipeline)
- **Document Processing**: Uses `gopkg.in/yaml.v3` for parsing and modifying YAML documents
- **Path Resolution**: Custom key selector implementation handling `.spec.replicas`, array access `[0]`, and nested paths
- **Context Isolation**: Each evaluation creates new contexts to prevent side effects

### CLI Implementation (`internal/cmd/`)

The command-line interface implements a subcommand-based architecture:

#### Subcommand Structure
- **root.go**: Main command dispatcher, removed Run function to force subcommand usage
- **run.go**: Core functionality for applying changes with enhanced progress reporting
- **tags.go**: Lists all tags found in configuration files
- **validate.go**: Validates configuration without applying changes
- **config.go**: Displays merged configuration in YAML format
- **version.go**: Shows version information
- **common.go**: Shared utilities for directory resolution and configuration loading

#### Enhanced Features
- **Directory Arguments**: All commands accept optional directory arguments for operation
- **Progress Reporting**: Detailed output with emoji indicators and change tracking
- **Change Tracking**: Shows `file -> document[index] -> key: old → new` for all modifications
- **Statistics**: Distinguishes between changes applied vs actual modifications made
- **Error Context**: Rich error messages with file and path context

#### Core Capabilities
- **Tag Filtering**: Complex logic supporting include/exclude patterns with glob matching
- **File Processing**: Multi-document YAML handling with atomic write operations  
- **Change Application**: Applies ValueFrom expressions to modify specific YAML paths
- **Configuration Validation**: Comprehensive validation with user-friendly error messages

### Key Design Decisions

1. **Simplified yq Integration**: Instead of full `github.com/mikefarah/yq/v4` integration, implemented basic key selector parsing for common patterns. This avoided API complexity while supporting essential use cases.

2. **Immutable Contexts**: EvalContext operations return new instances rather than modifying existing ones, enabling safe concurrent use and preventing unexpected state changes.

3. **Document Reference Strategy**: Chose to implement document references by re-evaluating ValueFrom expressions in document context rather than pre-computing values, allowing dynamic document-aware value generation.

4. **Tag Processing**: Implemented tag filtering at the change application level rather than configuration loading level, providing more flexible runtime control.

## Development Lessons Learned

### Testing Strategy
- **Integration Tests**: Essential for validating end-to-end workflows with real configuration files
- **Isolation**: Use temp directories and controlled fixtures for reliable test execution
- **Context Testing**: Verify immutability contracts and variable scoping behavior

### Error Handling Patterns
- **Contextual Errors**: Always wrap errors with context about what operation failed and on which file/path
- **User-Friendly Messages**: Distinguish between user configuration errors and system failures
- **Graceful Degradation**: Continue processing other files when individual files fail

### YAML Processing Challenges
- **Multi-Document Files**: Handle YAML files with multiple documents separated by `---`
- **Path Navigation**: Complex selectors like `.spec.template.spec.containers[0].image` require careful parsing
- **Type Preservation**: Maintain YAML node types when modifying values to preserve structure

### CLI Design Principles
- **Project Root Detection**: Always validate presence of `genifest.yaml` before processing
- **Clear Output**: Provide user feedback about which files were modified and what changes were applied
- **Tag Logic**: Implement intuitive include/exclude behavior that matches user expectations

### Performance Considerations
- **Lazy Loading**: Only parse YAML documents when changes need to be applied
- **Memory Efficiency**: Process files individually rather than loading entire project into memory
- **File I/O**: Batch file operations and only write when changes are actually made

### Code Quality Standards
- **Linting**: Address critical linting issues (errcheck, staticcheck) while allowing style preferences
- **Documentation**: Ensure all public APIs and complex functions have proper documentation
- **Test Coverage**: Focus on integration tests and critical path validation over 100% unit test coverage
- Always run genifeest by passing the diretory to the command: genifest run examples/guestbook, genifest valiadate examples/guestbook, genifest tag examples/genifest, etc. Do not change directories to run genifest.