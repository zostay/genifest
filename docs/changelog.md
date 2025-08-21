# Changelog

All notable changes to Genifest will be documented in this page.

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

*Enhanced keySelector syntax and improved configuration scoping.*

### Added
- **Advanced KeySelector Syntax**: Complete rewrite of path expression parsing
  - Grammar-based parser using `participle/v2` for robust expression handling  
  - Support for yq-style path expressions: `.field`, `[index]`, `["key"]`, slicing
  - Negative array indexing: `.items[-1]` for last element
  - Array slicing: `.items[1:3]`, `.items[2:]`, `.items[:3]`
  - Quoted key access: `.data.["app.yaml"]`, `.labels.["app.kubernetes.io/name"]`
  - Complex nested expressions: `.spec.template.spec.containers[0].image`
  - Parse-time validation with clear error messages

- **Path-Based Change Scoping**: Enhanced security and isolation
  - Changes defined in a directory only apply to files within that directory tree
  - Prevents accidental cross-contamination between application configurations  
  - Maintains proper boundaries for multi-application repositories
  - Exported `ChangeOrder.Path` field for cross-package access

### Improved  
- **Enhanced Error Messages**: Better context and debugging information
  - KeySelector parsing errors show exact problem location
  - Runtime errors include file and path context
  - Clear distinction between syntax and runtime errors

- **Documentation Enhancements**:
  - Comprehensive [KeySelector Reference](reference/keyselectors.md) with grammar details
  - Updated README with clear yq/jq syntax comparison
  - Enhanced CLI reference with latest features and examples
  - Improved concepts documentation with path scoping examples

### Fixed
- **Field+Bracket Parsing**: Fixed parsing of expressions like `.data.["app.yaml"]`  
- **Path Scoping**: Resolved issue where changes applied globally instead of being directory-scoped
- **Grammar Edge Cases**: Improved handling of complex nested expressions
- **Import Organization**: Cleaned up unused imports and dependencies

### Technical
- **New Package**: `internal/keysel` with complete parser implementation
- **AST Design**: Proper Abstract Syntax Tree for keySelector expressions
- **Test Coverage**: Comprehensive tests for all keySelector features
- **Type Safety**: Compile-time validation of keySelector expressions

## [v1.0.0-rc2] - 2025-08-19

*Major CLI restructuring and enhanced user experience improvements.*

### Added
- **CLI Architecture Overhaul**: Converted from flag-based to subcommand-based architecture
  - `genifest run [directory]` - Apply changes with enhanced progress reporting  
  - `genifest tags [directory]` - List all available tags in configuration
  - `genifest validate [directory]` - Validate configuration without applying changes
  - `genifest config [directory]` - Display merged configuration in YAML format
  - `genifest version` - Show version information
  - All commands support optional directory arguments for operation from any location

### Improved
- **Enhanced Output and Reporting**: 
  - Detailed progress reporting with emoji indicators
  - Change tracking shows `file -> document[index] -> key: old → new` for all modifications
  - Clear distinction between changes applied vs actual modifications made
  - Comprehensive statistics and file modification summaries

### Fixed
- **Code Quality Improvements**:
  - Extracted ~100 lines of duplicate code into shared utilities (`internal/cmd/common.go`)
  - Improved error handling with rich context and user-friendly messages
  - Fixed file path handling bug in configuration loading for nested directories
  - Enhanced file selector pattern matching logic

### Changed
- **User Experience Enhancements**:
  - Running `genifest` without subcommand now shows help instead of applying changes
  - Better validation with actionable error messages
  - Configuration display for debugging and understanding project structure

## [v1.0.0-rc1] - 2025-08-18

*This is a complete rewrite of genifest, removing all the old cruft and making it more flexible.*

### Added
- The primary configuration file is now named `genifest.yaml` and must be in the same directory that the `genifest` binary is run.
- The system supports three top-level types of configuration files: manifests, files, and scripts.
  - Manifests are for YAML files organized into application sub-directories, a typical arrangement for a Kubernetes cluster configuration.
  - Files are for general configuration files, also organized into application sub-directories. These may be embedded into other files using `valueFrom.file`.
  - Scripts are for executable scripts used to derive content for inclusion in manifests. Only scripts found in these directories will be permitted to run using `valueFrom.script`.
- Other `genifest.yaml` configurations found in these directories are loaded and merged into the top-level one before processing and applying changes.
- Changes are applied in-place, expecting the user to use version control to track changes as part of a gitops process.
- The system defines a simple tag-based scheme for choosing which changes to execute on each run.
- The following `valueFrom` types are defined:
  - `call` functions as a simple macro for calling other `valueFrom` expressions.
  - `pipeline` defines a way of chaining operations together so that the output from a previous step can feed into a following step
  - `file` embeds a file from a file directory into another YAML file
  - `template` allows for the creation of simple templates using `${var}` style interpolation
  - `script` executes custom scripts found in a scripts directory
  - `argRef` is used to read variables and arguments inside a `valueFrom` or function definition
  - `default` is used for literal values
  - `documentRef` is used to lookup values elsewhere in the current YAML document and use the looked up value

## [v0.2.0] - 2025-07-17

### Added
- The ddbLookup function in templates now supports `BOOL` fields and also nested fields with `M` (e.g., `Counter.M.prod.N`).

### Updated
- Updated github.com/bmatcuk/doublestar/v4 to v4.9.0
- Updated k8s.io/apimachinery to v0.33.2
- Updated k8s.io/client-go to v0.33.2
- Updated github.com/spf13/cobra to v1.9.1
- Updated github.com/bitnami/labs to v0.30.0

## [v0.1.4] - 2024-10-15

### Fixed
- Upgraded to go-std v0.9.1 to fix a bug in string indents.

## [v0.1.3] - 2024-09-05

### Added
- Adding Arm64 builds for Linux.

## [v0.1.2] - 2024-08-09

### Updated
- Upgraded to Ghost v0.6.2

## [v0.1.1] - 2024-08-09

### Fixed
- Fix a bug in `file` that caused it to files when `files_dir` was set.

## [v0.1.0] - 2024-08-09

### Added
- Added the `files_dir` setting to cluster configuration to set the location of the directory used to load files when calling the `files` function when templating.
- Fixed a minor problem with secrets skipping. It didn't always work correctly when the `applyTemplate` function to template a second file.
- Added the `ghost` section to cluster configuration for configuring `ghost` for secrets management.
- Added `ghost.config` to cluster configuration to select the configuration file to use for working with secrets.
- Added `ghost.keeper` to cluster configuration to select the secrets keeper to use for working with secrets.

## [v0.0.2] - 2024-06-27

### Fixed
- Fixing the install script.

## [v0.0.1] - 2024-06-27

### Added
- Adding binaries to the release.

## [v0.0.0] - 2024-06-25

### Added
- Initial release.

---

*For more details about any release, see the [GitHub releases page](https://github.com/zostay/genifest/releases).*