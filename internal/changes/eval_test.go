package changes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/zostay/genifest/internal/config"
)

// setupTestContext creates a test evaluation context with common test data.
func setupTestContext(t *testing.T) (*EvalContext, string) {
	tempDir := t.TempDir()

	// Create test directories
	scriptsDir := filepath.Join(tempDir, "scripts")
	filesDir := filepath.Join(tempDir, "files")

	err := os.MkdirAll(scriptsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create scripts dir: %v", err)
	}

	err = os.MkdirAll(filesDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create files dir: %v", err)
	}

	// Create test functions
	functions := []config.FunctionDefinition{
		{
			Name: "test-func",
			Params: []config.Parameter{
				{Name: "param1", Required: true},
				{Name: "param2", Required: false, Default: "default-value"},
			},
			ValueFrom: config.ValueFrom{
				BasicTemplate: &config.BasicTemplate{
					String: "${param1}-${param2}",
					Variables: []config.Argument{
						{Name: "param1", ValueFrom: config.ValueFrom{ArgumentRef: &config.ArgumentRef{Name: "param1"}}},
						{Name: "param2", ValueFrom: config.ValueFrom{ArgumentRef: &config.ArgumentRef{Name: "param2"}}},
					},
				},
			},
		},
	}

	ctx := NewEvalContext(tempDir, scriptsDir, filesDir, functions)
	return ctx, tempDir
}

// TestDefaultValue tests evaluation of default values.
func TestDefaultValue(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)

	valueFrom := config.ValueFrom{
		DefaultValue: &config.DefaultValue{Value: "test-value"},
	}

	result, err := ctx.Evaluate(valueFrom)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", result)
	}
}

// TestArgumentRef tests evaluation of argument references.
func TestArgumentRef(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)
	ctx.SetVariable("test-arg", "argument-value")

	valueFrom := config.ValueFrom{
		ArgumentRef: &config.ArgumentRef{Name: "test-arg"},
	}

	result, err := ctx.Evaluate(valueFrom)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != "argument-value" {
		t.Errorf("Expected 'argument-value', got '%s'", result)
	}
}

// TestArgumentRefNotFound tests error handling for missing arguments.
func TestArgumentRefNotFound(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)

	valueFrom := config.ValueFrom{
		ArgumentRef: &config.ArgumentRef{Name: "missing-arg"},
	}

	_, err := ctx.Evaluate(valueFrom)
	if err == nil {
		t.Fatal("Expected error for missing argument, got none")
	}

	expectedMsg := "argument \"missing-arg\" not found in context"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestBasicTemplate tests template evaluation with variable substitution.
func TestBasicTemplate(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)

	valueFrom := config.ValueFrom{
		BasicTemplate: &config.BasicTemplate{
			String: "Hello ${name}, you are ${age} years old!",
			Variables: []config.Argument{
				{
					Name:      "name",
					ValueFrom: config.ValueFrom{DefaultValue: &config.DefaultValue{Value: "World"}},
				},
				{
					Name:      "age",
					ValueFrom: config.ValueFrom{DefaultValue: &config.DefaultValue{Value: "25"}},
				},
			},
		},
	}

	result, err := ctx.Evaluate(valueFrom)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "Hello World, you are 25 years old!"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestBasicTemplateWithDollarEscape tests dollar sign escaping in templates.
func TestBasicTemplateWithDollarEscape(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)

	valueFrom := config.ValueFrom{
		BasicTemplate: &config.BasicTemplate{
			String: "Price: $$${amount}",
			Variables: []config.Argument{
				{
					Name:      "amount",
					ValueFrom: config.ValueFrom{DefaultValue: &config.DefaultValue{Value: "100"}},
				},
			},
		},
	}

	result, err := ctx.Evaluate(valueFrom)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "Price: $100"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestFunctionCall tests function call evaluation.
func TestFunctionCall(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)

	valueFrom := config.ValueFrom{
		FunctionCall: &config.FunctionCall{
			Name: "test-func",
			Arguments: []config.Argument{
				{
					Name:      "param1",
					ValueFrom: config.ValueFrom{DefaultValue: &config.DefaultValue{Value: "hello"}},
				},
				{
					Name:      "param2",
					ValueFrom: config.ValueFrom{DefaultValue: &config.DefaultValue{Value: "world"}},
				},
			},
		},
	}

	result, err := ctx.Evaluate(valueFrom)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "hello-world"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestFunctionCallWithDefaults tests function call with default parameter values.
