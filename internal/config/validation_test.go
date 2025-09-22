package config

import (
	"testing"
)

// TestValueFrom_ValidateWithContext tests ValueFrom union type validation.
func TestValueFrom_ValidateWithContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		valueFrom   ValueFrom
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid function call",
			valueFrom: ValueFrom{
				FunctionCall: &FunctionCall{
					Name: "my-func",
					Arguments: Arguments{
						{Name: "arg1", ValueFrom: ValueFrom{DefaultValue: &DefaultValue{Value: "test"}}},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid default value",
			valueFrom: ValueFrom{
				DefaultValue: &DefaultValue{Value: "test-value"},
			},
			expectError: false,
		},
		{
			name: "valid file inclusion",
			valueFrom: ValueFrom{
				FileInclusion: &FileInclusion{Source: "config.yaml"},
			},
			expectError: false,
		},
		{
			name: "valid basic template",
			valueFrom: ValueFrom{
				BasicTemplate: &BasicTemplate{
					String: "Hello $name",
					Variables: Arguments{
						{Name: "name", ValueFrom: ValueFrom{DefaultValue: &DefaultValue{Value: "World"}}},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid script exec",
			valueFrom: ValueFrom{
				ScriptExec: &ScriptExec{ExecCommand: "generate.sh"},
			},
			expectError: false,
		},
		{
			name: "valid argument ref",
			valueFrom: ValueFrom{
				ArgumentRef: &ArgumentRef{Name: "input"},
			},
			expectError: false,
		},
		{
			name: "valid document ref",
			valueFrom: ValueFrom{
				DocumentRef: &DocumentRef{KeySelector: ".metadata.name"},
			},
			expectError: false,
		},
		{
			name: "valid call pipeline",
			valueFrom: ValueFrom{
				CallPipeline: &CallPipeline{
					{
						ValueFrom: ValueFrom{DefaultValue: &DefaultValue{Value: "input"}},
						Output:    "step1",
					},
					{
						ValueFrom: ValueFrom{FunctionCall: &FunctionCall{Name: "process"}},
						Output:    "step2",
					},
				},
			},
			expectError: false,
		},

		// Error cases
		{
			name:        "empty ValueFrom",
			valueFrom:   ValueFrom{},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test exactly one field must be set in ValueFrom, but 0 fields are set",
		},
		{
			name: "multiple fields set",
			valueFrom: ValueFrom{
				DefaultValue:  &DefaultValue{Value: "test"},
				FileInclusion: &FileInclusion{Source: "test.yaml"},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test exactly one field must be set in ValueFrom, but 2 fields are set",
		},
		{
			name: "invalid function call - bad name",
			valueFrom: ValueFrom{
				FunctionCall: &FunctionCall{Name: "1invalid"},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test.call is \"1invalid\" which is not a valid identifier",
		},
		{
			name: "invalid default value - empty",
			valueFrom: ValueFrom{
				DefaultValue: &DefaultValue{Value: ""},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test.default value field is required",
		},
		{
			name: "invalid file inclusion - no source",
			valueFrom: ValueFrom{
				FileInclusion: &FileInclusion{App: "myapp"},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test.file source field is required",
		},
		{
			name: "invalid basic template - no string",
			valueFrom: ValueFrom{
				BasicTemplate: &BasicTemplate{},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test.template string field is required",
		},
		{
			name: "invalid script exec - no command",
			valueFrom: ValueFrom{
				ScriptExec: &ScriptExec{},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test.script exec field is required",
		},
		{
			name: "invalid argument ref - bad name",
			valueFrom: ValueFrom{
				ArgumentRef: &ArgumentRef{Name: "-invalid"},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test.argRef is \"-invalid\" which is not a valid identifier",
		},
		{
			name: "invalid document ref - no key selector",
			valueFrom: ValueFrom{
				DocumentRef: &DocumentRef{FileSelector: "*.yaml"},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test.documentRef keySelector is required",
		},
		{
			name: "invalid call pipeline - empty",
			valueFrom: ValueFrom{
				CallPipeline: &CallPipeline{},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test.pipeline call pipeline cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := &ValidationContext{
				PathBuilder: NewPathBuilder("test"),
				Filename:    "test.yaml",
				Functions: []FunctionDefinition{
					{Name: "my-func", ValueFrom: ValueFrom{DefaultValue: &DefaultValue{Value: "test"}}},
					{Name: "process", ValueFrom: ValueFrom{DefaultValue: &DefaultValue{Value: "processed"}}},
				},
			}
			err := tt.valueFrom.ValidateWithContext(ctx)

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

// TestParameter_ValidateWithContext tests parameter validation.
func TestParameter_ValidateWithContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		param       Parameter
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid optional parameter",
			param: Parameter{
				Name:     "my-param",
				Required: false,
				Default:  "default-value",
			},
			expectError: false,
		},
		{
			name: "valid required parameter",
			param: Parameter{
				Name:     "required-param",
				Required: true,
				Default:  "",
			},
			expectError: false,
		},
		{
			name: "valid single letter name",
			param: Parameter{
				Name:     "x",
				Required: false,
			},
			expectError: false,
		},

		// Error cases
		{
			name: "invalid name - empty",
			param: Parameter{
				Name:     "",
				Required: false,
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test is \"\" which is not a valid identifier",
		},
		{
			name: "invalid name - starts with number",
			param: Parameter{
				Name:     "1param",
				Required: false,
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test is \"1param\" which is not a valid identifier",
		},
		{
			name: "valid name - uppercase",
			param: Parameter{
				Name:     "MyParam",
				Required: false,
			},
			expectError: false,
			errorMsg:    "parameter name 'MyParam' is not a valid identifier",
		},
		{
			name: "required with default",
			param: Parameter{
				Name:     "bad-param",
				Required: true,
				Default:  "not-allowed",
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test is required and cannot have a default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := &ValidationContext{
				PathBuilder: NewPathBuilder("test"),
				Filename:    "test.yaml",
			}
			err := tt.param.ValidateWithContext(ctx)

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

// TestChangeOrder_ValidateWithContext tests change order validation.
func TestChangeOrder_ValidateWithContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		change      ChangeOrder
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid change order",
			change: ChangeOrder{
				DocumentRef: DocumentRef{
					KeySelector: ".metadata.name",
				},
				Tag: "deploy",
				ValueFrom: ValueFrom{
					DefaultValue: &DefaultValue{Value: "new-value"},
				},
			},
			expectError: false,
		},
		{
			name: "valid with empty tag",
			change: ChangeOrder{
				DocumentRef: DocumentRef{
					KeySelector: ".spec.replicas",
				},
				Tag: "",
				ValueFrom: ValueFrom{
					DefaultValue: &DefaultValue{Value: "3"},
				},
			},
			expectError: false,
		},

		// Error cases
		{
			name: "invalid document ref",
			change: ChangeOrder{
				DocumentRef: DocumentRef{
					// Missing required KeySelector
				},
				Tag: "deploy",
				ValueFrom: ValueFrom{
					DefaultValue: &DefaultValue{Value: "test"},
				},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test keySelector is required",
		},
		{
			name: "invalid tag",
			change: ChangeOrder{
				DocumentRef: DocumentRef{
					KeySelector: ".metadata.name",
				},
				Tag: "Deploy", // uppercase not allowed
				ValueFrom: ValueFrom{
					DefaultValue: &DefaultValue{Value: "test"},
				},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test is \"Deploy\" which is not a valid kebab-case tag",
		},
		{
			name: "invalid valueFrom",
			change: ChangeOrder{
				DocumentRef: DocumentRef{
					KeySelector: ".metadata.name",
				},
				Tag:       "deploy",
				ValueFrom: ValueFrom{
					// Empty - no field set
				},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test.valueFrom exactly one field must be set in ValueFrom, but 0 fields are set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := &ValidationContext{
				PathBuilder: NewPathBuilder("test"),
				Filename:    "test.yaml",
			}
			err := tt.change.ValidateWithContext(ctx)

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

// TestCallPipeline_ValidateWithContext tests call pipeline validation.
func TestCallPipeline_ValidateWithContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		pipeline    CallPipeline
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid single step pipeline",
			pipeline: CallPipeline{
				{
					ValueFrom: ValueFrom{DefaultValue: &DefaultValue{Value: "input"}},
					Output:    "result",
				},
			},
			expectError: false,
		},
		{
			name: "valid multi-step pipeline with function calls",
			pipeline: CallPipeline{
				{
					ValueFrom: ValueFrom{DefaultValue: &DefaultValue{Value: "input"}},
					Output:    "step1",
				},
				{
					ValueFrom: ValueFrom{FunctionCall: &FunctionCall{Name: "process"}},
					Output:    "step2",
				},
				{
					ValueFrom: ValueFrom{ScriptExec: &ScriptExec{ExecCommand: "format.sh"}},
					Output:    "final",
				},
			},
			expectError: false,
		},

		// Error cases
		{
			name:        "empty pipeline",
			pipeline:    CallPipeline{},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test call pipeline cannot be empty",
		},
		{
			name: "invalid output name",
			pipeline: CallPipeline{
				{
					ValueFrom: ValueFrom{DefaultValue: &DefaultValue{Value: "input"}},
					Output:    "1invalid",
				},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test[0] is \"1invalid\" which is not a valid identifier",
		},
		{
			name: "subsequent pipe not function or script",
			pipeline: CallPipeline{
				{
					ValueFrom: ValueFrom{DefaultValue: &DefaultValue{Value: "input"}},
					Output:    "step1",
				},
				{
					ValueFrom: ValueFrom{DefaultValue: &DefaultValue{Value: "another"}}, // Not allowed after first
					Output:    "step2",
				},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test[1].valueFrom must be either FunctionCall or ScriptExec for subsequent pipes",
		},
		{
			name: "valid pipeline with no output on final pipe",
			pipeline: CallPipeline{
				{
					ValueFrom: ValueFrom{DefaultValue: &DefaultValue{Value: "input"}},
					Output:    "step1",
				},
				{
					ValueFrom: ValueFrom{FunctionCall: &FunctionCall{Name: "process"}},
					Output:    "", // Final pipe without output - should be allowed
				},
			},
			expectError: false,
		},
		{
			name: "invalid pipeline with no output on non-final pipe",
			pipeline: CallPipeline{
				{
					ValueFrom: ValueFrom{DefaultValue: &DefaultValue{Value: "input"}},
					Output:    "", // Non-final pipe without output - should fail
				},
				{
					ValueFrom: ValueFrom{FunctionCall: &FunctionCall{Name: "process"}},
					Output:    "final",
				},
			},
			expectError: true,
			errorMsg:    "❌ test.yaml: .test[0] is required for non-final pipes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := &ValidationContext{
				PathBuilder: NewPathBuilder("test"),
				Filename:    "test.yaml",
				Functions: []FunctionDefinition{
					{Name: "my-func", ValueFrom: ValueFrom{DefaultValue: &DefaultValue{Value: "test"}}},
					{Name: "process", ValueFrom: ValueFrom{DefaultValue: &DefaultValue{Value: "processed"}}},
				},
			}
			err := tt.pipeline.ValidateWithContext(ctx)

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
