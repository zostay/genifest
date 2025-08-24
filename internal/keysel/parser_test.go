package keysel

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		selector  string
		shouldErr bool
	}{
		{
			name:     "root selector",
			selector: ".",
		},
		{
			name:     "simple field access",
			selector: ".spec",
		},
		{
			name:     "nested field access",
			selector: ".spec.replicas",
		},
		{
			name:     "field with array index",
			selector: ".spec.containers[0]",
		},
		{
			name:     "field with string key",
			selector: ".metadata.labels[\"app\"]",
		},
		{
			name:     "array slice",
			selector: ".items[1:3]",
		},
		{
			name:     "array slice start only",
			selector: ".items[1:]",
		},
		{
			name:     "array slice end only",
			selector: ".items[:3]",
		},
		{
			name:     "complex nested access",
			selector: ".spec.template.spec.containers[0].name",
		},
		{
			name:     "field followed by bracket with dot key",
			selector: ".data.[\"app.yaml\"]",
		},
		{
			name:     "array iteration",
			selector: ".spec.containers[]",
		},
		{
			name:     "select function",
			selector: ".spec.containers[] | select(.name == \"frontend\")",
		},
		{
			name:     "pipeline with field access",
			selector: ".spec.containers[] | select(.name == \"frontend\") | .image",
		},
		{
			name:     "complex guestbook example",
			selector: ".spec.template.spec.containers[] | select(.name == \"backend\") | .image",
		},
		{
			name:     "alternative operator with string",
			selector: ".metadata.annotations[\"missing\"] // \"default-value\"",
		},
		{
			name:     "alternative operator with nested path",
			selector: ".spec.replicas // \"1\"",
		},
		{
			name:     "pipeline with alternative",
			selector: ".spec.containers[] | select(.name == \"app\") | .image // \"default:latest\"",
		},
		{
			name:      "invalid selector",
			selector:  ".spec[",
			shouldErr: true,
		},
		{
			name:      "invalid function call",
			selector:  ".spec.containers[] | select(",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			parser, err := NewParser()
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			selector, err := parser.ParseSelector(tt.selector)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected parsing to fail for %q, but it succeeded", tt.selector)
				}
				return
			}

			if err != nil {
				t.Errorf("Failed to parse selector %q: %v", tt.selector, err)
				return
			}

			if selector == nil {
				t.Errorf("Parsed expression is nil for %q", tt.selector)
				return
			}

			// Basic validation - expression should have pipeline steps
			if selector.Left == nil {
				t.Errorf("Parsed expression has no left pipeline for %q", tt.selector)
				return
			}
			if len(selector.Left.Steps) == 0 {
				t.Errorf("Parsed expression has no pipeline steps for %q", tt.selector)
			}

			// Validate pipeline structure
			for i, step := range selector.Left.Steps {
				if step.Path == nil && step.Function == nil {
					t.Errorf("Pipeline step %d has neither path nor function for %q", i, tt.selector)
				}
			}

			// Test alternative operator validation for specific cases
			if strings.Contains(tt.selector, "//") {
				if selector.Alternative == nil {
					t.Errorf("Expected alternative value for selector %q with // operator", tt.selector)
				}
			} else {
				if selector.Alternative != nil {
					t.Errorf("Unexpected alternative value for selector %q without // operator", tt.selector)
				}
			}
		})
	}
}
