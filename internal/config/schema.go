package config

import (
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// ValidationMode determines how schema validation is performed.
type ValidationMode int

const (
	ValidationModePermissive ValidationMode = iota // Default: ignore unknown fields
	ValidationModeWarn                             // Warn about unknown fields but continue
	ValidationModeStrict                           // Fail on unknown fields
)

var (
	schemaLoader gojsonschema.JSONLoader
	schemaJSON   string
)

// InitSchema initializes the schema from the embedded JSON.
func InitSchema(schema string) {
	schemaJSON = schema
	schemaLoader = gojsonschema.NewStringLoader(schemaJSON)
}

// ValidationResult contains the results of schema validation.
type ValidationResult struct {
	Valid    bool
	Errors   []SchemaValidationError
	Warnings []ValidationWarning
}

// SchemaValidationError represents a schema validation error.
type SchemaValidationError struct {
	Path        string
	Message     string
	InvalidData interface{}
}

// ValidationWarning represents a schema validation warning (for unknown fields).
type ValidationWarning struct {
	Path    string
	Message string
	Field   string
}

// Error implements the error interface.
func (ve SchemaValidationError) Error() string {
	if ve.Path == "" {
		return ve.Message
	}
	return fmt.Sprintf("%s: %s", ve.Path, ve.Message)
}

// String returns a formatted warning message.
func (vw ValidationWarning) String() string {
	if vw.Path == "" {
		return fmt.Sprintf("unknown field '%s': %s", vw.Field, vw.Message)
	}
	return fmt.Sprintf("%s: unknown field '%s': %s", vw.Path, vw.Field, vw.Message)
}

// ValidateConfigWithSchema validates a configuration against the JSON schema.
func ValidateConfigWithSchema(data []byte, mode ValidationMode) (*ValidationResult, error) {
	// Convert YAML to JSON for schema validation
	var yamlData interface{}
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Convert to JSON-compatible format
	jsonData := convertYAMLToJSON(yamlData)
	documentLoader := gojsonschema.NewGoLoader(jsonData)

	// Perform schema validation
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	// Process validation results - start assuming valid, we'll adjust based on actual errors
	validationResult := &ValidationResult{
		Valid:    true, // We'll set this correctly based on our filtered errors
		Errors:   make([]SchemaValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Convert gojsonschema errors to our format
	for _, resultErr := range result.Errors() {
		validationError := SchemaValidationError{
			Path:        resultErr.Field(),
			Message:     resultErr.Description(),
			InvalidData: resultErr.Value(),
		}

		// Check if this is an "additional property" error (unknown field)
		if strings.Contains(resultErr.Type(), "additional_property") {
			if mode == ValidationModePermissive {
				continue // Ignore in permissive mode
			}

			warning := ValidationWarning{
				Path:    resultErr.Field(),
				Message: resultErr.Description(),
				Field:   extractFieldName(resultErr.Description()),
			}

			switch mode {
			case ValidationModeWarn:
				validationResult.Warnings = append(validationResult.Warnings, warning)
			case ValidationModeStrict:
				validationResult.Errors = append(validationResult.Errors, validationError)
			case ValidationModePermissive:
				// Already handled above, no action needed
			}
		} else {
			// Always report non-additional-property errors
			validationResult.Errors = append(validationResult.Errors, validationError)
		}
	}

	// Set validity based on whether we have any errors
	validationResult.Valid = len(validationResult.Errors) == 0

	return validationResult, nil
}

// convertYAMLToJSON converts YAML data to JSON-compatible format.
func convertYAMLToJSON(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[fmt.Sprintf("%v", k)] = convertYAMLToJSON(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convertYAMLToJSON(v)
		}
	}
	return i
}

// extractFieldName extracts the field name from an "additional property" error message.
func extractFieldName(message string) string {
	// Example: "Additional property foo is not allowed"
	if strings.Contains(message, "Additional property ") && strings.Contains(message, " is not allowed") {
		start := strings.Index(message, "Additional property ") + len("Additional property ")
		end := strings.Index(message[start:], " is not allowed")
		if end > 0 {
			return message[start : start+end]
		}
	}
	return "unknown"
}

// ValidateWithSchema is a convenience function that validates and returns appropriate errors.
func ValidateWithSchema(data []byte, mode ValidationMode) error {
	result, err := ValidateConfigWithSchema(data, mode)
	if err != nil {
		return err
	}

	if !result.Valid {
		var errors []string
		for _, err := range result.Errors {
			errors = append(errors, err.Error())
		}
		return fmt.Errorf("schema validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	// Return warnings as a special error type if there are any
	if len(result.Warnings) > 0 && mode == ValidationModeWarn {
		return &SchemaWarningsError{Warnings: result.Warnings}
	}

	return nil
}

// SchemaWarningsError is a special error type that indicates warnings were found.
type SchemaWarningsError struct {
	Warnings []ValidationWarning
}

// Error implements the error interface.
func (swe *SchemaWarningsError) Error() string {
	warnings := make([]string, 0, len(swe.Warnings))
	for _, w := range swe.Warnings {
		warnings = append(warnings, w.String())
	}
	return fmt.Sprintf("schema validation warnings:\n  - %s", strings.Join(warnings, "\n  - "))
}

// IsWarning returns true, indicating this is a warning rather than a hard error.
func (swe *SchemaWarningsError) IsWarning() bool {
	return true
}

// SchemaFileWarningsError represents warnings found in a specific file.
type SchemaFileWarningsError struct {
	Path     string
	Warnings []ValidationWarning
}

// Error implements the error interface.
func (sfw *SchemaFileWarningsError) Error() string {
	warnings := make([]string, 0, len(sfw.Warnings))
	for _, w := range sfw.Warnings {
		warnings = append(warnings, w.String())
	}
	return fmt.Sprintf("schema validation warnings in %s:\n  - %s", sfw.Path, strings.Join(warnings, "\n  - "))
}

// IsWarning returns true, indicating this is a warning rather than a hard error.
func (sfw *SchemaFileWarningsError) IsWarning() bool {
	return true
}
