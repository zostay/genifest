package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zostay/genifest/internal/config"
)

// ProjectInfo contains resolved project directory and loaded configuration.
type ProjectInfo struct {
	WorkDir string
	Config  *config.Config
}

// resolveProjectDirectory resolves an optional project directory argument to an absolute path.
// If projectDir is empty, uses the current working directory.
// Returns the absolute path to the project directory.
func resolveProjectDirectory(projectDir string) (string, error) {
	var workDir string
	var err error

	if projectDir != "" {
		// Use provided directory argument
		workDir = projectDir
		// Convert to absolute path if relative
		if !filepath.IsAbs(workDir) {
			currentDir, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("failed to get current directory: %w", err)
			}
			workDir = filepath.Join(currentDir, workDir)
		}
	} else {
		// Use current working directory
		workDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	return workDir, nil
}

// validateProjectDirectory checks if a directory exists and contains a genifest.yaml file.
func validateProjectDirectory(workDir string) error {
	// Verify the directory exists
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", workDir)
	}

	// Verify genifest.yaml exists
	configPath := filepath.Join(workDir, "genifest.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("genifest.yaml not found in directory: %s", workDir)
	}

	return nil
}

// loadProjectConfiguration resolves the project directory and loads the configuration.
// This is the main utility function that combines directory resolution, validation, and config loading.
func loadProjectConfiguration(projectDir string) (*ProjectInfo, error) {
	// Resolve the project directory
	workDir, err := resolveProjectDirectory(projectDir)
	if err != nil {
		return nil, err
	}

	// Validate the project directory
	if err := validateProjectDirectory(workDir); err != nil {
		return nil, err
	}

	// Load configuration
	cfg, err := config.LoadFromDirectory(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return &ProjectInfo{
		WorkDir: workDir,
		Config:  cfg,
	}, nil
}
