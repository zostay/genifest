package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestPathContext_YAMLMarshalling tests the YAML marshalling and unmarshalling of PathContext.
func TestPathContext_YAMLMarshalling(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		pc       PathContext
		expected string
	}{
		{
			name:     "simple path",
			pc:       PathContext{Path: "scripts"},
			expected: "scripts\n",
		},
		{
			name:     "nested path",
			pc:       PathContext{Path: "manifests/app1"},
			expected: "manifests/app1\n",
		},
		{
			name:     "empty path",
			pc:       PathContext{Path: ""},
			expected: "\"\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Test marshalling
			data, err := yaml.Marshal(tt.pc)
			if err != nil {
				t.Fatalf("Failed to marshal PathContext: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("Marshal result mismatch:\nexpected: %q\ngot:      %q", tt.expected, string(data))
			}

			// Test unmarshalling
			var pc PathContext
			err = yaml.Unmarshal(data, &pc)
			if err != nil {
				t.Fatalf("Failed to unmarshal PathContext: %v", err)
			}
			if pc.Path != tt.pc.Path {
				t.Errorf("Unmarshal result mismatch: expected %q, got %q", tt.pc.Path, pc.Path)
			}
			// contextPath should be empty after unmarshalling (filled by loader)
			if pc.contextPath != "" {
				t.Errorf("contextPath should be empty after unmarshalling, got %q", pc.contextPath)
			}
		})
	}
}

// TestPathContext_Methods tests the PathContext methods.
func TestPathContext_Methods(t *testing.T) {
	t.Parallel()
	pc := PathContext{
		contextPath: "/base/dir",
		Path:        "scripts",
	}

	// Test ContextPath getter
	if pc.ContextPath() != "/base/dir" {
		t.Errorf("ContextPath() = %q, expected %q", pc.ContextPath(), "/base/dir")
	}

	// Test SetContextPath
	pc.SetContextPath("/new/dir")
	if pc.contextPath != "/new/dir" {
		t.Errorf("SetContextPath failed: expected %q, got %q", "/new/dir", pc.contextPath)
	}
}

// TestPathContexts_YAMLMarshalling tests the YAML marshalling of PathContexts slice.
func TestPathContexts_YAMLMarshalling(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		pcs      PathContexts
		expected string
	}{
		{
			name:     "empty slice",
			pcs:      PathContexts{},
			expected: "[]\n",
		},
		{
			name: "single path",
			pcs: PathContexts{
				{Path: "scripts"},
			},
			expected: "- scripts\n",
		},
		{
			name: "multiple paths",
			pcs: PathContexts{
				{Path: "scripts"},
				{Path: "manifests"},
				{Path: "files/app1"},
			},
			expected: "- scripts\n- manifests\n- files/app1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Test marshalling
			data, err := yaml.Marshal(tt.pcs)
			if err != nil {
				t.Fatalf("Failed to marshal PathContexts: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("Marshal result mismatch:\nexpected: %q\ngot:      %q", tt.expected, string(data))
			}

			// Test unmarshalling
			var pcs PathContexts
			err = yaml.Unmarshal(data, &pcs)
			if err != nil {
				t.Fatalf("Failed to unmarshal PathContexts: %v", err)
			}
			if len(pcs) != len(tt.pcs) {
				t.Errorf("Length mismatch: expected %d, got %d", len(tt.pcs), len(pcs))
			}
			for i, pc := range pcs {
				if i >= len(tt.pcs) {
					break
				}
				if pc.Path != tt.pcs[i].Path {
					t.Errorf("Path[%d] mismatch: expected %q, got %q", i, tt.pcs[i].Path, pc.Path)
				}
				// contextPath should be empty after unmarshalling
				if pc.contextPath != "" {
					t.Errorf("contextPath[%d] should be empty after unmarshalling, got %q", i, pc.contextPath)
				}
			}
		})
	}
}

