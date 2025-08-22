package keysel

import (
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
			if len(selector.Pipeline) == 0 {
				t.Errorf("Parsed expression has no pipeline steps for %q", tt.selector)
			}

			// Validate pipeline structure
			for i, step := range selector.Pipeline {
				if step.Path == nil && step.Function == nil {
					t.Errorf("Pipeline step %d has neither path nor function for %q", i, tt.selector)
				}
			}
		})
	}
}
