package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zostay/genifest/internal/config"
)

// stripANSI removes ANSI escape codes from a string
func stripANSI(str string) string {
	// Simple function to strip ANSI escape codes
	// Handles \\033[...m pattern
	result := ""
	inEscape := false
	for i, r := range str {
		if r == '\033' && i+1 < len(str) && str[i+1] == '[' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result += string(r)
	}
	return result
}

func TestValidationSummaryError_Error(t *testing.T) {
	tests := []struct {
		name       string
		errorCount int
		expected   string
	}{
		{
			name:       "single error",
			errorCount: 1,
			expected:   "validation failed with 1 error(s)",
		},
		{
			name:       "multiple errors",
			errorCount: 5,
			expected:   "validation failed with 5 error(s)",
		},
		{
			name:       "zero errors",
			errorCount: 0,
			expected:   "validation failed with 0 error(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ValidationSummaryError{ErrorCount: tt.errorCount}
			if err.Error() != tt.expected {
				t.Errorf("Error() = %q, expected %q", err.Error(), tt.expected)
			}
		})
	}
}

func TestIsValueFromEmpty(t *testing.T) {
	tests := []struct {
		name      string
		valueFrom config.ValueFrom
		expected  bool
	}{
		{
			name:      "empty ValueFrom",
			valueFrom: config.ValueFrom{},
			expected:  true,
		},
		{
			name: "ValueFrom with DefaultValue",
			valueFrom: config.ValueFrom{
				DefaultValue: &config.DefaultValue{Value: "test"},
			},
			expected: false,
		},
		{
			name: "ValueFrom with FunctionCall",
			valueFrom: config.ValueFrom{
				FunctionCall: &config.FunctionCall{Name: "test-func"},
			},
			expected: false,
		},
		{
			name: "ValueFrom with ArgumentRef",
			valueFrom: config.ValueFrom{
				ArgumentRef: &config.ArgumentRef{Name: "test-arg"},
			},
			expected: false,
		},
		{
			name: "ValueFrom with DocumentRef",
			valueFrom: config.ValueFrom{
				DocumentRef: &config.DocumentRef{KeySelector: ".test"},
			},
			expected: false,
		},
		{
			name: "ValueFrom with BasicTemplate",
			valueFrom: config.ValueFrom{
				BasicTemplate: &config.BasicTemplate{String: "test"},
			},
			expected: false,
		},
		{
			name: "ValueFrom with ScriptExec",
			valueFrom: config.ValueFrom{
				ScriptExec: &config.ScriptExec{ExecCommand: "test.sh"},
			},
			expected: false,
		},
		{
			name: "ValueFrom with FileInclusion",
			valueFrom: config.ValueFrom{
				FileInclusion: &config.FileInclusion{Source: "test.yaml"},
			},
			expected: false,
		},
		{
			name: "ValueFrom with CallPipeline",
			valueFrom: config.ValueFrom{
				CallPipeline: &config.CallPipeline{
					{Output: "test", ValueFrom: config.ValueFrom{DefaultValue: &config.DefaultValue{Value: "test"}}},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValueFromEmpty(tt.valueFrom)
			if result != tt.expected {
				t.Errorf("isValueFromEmpty() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestValidateConfiguration_Success(t *testing.T) {
	// Create a temporary directory with valid configuration
	tempDir := t.TempDir()
	
	// Create genifest.yaml
	configContent := `metadata:
  cloudHome: "."
files: []
functions:
  - name: "test-function"
    valueFrom:
      default:
        value: "test-value"
changes:
  - fileSelector: "*.yaml"
    keySelector: ".spec.replicas"
    valueFrom:
      default:
        value: "3"`

	configPath := filepath.Join(tempDir, "genifest.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run validation
	err = validateConfiguration(tempDir)

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	output, _ := io.ReadAll(r)
	outputStr := string(output)

	// Check result
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check output contains expected elements
	expectedElements := []string{
		"🔍",
		"Validating configuration in",
		"Summary:",
		"0 file(s) managed",
		"1 change(s) defined",
		"1 function(s) defined",
		"✅",
		"Configuration validation successful!",
	}

	cleanOutput := stripANSI(outputStr)
	for _, element := range expectedElements {
		if !strings.Contains(cleanOutput, element) {
			t.Errorf("Expected output to contain %q, but it didn't.\nClean Output: %s", element, cleanOutput)
		}
	}
}

func TestValidateConfiguration_ValidationErrors(t *testing.T) {
	// Create a temporary directory with invalid configuration (missing files)
	tempDir := t.TempDir()
	
	// Create genifest.yaml with references to non-existent files
	configContent := `metadata:
  cloudHome: "."
files: 
  - "missing1.yaml"
  - "missing2.yaml"
functions:
  - name: "test-function"
    valueFrom:
      default:
        value: "test-value"
changes:
  - fileSelector: "*.yaml"
    keySelector: ".spec.replicas"
    valueFrom:
      default:
        value: "3"`

	configPath := filepath.Join(tempDir, "genifest.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run validation
	err = validateConfiguration(tempDir)

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	output, _ := io.ReadAll(r)
	outputStr := string(output)

	// Check result - should be ValidationSummaryError
	if err == nil {
		t.Errorf("Expected ValidationSummaryError, got no error")
	} else if summaryErr, ok := err.(*ValidationSummaryError); !ok {
		t.Errorf("Expected *ValidationSummaryError, got %T: %v", err, err)
	} else if summaryErr.ErrorCount != 2 {
		t.Errorf("Expected 2 errors, got %d", summaryErr.ErrorCount)
	}

	// Check output contains expected elements
	expectedElements := []string{
		"🔍",
		"Validating configuration in",
		"Summary:",
		"2 file(s) managed",
		"1 change(s) defined", 
		"1 function(s) defined",
		"❌",
		"Configuration validation failed with 2 error(s):",
		"referenced file does not exist: missing1.yaml",
		"referenced file does not exist: missing2.yaml",
		"💡",
		"Fix these issues and run 'genifest validate' again",
	}

	cleanOutput := stripANSI(outputStr)
	for _, element := range expectedElements {
		if !strings.Contains(cleanOutput, element) {
			t.Errorf("Expected output to contain %q, but it didn't.\nClean Output: %s", element, cleanOutput)
		}
	}
}

func TestValidateConfiguration_ConfigLoadingValidationError(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectedError string
	}{
		{
			name: "invalid function name",
			configContent: `metadata:
  cloudHome: "."
files: []
functions:
  - name: "1invalid-name"
    valueFrom:
      default:
        value: "test-value"
changes: []`,
			expectedError: "is not a valid identifier",
		},
		{
			name: "missing keySelector",
			configContent: `metadata:
  cloudHome: "."
files: []
functions: []
changes:
  - fileSelector: "*.yaml"
    valueFrom:
      default:
        value: "3"`,
			expectedError: "keySelector is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			
			configPath := filepath.Join(tempDir, "genifest.yaml")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run validation
			err = validateConfiguration(tempDir)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout
			output, _ := io.ReadAll(r)
			outputStr := string(output)

			// Check result - should be ValidationSummaryError
			if err == nil {
				t.Errorf("Expected ValidationSummaryError, got no error")
			} else if summaryErr, ok := err.(*ValidationSummaryError); !ok {
				t.Errorf("Expected *ValidationSummaryError, got %T: %v", err, err)
			} else if summaryErr.ErrorCount != 1 {
				t.Errorf("Expected 1 error, got %d", summaryErr.ErrorCount)
			}

			// Check output contains expected elements
			expectedElements := []string{
				"🔍",
				"Validating configuration in",
				"❌",
				"Configuration validation failed with 1 error:",
				tt.expectedError,
				"💡",
				"Fix these issues and run 'genifest validate' again",
			}

			cleanOutput := stripANSI(outputStr)
			for _, element := range expectedElements {
				if !strings.Contains(cleanOutput, element) {
					t.Errorf("Expected output to contain %q, but it didn't.\nClean Output: %s", element, cleanOutput)
				}
			}

			// Should NOT contain summary since config loading failed
			unexpectedElements := []string{
				"Summary:",
				"file(s) managed",
				"change(s) defined",
				"function(s) defined",
			}

			for _, element := range unexpectedElements {
				if strings.Contains(cleanOutput, element) {
					t.Errorf("Expected output NOT to contain %q, but it did.\nClean Output: %s", element, cleanOutput)
				}
			}
		})
	}
}

func TestValidateConfiguration_AdditionalValidationChecks(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		expectedErrors []string
	}{
		{
			name: "duplicate function names",
			configContent: `metadata:
  cloudHome: "."
files: []
functions:
  - name: "duplicate"
    valueFrom:
      default:
        value: "test1"
  - name: "duplicate"
    valueFrom:
      default:
        value: "test2"
changes: []`,
			expectedErrors: []string{
				"function 1: duplicate function name 'duplicate'",
			},
		},
		{
			name: "missing referenced files",
			configContent: `metadata:
  cloudHome: "."
files: 
  - "missing1.yaml"
  - "missing2.yaml"
functions: []
changes: []`,
			expectedErrors: []string{
				"referenced file does not exist: missing1.yaml",
				"referenced file does not exist: missing2.yaml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory
			tempDir := t.TempDir()
			
			configPath := filepath.Join(tempDir, "genifest.yaml")
			err := os.WriteFile(configPath, []byte(tt.configContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run validation
			err = validateConfiguration(tempDir)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout
			output, _ := io.ReadAll(r)
			outputStr := string(output)

			// Check that we got a ValidationSummaryError
			if err == nil {
				t.Errorf("Expected ValidationSummaryError, got no error")
			} else if summaryErr, ok := err.(*ValidationSummaryError); !ok {
				t.Errorf("Expected *ValidationSummaryError, got %T: %v", err, err)
			} else if summaryErr.ErrorCount != len(tt.expectedErrors) {
				t.Errorf("Expected %d errors, got %d", len(tt.expectedErrors), summaryErr.ErrorCount)
			}

			// Check that all expected errors are in the output
			for _, expectedError := range tt.expectedErrors {
				if !strings.Contains(stripANSI(outputStr), expectedError) {
					t.Errorf("Expected output to contain error %q, but it didn't.\nClean Output: %s", expectedError, stripANSI(outputStr))
				}
			}

			// Check basic structure elements are present
			expectedStructure := []string{
				"🔍",
				"Validating configuration in",
				"Summary:",
				"❌",
				fmt.Sprintf("Configuration validation failed with %d error(s):", len(tt.expectedErrors)),
				"💡",
				"Fix these issues and run 'genifest validate' again",
			}

			for _, element := range expectedStructure {
				if !strings.Contains(stripANSI(outputStr), element) {
					t.Errorf("Expected output to contain %q, but it didn't.\nClean Output: %s", element, stripANSI(outputStr))
				}
			}
		})
	}
}

func TestValidateConfiguration_WithTags(t *testing.T) {
	// Create a temporary directory with configuration that has tags
	tempDir := t.TempDir()
	
	configContent := `metadata:
  cloudHome: "."
files: []
functions:
  - name: "test-function"
    valueFrom:
      default:
        value: "test-value"
changes:
  - tag: "production"
    fileSelector: "*.yaml"
    keySelector: ".spec.replicas"
    valueFrom:
      default:
        value: "3"
  - tag: "staging"
    fileSelector: "*.yaml"
    keySelector: ".spec.image"
    valueFrom:
      default:
        value: "app:latest"
  - tag: "production"
    fileSelector: "service.yaml"
    keySelector: ".spec.type"
    valueFrom:
      default:
        value: "LoadBalancer"`

	configPath := filepath.Join(tempDir, "genifest.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run validation
	err = validateConfiguration(tempDir)

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	output, _ := io.ReadAll(r)
	outputStr := string(output)

	// Check result
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check output contains tag information
	expectedElements := []string{
		"🔍",
		"Validating configuration in",
		"Summary:",
		"0 file(s) managed",
		"3 change(s) defined",
		"1 function(s) defined",
		"2 unique tag(s) used", // Should show 2 unique tags (production, staging)
		"✅",
		"Configuration validation successful!",
	}

	cleanOutput := stripANSI(outputStr)
	for _, element := range expectedElements {
		if !strings.Contains(cleanOutput, element) {
			t.Errorf("Expected output to contain %q, but it didn't.\nClean Output: %s", element, cleanOutput)
		}
	}
}

func TestValidateConfiguration_NonExistentDirectory(t *testing.T) {
	// Test with non-existent directory
	nonExistentDir := "/path/that/does/not/exist"

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run validation
	err := validateConfiguration(nonExistentDir)

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	output, _ := io.ReadAll(r)
	outputStr := string(output)

	// Check result - should be a regular error (not ValidationSummaryError)
	if err == nil {
		t.Errorf("Expected error for non-existent directory, got none")
	} else if _, ok := err.(*ValidationSummaryError); ok {
		t.Errorf("Expected regular error, got ValidationSummaryError: %v", err)
	}

	// Output should be minimal since we can't load the config
	if strings.Contains(stripANSI(outputStr), "🔍") {
		t.Errorf("Should not show validation message for directory that doesn't exist.\nOutput: %s", outputStr)
	}
}

func TestValidateConfiguration_NoGeniefestaYaml(t *testing.T) {
	// Create directory without genifest.yaml
	tempDir := t.TempDir()

	// Capture stdout  
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run validation
	err := validateConfiguration(tempDir)

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	output, _ := io.ReadAll(r)
	outputStr := string(output)

	// Check result - should be a regular error
	if err == nil {
		t.Errorf("Expected error for missing genifest.yaml, got none")
	} else if _, ok := err.(*ValidationSummaryError); ok {
		t.Errorf("Expected regular error, got ValidationSummaryError: %v", err)
	}

	// Should contain error about missing genifest.yaml
	if !strings.Contains(err.Error(), "genifest.yaml not found") {
		t.Errorf("Expected error about missing genifest.yaml, got: %v", err)
	}

	// Output should be minimal
	if strings.Contains(stripANSI(outputStr), "🔍") {
		t.Errorf("Should not show validation message when genifest.yaml is missing.\nOutput: %s", outputStr)
	}
}

// Test helper to capture both stdout and stderr
func captureOutput(fn func()) (stdout, stderr string) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	// Create channels to collect output
	outCh := make(chan string)
	errCh := make(chan string)

	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rOut)
		outCh <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rErr)
		errCh <- buf.String()
	}()

	fn()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return <-outCh, <-errCh
}