// TestIsValidIdentifier tests the identifier validation function.
func TestIsValidIdentifier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid identifiers
		{"single letter", "a", true},
		{"lowercase word", "script", true},
		{"with numbers", "app1", true},
		{"with hyphens", "my-script", true},
		{"ends with number", "test123", true},
		{"underscore", "my_script", true},     // Underscores allowed in words
		{"uppercase letters", "Script", true}, // Mixed case allowed

		// Invalid identifiers
		{"only number", "1", false},
		{"starts with number", "1script", false},
		{"complex valid", "app-server-1", false},
		{"empty string", "", false},
		{"starts with hyphen", "-script", false},
		{"ends with hyphen", "script-", false},
		{"double hyphen", "my--script", false}, // Double hyphens not allowed
		{"spaces", "my script", false},
		{"special chars", "script!", false},
		{"only hyphen", "-", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isValidIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("isValidIdentifier(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestIsValidTag tests the kebab-case tag validation function.
func TestIsValidTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid tags (same rules as identifiers, plus empty is allowed)
		{"empty string", "", true}, // tags are optional
		{"single letter", "a", true},
		{"lowercase word", "tag", true},
		{"with numbers", "v1", true},
		{"with hyphens", "my-tag", true},
		{"complex valid", "deploy-v1-2", true},
		{"starts with number", "1tag", true}, // Tags can start with numbers
		{"underscore", "my_tag", true},       // Underscores not allowed in tags

		// Invalid tags
		{"starts with hyphen", "-tag", false},
		{"ends with hyphen", "tag-", false},
		{"uppercase letters", "Tag", false}, // Tags must be lowercase
		{"double hyphen", "my--tag", false}, // Double hyphens not allowed
		{"spaces", "my tag", false},
		{"special chars", "tag!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isValidTag(tt.input)
			if result != tt.expected {
				t.Errorf("isValidTag(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestValidationContext_LookupFunction tests function lookup and scoping rules.
func TestValidationContext_LookupFunction(t *testing.T) {
	t.Parallel()
	functions := []FunctionDefinition{
		{Name: "root-func", path: "."},
		{Name: "app-func", path: "apps/app1"},
		{Name: "shared-func", path: "."},         // root level
		{Name: "shared-func", path: "apps"},      // apps level (should shadow root)
		{Name: "shared-func", path: "apps/app1"}, // app1 level (should shadow apps)
		{Name: "deep-func", path: "apps/app1/env/prod"},
	}

	tests := []struct {
		name         string
		currentPath  string
		funcName     string
		expectFound  bool
		expectedPath string
	}{
		// Function available from same path
		{"same path root", ".", "root-func", true, "."},
		{"same path app", "apps/app1", "app-func", true, "apps/app1"},

		// Function available from parent path
		{"child can access parent", "apps/app1", "root-func", true, "."},
		{"grandchild can access grandparent", "apps/app1/env", "root-func", true, "."},

		// Function not available from child path (scoping rule)
		{"parent cannot access child", ".", "app-func", false, ""},
		{"sibling cannot access sibling", "apps/app2", "app-func", false, ""},

		// Function shadowing (closest path wins)
		{"root sees root version", ".", "shared-func", true, "."},
		{"apps sees root version", "apps", "shared-func", true, "."}, // "." and "apps" both have depth 0, but "." comes first
		{"app1 sees app1 version", "apps/app1", "shared-func", true, "apps/app1"},
		{"app1/env sees app1 version", "apps/app1/env", "shared-func", true, "apps/app1"},

		// Deep function access
		{"deep function from same path", "apps/app1/env/prod", "deep-func", true, "apps/app1/env/prod"},
		{"deep function from parent path", "apps/app1/env/prod/config", "deep-func", true, "apps/app1/env/prod"},
		{"deep function not accessible from parent", "apps/app1/env", "deep-func", false, ""},

		// Non-existent function
		{"non-existent function", "apps/app1", "missing-func", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}

			ctx := &ValidationContext{
				CloudHome:   wd,
				Functions:   functions,
				CurrentPath: tt.currentPath,
			}

			fn, found := ctx.LookupFunction(tt.funcName)
			if found != tt.expectFound {
				t.Errorf("LookupFunction found = %v, expected %v", found, tt.expectFound)
				return
			}

			if tt.expectFound {
				if fn == nil {
					t.Error("Expected function to be non-nil when found=true")
					return
				}
				if fn.path != tt.expectedPath {
					// Debug: show what functions are available
					t.Logf("Available functions for %q from %q:", tt.funcName, tt.currentPath)
					for _, f := range functions {
						if f.Name == tt.funcName && ctx.isFunctionAvailable(f.path) {
							depth := strings.Count(f.path, string(filepath.Separator))
							t.Logf("  - path: %q, depth: %d", f.path, depth)
						}
					}
					t.Errorf("Function path = %q, expected %q", fn.path, tt.expectedPath)
				}
				if fn.Name != tt.funcName {
					t.Errorf("Function name = %q, expected %q", fn.Name, tt.funcName)
				}
			} else if fn != nil {
				t.Error("Expected function to be nil when found=false")
			}
		})
	}
}

// TestValidationContext_isFunctionAvailable tests the function availability logic.
func TestValidationContext_isFunctionAvailable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		currentPath  string
		functionPath string
		expected     bool
	}{
		// Same path
		{"same path root", ".", ".", true},
		{"same path nested", "apps/app1", "apps/app1", true},

		// Parent-child relationships
		{"parent available from child", "apps/app1", ".", true},
		{"parent available from grandchild", "apps/app1/env", ".", true},
		{"parent available from deep child", "apps/app1/env/prod", "apps", true},

		// Child not available from parent
		{"child not available from parent", ".", "apps/app1", false},
		{"grandchild not available from grandparent", ".", "apps/app1/env", false},

		// Sibling relationships
		{"sibling not available", "apps/app1", "apps/app2", false},
		{"cousin not available", "apps/app1/env", "apps/app2/env", false},

		// Edge cases with path cleaning
		{"with trailing slash", "apps/app1/", "apps", true},
		{"with dot segments", "apps/app1/./env", "apps/app1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}

			ctx := &ValidationContext{
				CloudHome:   wd,
				CurrentPath: tt.currentPath,
			}

			result := ctx.isFunctionAvailable(tt.functionPath)
			if result != tt.expected {
				t.Errorf("isFunctionAvailable(%q, %q) = %v, expected %v",
					tt.currentPath, tt.functionPath, result, tt.expected)
			}
		})
	}
}

