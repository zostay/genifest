## v1.0.0-rc2  2025-08-19

_Major CLI restructuring and enhanced user experience improvements._

* **CLI Architecture Overhaul**: Converted from flag-based to subcommand-based architecture
  * `genifest run [directory]` - Apply changes with enhanced progress reporting  
  * `genifest tags [directory]` - List all available tags in configuration
  * `genifest validate [directory]` - Validate configuration without applying changes
  * `genifest config [directory]` - Display merged configuration in YAML format
  * `genifest version` - Show version information
  * All commands support optional directory arguments for operation from any location
* **Enhanced Output and Reporting**: 
  * Detailed progress reporting with emoji indicators
  * Change tracking shows `file -> document[index] -> key: old → new` for all modifications
  * Clear distinction between changes applied vs actual modifications made
  * Comprehensive statistics and file modification summaries
* **Code Quality Improvements**:
  * Extracted ~100 lines of duplicate code into shared utilities (`internal/cmd/common.go`)
  * Improved error handling with rich context and user-friendly messages
  * Fixed file path handling bug in configuration loading for nested directories
  * Enhanced file selector pattern matching logic
* **User Experience Enhancements**:
  * Running `genifest` without subcommand now shows help instead of applying changes
  * Better validation with actionable error messages
  * Configuration display for debugging and understanding project structure

## v1.0.0-rc1  2025-08-18

_This is a complete rewrite of genifest, removing all the old cruft and making it more flexible._
* The primary configuration file is now named `genifest.yaml` and must be in the same directory that the `genifest` binary is run.
* The system supports three top-level types of configuration files: manifests, files, and scripts.
   * Manifests are for YAML files organized into application sub-directories, a typical arrangement for a Kubernetes cluster configuration.
   * Files are for general configuration files, also organized into application sub-directories. These may be embedded into other files using `valueFrom.file`.
   * Scripts are for executable scripts used to derive content for inclusion in manifests. Only scripts found in these directories will be permitted to run using `valueFrom.script`.
* Other `genifest.yaml` configurations found in these directories are loaded and merged into the top-level one before processing and applying changes.
* Changes are applied in-place, expecting the user to use version control to track changes as part of a gitops process.
* The system defines a simple tag-based scheme for choosing which changes to execute on each run.
* The following `valueFrom` types are defined:
  * `call` functions as a simple macro for calling other `valueFrom` expressions.
  * `pipeline` defines a way of chaining operations together so that the output from a previous step can feed into a following step
  * `file` embeds a file from a file directory into another YAML file
  * `template` allows for the creation of simple templates using `${var}` style interpolation
  * `script` executes custom scripts found in a scripts directory
  * `argRef` is used to read variables and arguments inside a `valueFrom` or function definition
  * `default` is used for literal values
  * `documentRef` is used to lookup values elsewhere in the current YAML document and use the looked up value

## v0.2.0  2025-07-17

 * The ddbLookup function in templates now supports `BOOL` fields and also nested fields with `M` (e.g., `Counter.M.prod.N`).
 * Updated github.com/bmatcuk/doublestar/v4 to v4.9.0
 * Updated k8s.io/apimachinery to v0.33.2
 * Updated k8s.io/client-go to v0.33.2
 * Updated github.com/spf13/cobra to v1.9.1
 * Updated github.com/bitnami/labs to v0.30.0

## v0.1.4  2024-10-15

 * Upgraded to go-std v0.9.1 to fix a bug in string indents.

## v0.1.3  2024-09-05

 * Adding Arm64 builds for Linux.

## v0.1.2  2024-08-09

 * Upgraded to Ghost v0.6.2

## v0.1.1  2024-08-09

 * Fix a bug in `file` that caused it to files when `files_dir` was set.

## v0.1.0  2024-08-09

 * Added the `files_dir` setting to cluster configuration to set the location of the directory used to load files when calling the `files` function when templating.
 * Fixed a minor problem with secrets skipping. It didn't always work correctly when the `applyTemplate` function to template a second file.
 * Added the `ghost` section to cluster configuration for configuring `ghost` for secrets management.
 * Added `ghost.config` to cluster configuration to select the configuration file to use for working with secrets.
 * Added `ghost.keeper` to cluster configuration to select the secrets keeper to use for working with secrets.

## v0.0.2  2024-06-27

 * Fixing the install script.

## v0.0.1  2024-06-27

 * Adding binaries to the release.

## v0.0.0  2024-06-25

 * Initial release.
