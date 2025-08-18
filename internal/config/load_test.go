package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadFromDirectory_GuestbookExample tests loading the guestbook example configuration.
func TestLoadFromDirectory_GuestbookExample(t *testing.T) {
	t.Parallel()
	// Get the absolute path to the examples/guestbook directory
	projectRoot := getProjectRoot(t)
	guestbookDir := filepath.Join(projectRoot, "examples", "guestbook")

	// Load the configuration
	config, err := LoadFromDirectory(guestbookDir)
	if err != nil {
		t.Fatalf("Failed to load guestbook configuration: %v", err)
	}

	// Validate basic structure
	if config == nil {
		t.Fatal("Config is nil")
	}

	// Check metadata
	if config.Metadata.CloudHome == "" {
		t.Error("CloudHome should be set")
	}

	// Verify metadata paths are populated
	if len(config.Metadata.Scripts) == 0 {
		t.Error("Expected scripts metadata to be populated")
	}
	if len(config.Metadata.Manifests) == 0 {
		t.Error("Expected manifests metadata to be populated")
	}
	if len(config.Metadata.Files) == 0 {
		t.Error("Expected files metadata to be populated")
	}

	// Check that files are loaded from the configuration
	// This includes both files explicitly listed in genifest.yaml and files
	// discovered through synthetic configs in subdirectories
	expectedFiles := []string{
		"manifests/guestbook/frontend-deployment.yaml",
		"manifests/guestbook/backend-deployment.yaml",
		"manifests/guestbook/frontend-service.yaml",
		"manifests/guestbook/backend-service.yaml",
		"manifests/guestbook/ingress.yaml",
		"manifests/guestbook/configmap.yaml",
		"manifests/guestbook/secret.yaml",
		"manifests/postgres/deployment.yaml",
		"manifests/postgres/service.yaml",
		"manifests/postgres/configmap.yaml",
		"manifests/postgres/secret.yaml",
		"manifests/postgres/pvc.yaml",
	}

	// Check that we have at least the expected files (there may be more from synthetic configs)
	if len(config.Files) < len(expectedFiles) {
		t.Errorf("Expected at least %d files, got %d", len(expectedFiles), len(config.Files))
	}

	// Verify each expected file is present
	fileMap := make(map[string]bool)
	for _, file := range config.Files {
		fileMap[file] = true
	}

	for _, expectedFile := range expectedFiles {
		if !fileMap[expectedFile] {
			t.Errorf("Expected file %q not found in config.Files", expectedFile)
		}
	}

	// Check functions are loaded
	expectedFunctions := []string{"get-replicas", "get-image-tag", "get-database-host"}
	if len(config.Functions) != len(expectedFunctions) {
		t.Errorf("Expected %d functions, got %d", len(expectedFunctions), len(config.Functions))
	}

	// Verify each expected function is present
	funcMap := make(map[string]bool)
	for _, fn := range config.Functions {
		funcMap[fn.Name] = true
	}

	for _, expectedFunc := range expectedFunctions {
		if !funcMap[expectedFunc] {
			t.Errorf("Expected function %q not found in config.Functions", expectedFunc)
		}
	}

	// Check that changes are loaded
	if len(config.Changes) == 0 {
		t.Error("Expected changes to be loaded from configuration")
	}

	// Verify some specific changes
	var foundTaggedChange, foundImageChange bool
	for _, change := range config.Changes {
		if change.Tag == "production" || change.Tag == "staging" {
			foundTaggedChange = true
		}
		if change.KeySelector == ".spec.template.spec.containers[0].image" {
			foundImageChange = true
		}
	}

	if !foundTaggedChange {
		t.Error("Expected to find tagged changes for production/staging")
	}
	if !foundImageChange {
		t.Error("Expected to find changes for container image")
	}
}

// TestLoadFromDirectory_GuestbookSyntheticConfigs tests that synthetic configs are created
// for directories without genifest.yaml files.
func TestLoadFromDirectory_GuestbookSyntheticConfigs(t *testing.T) {
	t.Parallel()
	projectRoot := getProjectRoot(t)
	guestbookDir := filepath.Join(projectRoot, "examples", "guestbook")

	config, err := LoadFromDirectory(guestbookDir)
	if err != nil {
		t.Fatalf("Failed to load guestbook configuration: %v", err)
	}

	// The guestbook/manifests/guestbook and guestbook/manifests/postgres directories
	// don't have genifest.yaml files, so synthetic configs should be created.
	// We should find YAML files from those directories in the final merged config.

	foundGuestbookManifests := false
	foundPostgresManifests := false

	for _, file := range config.Files {
		if filepath.Dir(file) == "manifests/guestbook" {
			foundGuestbookManifests = true
		}
		if filepath.Dir(file) == "manifests/postgres" {
			foundPostgresManifests = true
		}
	}

	if !foundGuestbookManifests {
		t.Error("Expected to find manifests from guestbook directory (synthetic config)")
	}
	if !foundPostgresManifests {
		t.Error("Expected to find manifests from postgres directory (synthetic config)")
	}
}

// TestLoadFromDirectory_GuestbookValidation tests that the loaded configuration passes validation.
func TestLoadFromDirectory_GuestbookValidation(t *testing.T) {
	t.Parallel()
	projectRoot := getProjectRoot(t)
	guestbookDir := filepath.Join(projectRoot, "examples", "guestbook")

	config, err := LoadFromDirectory(guestbookDir)
	if err != nil {
		t.Fatalf("Failed to load guestbook configuration: %v", err)
	}

	// The configuration should pass validation
	err = config.Validate()
	if err != nil {
		t.Errorf("Guestbook configuration failed validation: %v", err)
	}
}

// TestLoadFromDirectory_GuestbookFunctionScoping tests that functions are properly scoped.
func TestLoadFromDirectory_GuestbookFunctionScoping(t *testing.T) {
	t.Parallel()
	projectRoot := getProjectRoot(t)
	guestbookDir := filepath.Join(projectRoot, "examples", "guestbook")

	config, err := LoadFromDirectory(guestbookDir)
	if err != nil {
		t.Fatalf("Failed to load guestbook configuration: %v", err)
	}

	// Create a validation context to test function lookup
	ctx := &ValidationContext{
		CloudHome:   config.Metadata.CloudHome,
		Functions:   config.Functions,
		CurrentPath: "manifests/guestbook",
	}

	// Functions defined in the root should be accessible from subdirectories
	fn, found := ctx.LookupFunction("get-replicas")
	if !found {
		t.Error("get-replicas function should be accessible from manifests/guestbook")
	}
	if found && fn.Name != "get-replicas" {
		t.Errorf("Expected function name get-replicas, got %s", fn.Name)
	}

	_, found = ctx.LookupFunction("get-image-tag")
	if !found {
		t.Error("get-image-tag function should be accessible from manifests/guestbook")
	}

	_, found = ctx.LookupFunction("get-database-host")
	if !found {
		t.Error("get-database-host function should be accessible from manifests/guestbook")
	}

	// Test from a deeper path
	ctx.CurrentPath = "manifests/postgres"
	_, found = ctx.LookupFunction("get-replicas")
	if !found {
		t.Error("get-replicas function should be accessible from manifests/postgres")
	}
}

// getProjectRoot finds the project root directory for testing.
func getProjectRoot(t *testing.T) string {
	t.Parallel()
	// Start from the current working directory and walk up to find the project root
	cwd, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Look for go.mod file to identify project root
	dir := cwd
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	// Fallback: assume we're in internal/config and go up two levels
	return filepath.Join(cwd, "..", "..")
}
