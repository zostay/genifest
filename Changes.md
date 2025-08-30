## v1.0.0-rc5  TBD

_Complete revamp of tag system with groups-based selection and improved CLI argument handling._

* **Groups-Based Tag System**: Revolutionary new approach to tag selection and organization

    * **Groups configuration** with customizable tag expression patterns for organizing changes
    * **Default "all" group** containing `["*"]` for backward compatibility with existing workflows  
    * **Tag expressions** supporting wildcards (`*`), negations (`!tag-name`), and directory scoping (`dir:tag`)
    * **Expression evaluation** with sequential processing where later expressions override earlier ones
    * **Flexible matching** allowing complex combinations like `["*", "!secret-*", "secret-foo"]`

* **Enhanced CLI Argument Structure**: Modernized command-line interface with intuitive argument handling

    * **Zero arguments**: Uses "all" group in current directory (`genifest run`)
    * **One argument**: Group name in current directory OR directory path with "all" group
    * **Two arguments**: Specified group name in specified directory (`genifest run dev examples/guestbook`)
    * **Intelligent parsing** that distinguishes between group names and directory paths
    * **--tag option** for adding additional tag expressions to any group selection

* **Directory-Scoped Tag Expressions**: Advanced scoping system for complex project structures

    * **Scoped expressions** using `<directory>:<tag-expression>` syntax for targeted rule application
    * **Automatic directory prefixing** when merging subordinate configurations  
    * **Nested rule application** where child directory rules are applied after parent rules
    * **Path-aware matching** ensuring expressions only apply to appropriate directory contexts

* **Configuration Merging Improvements**: Enhanced configuration loading with groups support

    * **Groups merging** with proper directory scoping and inheritance
    * **Default groups** automatically provided when not specified in configuration
    * **Hierarchical merging** preserving parent-child relationships in nested configurations
    * **Context preservation** maintaining directory information throughout the merge process

* **Breaking Changes** (no backward compatibility for RC software):

    * **Removed --include-tags/--exclude-tags** in favor of groups-based selection
    * **New argument structure** requires explicit group names for non-default selections
    * **Groups section** now required for custom tag selection (defaults provided automatically)

* **Real-World Examples**:

    ```yaml
    groups:
      all: ["*"]                           # All tags (default behavior)
      config-only: ["config"]              # Only configuration changes
      no-secrets: ["*", "!secret-*"]       # Everything except secrets
      dev: ["config", "image", "!production"] # Development environment
      prod: ["*", "!dev-*", "!test-*"]     # Production with exclusions
    ```

* **Advanced Usage Patterns**:

    ```bash
    genifest run                          # All changes (default group)
    genifest run config-only             # Only config changes
    genifest run dev examples/app        # Dev group in specific directory  
    genifest run --tag "!secret" prod    # Add negation to prod group
    ```

## v1.0.0-rc4  2025-08-29

_DocumentSelector for multi-document YAML targeting and smart bracket parsing enhancements._

* **DocumentSelector Feature**: Multi-document YAML file targeting capability

    * **Document targeting** using simple key-value matching with dot notation (`metadata.name`, `kind`)
    * **Multi-document support** for YAML files with resources separated by `---`
    * **Precise selection** allowing different changes to different documents in the same file
    * **Optional usage** - when omitted, changes apply to all documents in the file
    * **Real-world usage** for ConfigMaps, Secrets, and Deployments in consolidated files

* **Smart Bracket Parsing**: Enhanced keySelector expression parsing for complex keys

    * **Quote-aware parsing** correctly distinguishes numeric indices from quoted string keys
    * **String key support** for keys starting with numbers like `["1password.json"]`
    * **Prevents numeric parsing** of quoted strings that happen to start with digits
    * **Backward compatibility** maintaining existing numeric index functionality `[0]`, `[-1]`
    * **Special character support** for keys with dots, dashes, and complex names

* **README Documentation Enhancements**:

    * **Document Selection section** with comprehensive multi-document YAML examples
    * **Smart bracket parsing** examples showing `["1password.json"]` vs numeric `[1]`
    * **Alternative operator documentation** with `//` fallback value examples
    * **Enhanced key features** highlighting quote handling and parsing improvements
    * **DocumentSelector features** explaining targeting capabilities and use cases

## v1.0.0-rc3  2025-08-22

_Enhanced keySelector syntax with advanced pipeline operations and comprehensive documentation updates._

* **Advanced KeySelector Syntax**: Complete implementation of complex yq-style expressions

    * **Array iteration** with `[]` syntax for processing all elements in arrays
    * **Pipeline operations** using `|` operator for chaining multiple operations
    * **Filtering functions** with `select()` for conditional element selection
    * **Comparison operators** supporting `==` and `!=` for equality tests
    * **Complex expressions** like `.spec.containers[] | select(.name == "frontend") | .image`
    * **Grammar-based parsing** using `participle/v2` for robust expression handling
    * **Write operation support** for complex expressions enabling modification of selected elements

* **Implementation Architecture**:

    * **New parser package** `internal/keysel` with complete AST-based expression parsing
    * **Evaluation engine** supporting pipeline processing with array iteration and filtering
    * **Backward compatibility** maintaining support for simple path expressions
    * **Expression validation** at parse-time with clear error messages
    * **Dual write paths** handling both simple and complex expressions efficiently

* **Real-world Applications**:

    * **Container targeting** by name instead of index (`.containers[] | select(.name == "app")`)
    * **Multi-container deployments** with selective updates to specific containers
    * **Sidecar management** updating proxy, logging, and monitoring containers independently
    * **Environment-specific configuration** with conditional container selection
    * **Modern Kubernetes patterns** supporting named container architectures

* **Documentation Enhancements**:

    * **Comprehensive keySelector reference** with complete syntax guide and examples
    * **Advanced patterns documentation** with real-world pipeline examples
    * **Updated schema documentation** reflecting new syntax capabilities  
    * **Enhanced README** with modern keySelector examples and feature comparisons
    * **Changelog synchronization** between main changelog and docs site

* **Code Quality Improvements**:

    * **Parser grammar fixes** resolving infinite loop issues with empty path components
    * **Expression evaluation** supporting complex pipelines with proper error handling
    * **Test coverage** comprehensive tests for all new syntax features
    * **Integration validation** verified with guestbook examples

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
