package config

import (
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
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", rootDir, err)
	}

	// Initialize loading context
	lc := &loadingContext{
		configs:     make([]configWithPath, 0),
		rootDir:     abs,
		processed:   make(map[string]bool),
		primaryHome: abs,
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

type configWithPath struct {
	config    *Config
	path      string
	depth     int
	cloudHome string // The effective cloudHome for this config
}

// loadingContext tracks the state during recursive config loading.
type loadingContext struct {
	configs     []configWithPath
	rootDir     string
	processed   map[string]bool // Track processed directories to avoid cycles
	primaryHome string          // The primary cloudHome from root
}

func loadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// createSyntheticConfig creates a config for directories without genifest.yaml
// containing all .yaml and .yml files in the directory.
func createSyntheticConfig(dir string) (*Config, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), ".yaml") || strings.HasSuffix(strings.ToLower(name), ".yml") {
			files = append(files, name)
		}
	}

	return &Config{
		Files: files,
	}, nil
}

// loadDirectoryConfig loads a config from a directory, either from genifest.yaml or synthetic.
func loadDirectoryConfig(dirPath string, relativePath string, cloudHome string) (configWithPath, error) {
	genifestPath := filepath.Join(dirPath, "genifest.yaml")
	var config *Config
	var err error

	if _, statErr := os.Stat(genifestPath); statErr == nil {
		// genifest.yaml exists, load it normally
		config, err = loadConfigFile(genifestPath)
		if err != nil {
			return configWithPath{}, fmt.Errorf("failed to load %s: %w", genifestPath, err)
		}
	} else {
		// No genifest.yaml, create synthetic config
		config, err = createSyntheticConfig(dirPath)
		if err != nil {
			return configWithPath{}, fmt.Errorf("failed to create synthetic config for %s: %w", dirPath, err)
		}
	}

	return configWithPath{
		config:    config,
		path:      relativePath,
		depth:     strings.Count(relativePath, string(filepath.Separator)),
		cloudHome: cloudHome,
	}, nil
}

// loadMetadataPaths processes paths from metadata (Scripts, Manifests, Files).
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
	configWP, err := loadDirectoryConfig(dirPath, relativePath, cloudHome)
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

	// Load Scripts directories (single level only)
	if err := lc.loadMetadataPaths(configDir, config.Metadata.Scripts, 0, effectiveCloudHome); err != nil {
		return err
	}

	// Load Manifests directories (two levels)
	if err := lc.loadMetadataPaths(configDir, config.Metadata.Manifests, 1, effectiveCloudHome); err != nil {
		return err
	}

	// Load Files directories (two levels)
	if err := lc.loadMetadataPaths(configDir, config.Metadata.Files, 1, effectiveCloudHome); err != nil {
		return err
	}

	return nil
}

func mergeConfigs(configs []configWithPath, primaryHome string) *Config {
	// Sort by depth (shallower first) for metadata merging
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].depth < configs[j].depth
	})

	result := &Config{
		Metadata: MetaConfig{CloudHome: primaryHome},
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

		// Process Scripts
		for _, scriptCtx := range c.config.Metadata.Scripts {
			newCtx := PathContext{
				contextPath: configDir,
				Path:        scriptCtx.Path,
			}
			result.Metadata.Scripts = append(result.Metadata.Scripts, newCtx)
		}

		// Process Manifests
		for _, manifestCtx := range c.config.Metadata.Manifests {
			newCtx := PathContext{
				contextPath: configDir,
				Path:        manifestCtx.Path,
			}
			result.Metadata.Manifests = append(result.Metadata.Manifests, newCtx)
		}

		// Process Files
		for _, fileCtx := range c.config.Metadata.Files {
			newCtx := PathContext{
				contextPath: configDir,
				Path:        fileCtx.Path,
			}
			result.Metadata.Files = append(result.Metadata.Files, newCtx)
		}
	}

	// Merge Files, Changes, and Functions across all configurations
	for _, c := range configs {
		result.Files = append(result.Files, c.config.Files...)

		// Set the path for change orders
		for _, change := range c.config.Changes {
			change.path = c.path
			result.Changes = append(result.Changes, change)
		}

		// Set the path for function definitions
		for _, fn := range c.config.Functions {
			fn.path = c.path
			result.Functions = append(result.Functions, fn)
		}
	}

	return result
}
