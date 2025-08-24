package config

import (
	"errors"
	"testing"
)

func TestSchemaValidation(t *testing.T) {
	t.Parallel()
	// Initialize schema for testing
	InitSchema(testSchema)

	tests := []struct {
		name   string
		yaml   string
		mode   ValidationMode
		valid  bool
		errors int
		warns  int
	}{
		{
			name: "valid basic config",
			yaml: `
metadata:
  cloudHome: "."
  paths:
    - path: "manifests"
      files: true
      depth: 1
functions:
  - name: "test-func"
    valueFrom:
      default:
        value: "test"
`,
			mode:   ValidationModeStrict,
			valid:  true,
			errors: 0,
			warns:  0,
		},
		{
			name: "unknown fields in permissive mode",
			yaml: `
metadata:
  cloudHome: "."
  unknownField: "ignored"
  paths:
    - path: "manifests"
      files: true
`,
			mode:   ValidationModePermissive,
			valid:  true,
			errors: 0,
			warns:  0,
		},
		{
			name: "unknown fields in warning mode",
			yaml: `
metadata:
  cloudHome: "."
  unknownField: "warning"
  paths:
    - path: "manifests"
      files: true
`,
			mode:   ValidationModeWarn,
			valid:  true,
			errors: 0,
			warns:  1,
		},
		{
			name: "unknown fields in strict mode",
			yaml: `
metadata:
  cloudHome: "."
  unknownField: "error"
  paths:
    - path: "manifests"
      files: true
`,
			mode:   ValidationModeStrict,
			valid:  false,
			errors: 1,
			warns:  0,
		},
		{
			name: "invalid function name",
			yaml: `
functions:
  - name: "Invalid-Name-With-Capital"
    valueFrom:
      default:
        value: "test"
`,
			mode:   ValidationModePermissive,
			valid:  false,
			errors: 1,
			warns:  0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ValidateConfigWithSchema([]byte(tt.yaml), tt.mode)
			if err != nil {
				t.Fatalf("ValidateConfigWithSchema failed: %v", err)
			}

			if result.Valid != tt.valid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.valid, result.Valid)
			}

			if len(result.Errors) != tt.errors {
				t.Errorf("Expected %d errors, got %d: %v", tt.errors, len(result.Errors), result.Errors)
			}

			if len(result.Warnings) != tt.warns {
				t.Errorf("Expected %d warnings, got %d: %v", tt.warns, len(result.Warnings), result.Warnings)
			}
		})
	}
}

func TestValidateWithSchema(t *testing.T) {
	t.Parallel()
	// Initialize schema for testing
	InitSchema(testSchema)

	t.Run("valid config returns no error", func(t *testing.T) {
		t.Parallel()
		yaml := `
metadata:
  cloudHome: "."
functions:
  - name: "test"
    valueFrom:
      default:
        value: "test"
`
		err := ValidateWithSchema([]byte(yaml), ValidationModeStrict)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("invalid config returns error", func(t *testing.T) {
		t.Parallel()
		yaml := `
metadata:
  cloudHome: "."
  invalidField: "error"
`
		err := ValidateWithSchema([]byte(yaml), ValidationModeStrict)
		if err == nil {
			t.Error("Expected error, got none")
		}
	})

	t.Run("warnings mode returns special error", func(t *testing.T) {
		t.Parallel()
		yaml := `
metadata:
  cloudHome: "."
  warningField: "warn"
`
		err := ValidateWithSchema([]byte(yaml), ValidationModeWarn)
		if err == nil {
			t.Error("Expected warning error, got none")
		}

		var warningErr *SchemaWarningsError
		if !errors.As(err, &warningErr) {
			t.Errorf("Expected SchemaWarningsError, got: %T", err)
		} else if !warningErr.IsWarning() {
			t.Error("Expected IsWarning() to return true")
		}
	})
}

// testSchema is a minimal schema for testing.
const testSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "metadata": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "cloudHome": {
          "type": "string"
        },
        "paths": {
          "type": "array",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "path": {"type": "string"},
              "files": {"type": "boolean"},
              "scripts": {"type": "boolean"},
              "depth": {"type": "integer"}
            }
          }
        }
      }
    },
    "functions": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["name", "valueFrom"],
        "properties": {
          "name": {
            "type": "string",
            "pattern": "^[a-z][a-z0-9-]*[a-z0-9]$|^[a-z]$"
          },
          "valueFrom": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "default": {
                "type": "object",
                "additionalProperties": false,
                "required": ["value"],
                "properties": {
                  "value": {"type": "string"}
                }
              }
            }
          }
        }
      }
    }
  }
}`
