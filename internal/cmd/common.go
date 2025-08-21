package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// printError prints a user-friendly error message and exits with status code 1.
// This prevents Cobra from showing usage information for application errors.
func printError(err error) {
	if err == nil {
		return
	}

	// Pretty print validation errors
	if isValidationError(err) {
		printValidationError(err)
		os.Exit(1)
		return
	}

	// Print other errors with a simple format
	fmt.Fprintf(os.Stderr, "❌ Error: %s\n", err.Error())
	os.Exit(1)
}

// isValidationError checks if an error is a validation-related error.
func isValidationError(err error) bool {
	// Check if it's our custom ValidationError type
	var ve *config.ValidationError
	if errors.As(err, &ve) {
		return true
	}

	// Fall back to string-based detection for other validation errors
	errStr := err.Error()
	return strings.Contains(errStr, "validation failed") ||
		strings.Contains(errStr, "configuration validation") ||
		strings.Contains(errStr, "function") ||
		strings.Contains(errStr, "parameter") ||
		strings.Contains(errStr, "argument") ||
		strings.Contains(errStr, "valueFrom")
}

// printValidationError prints a well-formatted validation error.
func printValidationError(err error) {
	// Check if it's our custom ValidationError type
	var ve *config.ValidationError
	if errors.As(err, &ve) {
		// Use the custom error's nicely formatted output
		fmt.Fprintf(os.Stderr, "%s\n", ve.Error())
		fmt.Fprintf(os.Stderr, "\n💡 Tip: Use 'genifest validate' to check your configuration\n")
		return
	}

	// Handle legacy validation errors
	fmt.Fprintf(os.Stderr, "❌ Configuration Validation Error\n\n")

	errStr := err.Error()

	// Remove common prefixes to make errors cleaner
	errStr = strings.TrimPrefix(errStr, "failed to load configuration: ")
	errStr = strings.TrimPrefix(errStr, "configuration validation failed: ")
	errStr = strings.TrimPrefix(errStr, "validation failed: ")

	// Split into main error and context
	if strings.Contains(errStr, ": ") {
		parts := strings.SplitN(errStr, ": ", 2)
		if len(parts) == 2 {
			context := parts[0]
			message := parts[1]

			fmt.Fprintf(os.Stderr, "Context: %s\n", context)
			fmt.Fprintf(os.Stderr, "Issue:   %s\n", message)
		} else {
			fmt.Fprintf(os.Stderr, "Issue: %s\n", errStr)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Issue: %s\n", errStr)
	}

	fmt.Fprintf(os.Stderr, "\n💡 Tip: Use 'genifest validate' to check your configuration\n")
}
