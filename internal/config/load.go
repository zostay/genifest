package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadFromDirectory loads configurations using a metadata-driven approach.
// It starts by reading the top-level genifest.yaml and then loads directories
// indicated by the metadata settings (Scripts, Manifests, Files).
func LoadFromDirectory(rootDir string) (*Config, error) {
	return LoadFromDirectoryWithValidation(rootDir, ValidationModePermissive)
}

// LoadFromDirectoryWithValidation loads configurations with specified schema validation mode.
func LoadFromDirectoryWithValidation(rootDir string, mode ValidationMode) (*Config, error) {
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", rootDir, err)
	}

	// Initialize loading context
	lc := &loadingContext{
		configs:        make([]configWithPath, 0),
		rootDir:        abs,
		processed:      make(map[string]bool),
		primaryHome:    abs,
		validationMode: mode,
	}

	// Load the root directory first
	if err := lc.loadDirectoryRecursive(abs, 0, 0, abs); err != nil {
		return nil, err
	}

	if len(lc.configs) == 0 {
		return &Config{
			Metadata: MetaConfig{CloudHome: abs},
		}, nil
	}

	// Set primary home from the first config if it has one
	if len(lc.configs) > 0 && lc.configs[0].config.Metadata.CloudHome != "" {
		newHome := filepath.Join(abs, lc.configs[0].config.Metadata.CloudHome)
		if absHome, err := filepath.Abs(newHome); err == nil {
			lc.primaryHome = absHome
		}
	}

	result := mergeConfigs(lc.configs, lc.primaryHome)

	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return result, nil
}

// configWithPath holds a configuration along with metadata about where it was loaded from.
type configWithPath struct {
	config    *Config
	path      string
	depth     int
	cloudHome string // The effective cloudHome for this config
}

// loadingContext tracks the state during recursive config loading.
// It maintains a list of loaded configs and prevents infinite recursion.
type loadingContext struct {
	configs        []configWithPath
	rootDir        string
	processed      map[string]bool // Track processed directories to avoid cycles
	primaryHome    string          // The primary cloudHome from root
	validationMode ValidationMode  // Schema validation mode to use
}

// loadConfigFileWithValidation loads a configuration file with specified validation mode.
func loadConfigFileWithValidation(path string, mode ValidationMode) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Perform schema validation if not in permissive mode
	if mode != ValidationModePermissive {
		if err := ValidateWithSchema(data, mode); err != nil {
			// Check if it's a warning error
			var warningErr *SchemaWarningsError
			if errors.As(err, &warningErr) && mode == ValidationModeWarn {
				// Display warnings immediately but continue loading
				fmt.Fprintf(os.Stderr, "⚠️  Schema validation warnings in %s:\n", path)
				for _, warning := range warningErr.Warnings {
					fmt.Fprintf(os.Stderr, "  • %s\n", warning.String())
				}
				fmt.Fprintf(os.Stderr, "\n")
			} else {
				return nil, fmt.Errorf("schema validation failed for %s: %w", path, err)
			}
		}
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// createSyntheticConfig creates a config for directories without genifest.yaml
// containing all .yaml and .yml files in the directory. This allows directories
// to be included in the configuration even without explicit genifest.yaml files.
func createSyntheticConfig(_ string) *Config {
	return &Config{
		Files: FilesConfig{
			Include: []string{"*.yml", "*.yaml"},
			Exclude: []string{"genifest.yml", "genifest.yaml"},
		},
	}
}

// loadDirectoryConfig loads a config from a directory, either from genifest.yaml or synthetic.
// If genifest.yaml exists, it loads that; otherwise it creates a synthetic config
// with all YAML files in the directory.
func loadDirectoryConfig(dirPath string, relativePath string, cloudHome string, mode ValidationMode) (configWithPath, error) {
	genifestPath := filepath.Join(dirPath, "genifest.yaml")
	var config *Config
	var err error

	if _, statErr := os.Stat(genifestPath); statErr == nil {
		// genifest.yaml exists, load it normally with validation
		config, err = loadConfigFileWithValidation(genifestPath, mode)
		if err != nil {
			return configWithPath{}, fmt.Errorf("failed to load %s: %w", genifestPath, err)
		}
	} else {
		// No genifest.yaml, create synthetic config
		config = createSyntheticConfig(dirPath)
	}

	return configWithPath{
		config:    config,
		path:      relativePath,
		depth:     strings.Count(relativePath, string(filepath.Separator)),
		cloudHome: cloudHome,
	}, nil
}

// loadMetadataPaths processes paths from metadata (Scripts, Manifests, Files).
// It loads each specified directory up to maxDepth levels, respecting the cloudHome boundary.
func (lc *loadingContext) loadMetadataPaths(basePath string, paths PathContexts, maxDepth int, cloudHome string) error {
	for _, pathCtx := range paths {
		fullPath := filepath.Join(basePath, pathCtx.Path)

		// Resolve to absolute path for consistency
		absPath, err := filepath.Abs(fullPath)
		if err != nil {
			continue // Skip invalid paths
		}

		// Check if directory exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			continue // Skip non-existent directories
		}

		// Load this directory and its subdirectories up to maxDepth
		if err := lc.loadDirectoryRecursive(absPath, 0, maxDepth, cloudHome); err != nil {
			return err
		}
	}
	return nil
}