func TestFunctionCallWithDefaults(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)

	valueFrom := config.ValueFrom{
		FunctionCall: &config.FunctionCall{
			Name: "test-func",
			Arguments: []config.Argument{
				{
					Name:      "param1",
					ValueFrom: config.ValueFrom{DefaultValue: &config.DefaultValue{Value: "hello"}},
				},
				// param2 should use default value
			},
		},
	}

	result, err := ctx.Evaluate(valueFrom)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "hello-default-value"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestFunctionCallMissingRequired tests error handling for missing required parameters.
func TestFunctionCallMissingRequired(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)

	valueFrom := config.ValueFrom{
		FunctionCall: &config.FunctionCall{
			Name:      "test-func",
			Arguments: []config.Argument{
				// Missing required param1
			},
		},
	}

	_, err := ctx.Evaluate(valueFrom)
	if err == nil {
		t.Fatal("Expected error for missing required parameter, got none")
	}

	expectedMsg := "required parameter \"param1\" not provided for function \"test-func\""
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestFunctionCallNotFound tests error handling for unknown functions.
func TestFunctionCallNotFound(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)

	valueFrom := config.ValueFrom{
		FunctionCall: &config.FunctionCall{
			Name:      "unknown-func",
			Arguments: []config.Argument{},
		},
	}

	_, err := ctx.Evaluate(valueFrom)
	if err == nil {
		t.Fatal("Expected error for unknown function, got none")
	}

	expectedMsg := "function \"unknown-func\" not found"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestScriptExec tests script execution.
