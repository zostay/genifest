package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/zostay/genifest/internal/config"
)

// TestMatchesGlob tests the glob pattern matching function.
func TestMatchesGlob(t *testing.T) {
	t.Parallel()
	tests := []struct {
		pattern string
		str     string
		want    bool
	}{
		{"production", "production", true},
		{"prod*", "production", true},
		{"staging", "production", false},
		{"*", "anything", true},
		{"", "", true},
		{"test-*", "test-env", true},
		{"test-*", "production", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.str, func(t *testing.T) {
			t.Parallel()
			got := matchesGlob(tt.pattern, tt.str)
			if got != tt.want {
				t.Errorf("matchesGlob(%q, %q) = %v, want %v", tt.pattern, tt.str, got, tt.want)
			}
		})
	}
}

// TestDetermineTags tests the tag determination logic.
func TestDetermineTags(t *testing.T) { //nolint:tparallel // Subtests cannot run in parallel due to global variable modification
	t.Parallel()
	cfg := &config.Config{
		Changes: []config.ChangeOrder{
			{Tag: "production"},
			{Tag: "staging"},
			{Tag: ""}, // untagged
			{Tag: "test"},
		},
	}

	tests := []struct {
		name        string
		includeTags []string
		excludeTags []string
		want        []string
	}{
		{
			name:        "no filters",
			includeTags: nil,
			excludeTags: nil,
			want:        []string{"production", "staging", "test", ""},
		},
		{
			name:        "include production only",
			includeTags: []string{"production"},
			excludeTags: nil,
			want:        []string{"production"},
		},
		{
			name:        "exclude production",
			includeTags: nil,
			excludeTags: []string{"production"},
			want:        []string{"staging", "test", ""},
		},
		{
			name:        "include prod* exclude staging",
			includeTags: []string{"prod*", "test"},
			excludeTags: []string{"staging"},
			want:        []string{"production", "test"},
		},
	}

	for _, tt := range tests { //nolint:paralleltest // Cannot use t.Parallel due to global variable modification
		t.Run(tt.name, func(t *testing.T) {
			// Note: Cannot use t.Parallel() here because we modify global variables
			// Set global variables for test
			oldInclude := includeTags
			oldExclude := excludeTags
			defer func() {
				includeTags = oldInclude
				excludeTags = oldExclude
			}()

			includeTags = tt.includeTags
			excludeTags = tt.excludeTags

			got := determineTags(cfg)

			// Sort both slices for comparison since order doesn't matter
			gotMap := make(map[string]bool)
			for _, tag := range got {
				gotMap[tag] = true
			}
			wantMap := make(map[string]bool)
			for _, tag := range tt.want {
				wantMap[tag] = true
			}

			if len(gotMap) != len(wantMap) {
				t.Errorf("determineTags() = %v, want %v", got, tt.want)
				return
			}

			for tag := range wantMap {
				if !gotMap[tag] {
					t.Errorf("determineTags() missing tag %q, got %v, want %v", tag, got, tt.want)
				}
			}
		})
	}
}

// TestSetValueInDocument tests YAML document value setting.
func TestSetValueInDocument(t *testing.T) {
	t.Parallel()
	// Create a test YAML document
	yamlContent := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: app
          image: test:v1
          ports:
            - port: 80
`

	var doc yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &doc)
	if err != nil {
		t.Fatalf("Failed to parse test YAML: %v", err)
	}

	tests := []struct {
		name        string
		keySelector string
		value       string
		wantChanged bool
		wantError   bool
	}{
		{
			name:        "set replicas",
			keySelector: ".spec.replicas",
			value:       "3",
			wantChanged: true,
			wantError:   false,
		},
		{
			name:        "set container image",
			keySelector: ".spec.template.spec.containers[0].image",
			value:       "test:v2",
			wantChanged: true,
			wantError:   false,
		},
		{
			name:        "set port",
			keySelector: ".spec.template.spec.containers[0].ports[0].port",
			value:       "8080",
			wantChanged: true,
			wantError:   false,
		},
		{
			name:        "invalid path",
			keySelector: ".spec.invalid.path",
			value:       "value",
			wantChanged: false,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Make a copy of the document for each test
			var testDoc yaml.Node
			err := yaml.Unmarshal([]byte(yamlContent), &testDoc)
			if err != nil {
				t.Fatalf("Failed to parse test YAML: %v", err)
			}

			changed, err := setValueInDocument(&testDoc, tt.keySelector, tt.value)

			if tt.wantError && err == nil {
				t.Errorf("setValueInDocument() expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("setValueInDocument() unexpected error: %v", err)
			}
			if changed != tt.wantChanged {
				t.Errorf("setValueInDocument() changed = %v, want %v", changed, tt.wantChanged)
			}
		})
	}
}

// TestWriteYAMLFile tests YAML file writing.
func TestWriteYAMLFile(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.yaml")

	// Create test documents
	yamlContent := `
name: test
value: 123
`
	var doc yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &doc)
	if err != nil {
		t.Fatalf("Failed to parse test YAML: %v", err)
	}

	documents := []yaml.Node{doc}

	// Write the file
	err = writeYAMLFile(testFile, documents, 0600)
	if err != nil {
		t.Fatalf("writeYAMLFile() error = %v", err)
	}

	// Verify the file was written
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("writeYAMLFile() did not create file")
	}

	// Read back the content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read back file: %v", err)
	}

	// Verify content contains expected values
	contentStr := string(content)
	if !strings.Contains(contentStr, "name: test") {
		t.Errorf("Output file missing expected content 'name: test', got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "value: 123") {
		t.Errorf("Output file missing expected content 'value: 123', got: %s", contentStr)
	}
}

// TestCLIBasicFunctionality tests the basic CLI workflow.
func TestCLIBasicFunctionality(t *testing.T) {
	t.Parallel()
	// This test requires the guestbook example to be present
	projectRoot := getProjectRoot(t)
	guestbookDir := filepath.Join(projectRoot, "examples", "guestbook")

	// Change to guestbook directory
	oldDir, _ := os.Getwd()
	defer func() {
		_ = os.Chdir(oldDir)
	}()

	err := os.Chdir(guestbookDir)
	if err != nil {
		t.Skipf("Skipping test: guestbook example not found at %s", guestbookDir)
	}

	// Check if genifest.yaml exists
	if _, err := os.Stat("genifest.yaml"); os.IsNotExist(err) {
		t.Skip("Skipping test: genifest.yaml not found in guestbook directory")
	}

	// Test that we can load the configuration without errors
	cfg, err := config.LoadFromDirectory(".")
	if err != nil {
		t.Fatalf("Failed to load guestbook configuration: %v", err)
	}

	// Verify we have some changes defined
	if len(cfg.Changes) == 0 {
		t.Error("Expected guestbook config to have changes defined")
	}

	// Verify we have functions defined
	if len(cfg.Functions) == 0 {
		t.Error("Expected guestbook config to have functions defined")
	}

	// Test tag determination
	oldInclude := includeTags
	oldExclude := excludeTags
	defer func() {
		includeTags = oldInclude
		excludeTags = oldExclude
	}()

	includeTags = []string{"production"}
	excludeTags = nil

	tags := determineTags(cfg)
	found := false
	for _, tag := range tags {
		if tag == "production" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'production' tag to be included when using --include-tags production")
	}
}

// getProjectRoot finds the project root directory for testing.
func getProjectRoot(t *testing.T) string {
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

	// Fallback: assume we're in internal/cmd and go up three levels
	return filepath.Join(cwd, "..", "..", "..")
}