// loadDirectoryRecursive loads a directory and its subdirectories up to maxDepth.
// It prevents infinite recursion by tracking processed directories and loads both
// explicit genifest.yaml files and creates synthetic configs for YAML-only directories.
func (lc *loadingContext) loadDirectoryRecursive(dirPath string, currentDepth, maxDepth int, cloudHome string) error {
	// Avoid processing the same directory twice
	if lc.processed[dirPath] {
		return nil
	}
	lc.processed[dirPath] = true

	// Get relative path from root
	relativePath, err := filepath.Rel(lc.rootDir, dirPath)
	if err != nil {
		return err
	}

	// Load the config for this directory
	configWP, err := loadDirectoryConfig(dirPath, relativePath, cloudHome, lc.validationMode)
	if err != nil {
		return err
	}

	lc.configs = append(lc.configs, configWP)

	// Process metadata from this config to load additional directories
	if err := lc.processConfigMetadata(configWP, dirPath); err != nil {
		return err
	}

	// If we haven't reached max depth, load subdirectories
	if currentDepth < maxDepth {
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				subDirPath := filepath.Join(dirPath, entry.Name())
				if err := lc.loadDirectoryRecursive(subDirPath, currentDepth+1, maxDepth, cloudHome); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// processConfigMetadata processes metadata from a loaded config to discover additional directories.
// It handles cloudHome changes and loads directories specified in Scripts, Manifests, and Files
// metadata with appropriate depth limits (Scripts: 0 levels, Manifests/Files: 1 level).
func (lc *loadingContext) processConfigMetadata(configWP configWithPath, configDir string) error {
	config := configWP.config
	effectiveCloudHome := configWP.cloudHome

	// Handle cloudHome changes
	if config.Metadata.CloudHome != "" {
		// Calculate new cloudHome relative to current config directory
		newCloudHome := filepath.Join(configDir, config.Metadata.CloudHome)
		absCloudHome, err := filepath.Abs(newCloudHome)
		if err == nil {
			effectiveCloudHome = absCloudHome
		}
	}

	// Load unified Paths directories with their configured depths
	for _, pathConfig := range config.Metadata.Paths {
		maxDepth := pathConfig.Depth // Use 0-based depth directly

		// Create PathContext slice for compatibility with loadMetadataPaths
		pathContexts := PathContexts{PathContext{Path: pathConfig.Path}}
		pathContexts[0].SetContextPath(configDir)

		if err := lc.loadMetadataPaths(configDir, pathContexts, maxDepth, effectiveCloudHome); err != nil {
			return err
		}
	}

	return nil
}

func mergeFilesWithRelativePaths(c *configWithPath, output []string, input []string) []string {
	// Merge files with proper relative paths
	for _, file := range input {
		var fullFilePath string
		if c.path == "." || c.path == "" {
			// Files from root directory
			fullFilePath = file
		} else {
			// Files from subdirectories - prefix with relative path
			fullFilePath = filepath.Join(c.path, file)
		}
		output = append(output, fullFilePath)
	}

	return output
}

// mergeConfigs combines multiple configurations into a single Config.
// It merges metadata paths, files, changes, and functions while preserving
// the context path information for proper scoping.
func mergeConfigs(configs []configWithPath, primaryHome string) *Config {
	// Sort by depth (shallower first) for metadata merging
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].depth < configs[j].depth
	})

	result := &Config{
		Metadata: MetaConfig{CloudHome: primaryHome},
		Groups:   GetDefaultGroups(), // Start with default groups
	}

	// Merge metadata (outer folders override inner folders for CloudHome)
	for _, c := range configs {
		// Only update CloudHome if this is from a shallower directory
		if c.depth == 0 && c.config.Metadata.CloudHome != "" {
			result.Metadata.CloudHome = c.cloudHome
		}

		// Merge path lists - these are cumulative across all configs
		// Fill in contextPath for each PathContext before merging
		configDir := filepath.Dir(c.path)
		if c.path == "." || c.path == "" {
			configDir = "."
		}

		// Process unified Paths
		for _, pathConfig := range c.config.Metadata.Paths {
			newConfig := PathConfig{
				Path:    pathConfig.Path,
				Scripts: pathConfig.Scripts,
				Files:   pathConfig.Files,
				Depth:   pathConfig.Depth,
			}
			newConfig.SetContextPath(configDir)
			result.Metadata.Paths = append(result.Metadata.Paths, newConfig)
		}
	}

	// Merge Files, Changes, and Functions across all configurations
	for _, c := range configs {
		result.Files.Include = mergeFilesWithRelativePaths(&c, result.Files.Include, c.config.Files.Include)
		result.Files.Exclude = mergeFilesWithRelativePaths(&c, result.Files.Exclude, c.config.Files.Exclude)

		// Set the path for change orders
		for _, change := range c.config.Changes {
			change.Path = c.path
			result.Changes = append(result.Changes, change)
		}

		// Set the path for function definitions
		for _, fn := range c.config.Functions {
			fn.path = c.path
			result.Functions = append(result.Functions, fn)
		}

		// Merge Groups - subordinate groups get directory prefixes applied
		if c.config.Groups != nil {
			configDir := filepath.Dir(c.path)
			if c.path == "." || c.path == "" {
				configDir = "."
			}

			// For root config, merge directly. For nested configs, apply directory scoping
			if c.depth == 0 {
				// Root config - merge groups directly
				for name, expressions := range c.config.Groups {
					result.Groups[name] = expressions
				}
			} else {
				// Nested config - apply directory prefixes
				result.Groups = result.Groups.MergeGroups(c.config.Groups, configDir)
			}
		}
	}

	return result
}