func TestScriptExec(t *testing.T) {
	t.Parallel()
	ctx, tempDir := setupTestContext(t)

	// Create a test script
	scriptPath := filepath.Join(tempDir, "scripts", "test-script.sh")
	scriptContent := `#!/bin/bash
echo "Hello from script: $1"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700) //nolint:gosec // need an executable for testing
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	valueFrom := config.ValueFrom{
		ScriptExec: &config.ScriptExec{
			ExecCommand: "test-script.sh",
			Args: []config.Argument{
				{
					Name:      "arg1",
					ValueFrom: config.ValueFrom{DefaultValue: &config.DefaultValue{Value: "world"}},
				},
			},
		},
	}

	result, err := ctx.Evaluate(valueFrom)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "Hello from script: world"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestScriptExecWithEnv tests script execution with environment variables.
func TestScriptExecWithEnv(t *testing.T) {
	t.Parallel()
	ctx, tempDir := setupTestContext(t)

	// Create a test script that uses environment variables
	scriptPath := filepath.Join(tempDir, "scripts", "env-script.sh")
	scriptContent := `#!/bin/bash
echo "ENV_VAR: $TEST_ENV_VAR"
`
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0700) //nolint:gosec // need an executable for testing
	if err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	valueFrom := config.ValueFrom{
		ScriptExec: &config.ScriptExec{
			ExecCommand: "env-script.sh",
			Env: []config.Argument{
				{
					Name:      "TEST_ENV_VAR",
					ValueFrom: config.ValueFrom{DefaultValue: &config.DefaultValue{Value: "test-value"}},
				},
			},
		},
	}

	result, err := ctx.Evaluate(valueFrom)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "ENV_VAR: test-value"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestScriptExecNotFound tests error handling for missing scripts.
func TestScriptExecNotFound(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)

	valueFrom := config.ValueFrom{
		ScriptExec: &config.ScriptExec{
			ExecCommand: "missing-script.sh",
		},
	}

	_, err := ctx.Evaluate(valueFrom)
	if err == nil {
		t.Fatal("Expected error for missing script, got none")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error about script not found, got: %v", err)
	}
}

// TestFileInclusion tests file inclusion.
func TestFileInclusion(t *testing.T) {
	t.Parallel()
	ctx, tempDir := setupTestContext(t)

	// Create a test file
	testContent := "This is test file content"
	filePath := filepath.Join(tempDir, "files", "test.txt")
	err := os.WriteFile(filePath, []byte(testContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	valueFrom := config.ValueFrom{
		FileInclusion: &config.FileInclusion{
			Source: "test.txt",
		},
	}

	result, err := ctx.Evaluate(valueFrom)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != testContent {
		t.Errorf("Expected '%s', got '%s'", testContent, result)
	}
}

// TestFileInclusionWithApp tests file inclusion with app subdirectory.
func TestFileInclusionWithApp(t *testing.T) {
	t.Parallel()
	ctx, tempDir := setupTestContext(t)

	// Create app subdirectory and test file
	appDir := filepath.Join(tempDir, "files", "myapp")
	err := os.MkdirAll(appDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	testContent := "App-specific content"
	filePath := filepath.Join(appDir, "config.yaml")
	err = os.WriteFile(filePath, []byte(testContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	valueFrom := config.ValueFrom{
		FileInclusion: &config.FileInclusion{
			App:    "myapp",
			Source: "config.yaml",
		},
	}

	result, err := ctx.Evaluate(valueFrom)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != testContent {
		t.Errorf("Expected '%s', got '%s'", testContent, result)
	}
}

// TestCallPipeline tests pipeline execution.
func TestCallPipeline(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)

	pipeline := config.CallPipeline{
		{
			ValueFrom: config.ValueFrom{DefaultValue: &config.DefaultValue{Value: "initial"}},
			Output:    "step1",
		},
		{
			ValueFrom: config.ValueFrom{
				BasicTemplate: &config.BasicTemplate{
					String: "${step1}-processed",
					Variables: []config.Argument{
						{
							Name:      "step1",
							ValueFrom: config.ValueFrom{ArgumentRef: &config.ArgumentRef{Name: "step1"}},
						},
					},
				},
			},
		},
	}

	valueFrom := config.ValueFrom{
		CallPipeline: &pipeline,
	}

	result, err := ctx.Evaluate(valueFrom)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "initial-processed"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

// TestEmptyValueFrom tests error handling for empty ValueFrom.
func TestEmptyValueFrom(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)

	valueFrom := config.ValueFrom{} // Empty ValueFrom

	_, err := ctx.Evaluate(valueFrom)
	if err == nil {
		t.Fatal("Expected error for empty ValueFrom, got none")
	}

	expectedMsg := "no ValueFrom type specified"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// TestDocumentRef tests document reference evaluation.
func TestDocumentRef(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)

	// Create a test YAML document
	yamlContent := `
apiVersion: v1
kind: Service
metadata:
  name: test-service
  namespace: default
spec:
  replicas: 3
  ports:
    - port: 80
      targetPort: 8080
  selector:
    app: test-app
`

	var doc yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &doc)
	if err != nil {
		t.Fatalf("Failed to parse test YAML: %v", err)
	}

	// Set the document in context
	ctx = ctx.WithDocument(&doc)

	testCases := []struct {
		name     string
		selector string
		expected string
	}{
		{
			name:     "simple field",
			selector: ".kind",
			expected: "Service",
		},
		{
			name:     "nested field",
			selector: ".metadata.name",
			expected: "test-service",
		},
		{
			name:     "deeper nested field",
			selector: ".spec.replicas",
			expected: "3",
		},
		{
			name:     "array access",
			selector: ".spec.ports[0].port",
			expected: "80",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			valueFrom := config.ValueFrom{
				DocumentRef: &config.DocumentRef{
					KeySelector: tc.selector,
				},
			}

			result, err := ctx.Evaluate(valueFrom)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestDocumentRefErrors tests error cases for document reference.
func TestDocumentRefErrors(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)

	// Test with no document
	valueFrom := config.ValueFrom{
		DocumentRef: &config.DocumentRef{
			KeySelector: ".metadata.name",
		},
	}

	_, err := ctx.Evaluate(valueFrom)
	if err == nil {
		t.Fatal("Expected error for missing document, got none")
	}

	expectedMsg := "no current document available for document reference"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}

	// Test with invalid selector
	yamlContent := `
metadata:
  name: test
`
	var doc yaml.Node
	err = yaml.Unmarshal([]byte(yamlContent), &doc)
	if err != nil {
		t.Fatalf("Failed to parse test YAML: %v", err)
	}

	ctx = ctx.WithDocument(&doc)

	valueFrom = config.ValueFrom{
		DocumentRef: &config.DocumentRef{
			KeySelector: ".metadata.missing",
		},
	}

	_, err = ctx.Evaluate(valueFrom)
	if err == nil {
		t.Fatal("Expected error for missing key, got none")
	}

	if !strings.Contains(err.Error(), "field \"missing\" not found") {
		t.Errorf("Expected error about missing field, got: %v", err)
	}
}

// TestContextImmutability tests that context operations don't modify the original.
func TestContextImmutability(t *testing.T) {
	t.Parallel()
	ctx, _ := setupTestContext(t)
	ctx.SetVariable("original", "value")

	// Test WithVariables doesn't modify original
	newCtx := ctx.WithVariables(map[string]string{"new": "variable"})

	// Original should still have only the original variable
	if _, exists := ctx.GetVariable("new"); exists {
		t.Error("Original context was modified by WithVariables")
	}

	// New context should have both variables
	if val, exists := newCtx.GetVariable("original"); !exists || val != "value" {
		t.Error("New context doesn't have original variable")
	}
	if val, exists := newCtx.GetVariable("new"); !exists || val != "variable" {
		t.Error("New context doesn't have new variable")
	}
}