// TestMetaConfig_validatePathWithinHome tests path security validation.
func TestMetaConfig_validatePathWithinHome(t *testing.T) {
	t.Parallel()
	mc := &MetaConfig{
		CloudHome: "/home/user/project",
	}

	tests := []struct {
		name        string
		path        string
		pathType    string
		expectError bool
		errorMsg    string
	}{
		// Valid paths
		{"empty path", "", "script", false, ""},
		{"simple relative path", "scripts", "script", false, ""},
		{"nested relative path", "apps/app1/scripts", "script", false, ""},
		{"dot path", ".", "script", false, ""},
		{"dot-dot within bounds", "apps/../scripts", "script", false, ""},

		// Invalid paths - absolute paths
		{"absolute path unix", "/etc/passwd", "script", true, "script path '/etc/passwd' must be relative, not absolute"},
		// Note: Windows absolute paths like "C:\Windows" are only detected as absolute on Windows

		// Invalid paths - escaping cloudHome
		{"dot-dot escape", "..", "script", true, "script path '..' attempts to reference parent directories outside of cloudHome"},
		{"dot-dot prefix", "../../../etc", "script", true, "script path '../../../etc' attempts to reference parent directories outside of cloudHome"},
		{"nested dot-dot escape", "apps/../../etc", "script", true, "script path 'apps/../../etc' attempts to reference parent directories outside of cloudHome"},
		{"slash dot-dot", "apps/../..", "script", true, "script path 'apps/../..' attempts to reference parent directories outside of cloudHome"},
		{"backslash dot-dot", "apps\\..\\..", "script", true, "script path 'apps\\..\\..' attempts to reference parent directories outside of cloudHome"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := mc.validatePathWithinHome(mc.CloudHome, tt.path, tt.pathType)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path %q, but got none", tt.path)
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("Error message mismatch:\nexpected: %q\ngot:      %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error for path %q: %v", tt.path, err)
			}
		})
	}
}

// TestMetaConfig_ValidateWithContext tests metadata validation with path security.
func TestMetaConfig_ValidateWithContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		meta        MetaConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid metadata",
			meta: MetaConfig{
				CloudHome: "/home/user/project",
				Paths: PathConfigs{
					{Path: "scripts", Scripts: true},
					{Path: "tools", Scripts: true},
					{Path: "manifests", Files: true},
					{Path: "files", Files: true},
				},
			},
			expectError: false,
		},
		{
			name: "empty cloudHome (allowed)",
			meta: MetaConfig{
				CloudHome: "",
				Paths: PathConfigs{
					{Path: "scripts", Scripts: true},
				},
			},
			expectError: false,
		},
		{
			name: "invalid script path",
			meta: MetaConfig{
				CloudHome: "/home/user/project",
				Paths: PathConfigs{
					{Path: "../../../etc/passwd", Scripts: true},
				},
			},
			expectError: true,
			errorMsg:    "path '../../../etc/passwd' attempts to reference parent directories outside of cloudHome",
		},
		{
			name: "invalid manifest path",
			meta: MetaConfig{
				CloudHome: "/home/user/project",
				Paths: PathConfigs{
					{Path: "/etc/passwd", Files: true},
				},
			},
			expectError: true,
			errorMsg:    "path '/etc/passwd' must be relative, not absolute",
		},
		{
			name: "invalid file path",
			meta: MetaConfig{
				CloudHome: "/home/user/project",
				Paths: PathConfigs{
					{Path: "..", Files: true},
				},
			},
			expectError: true,
			errorMsg:    "path '..' attempts to reference parent directories outside of cloudHome",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.meta.ValidateWithContext(nil)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("Error message mismatch:\nexpected: %q\ngot:      %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
