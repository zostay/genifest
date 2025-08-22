package keysel

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestEvaluateSelector(t *testing.T) {
	t.Parallel()
	// Sample YAML document for testing
	yamlContent := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  labels:
    app: test
    version: "1.0"
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: web
          image: nginx:latest
          ports:
            - containerPort: 80
        - name: sidecar
          image: busybox:latest
  selector:
    matchLabels:
      app: test
`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &node)
	if err != nil {
		t.Fatalf("Failed to parse test YAML: %v", err)
	}

	evaluator := NewEvaluator()

	tests := []struct {
		name      string
		selector  string
		expected  string
		shouldErr bool
	}{
		{
			name:     "root access",
			selector: ".",
			expected: "apiVersion: apps/v1\nkind: Deployment", // Should contain the root content
		},
		{
			name:     "simple field access",
			selector: ".kind",
			expected: "Deployment",
		},
		{
			name:     "nested field access",
			selector: ".metadata.name",
			expected: "test-deployment",
		},
		{
			name:     "numeric field access",
			selector: ".spec.replicas",
			expected: "3",
		},
		{
			name:     "quoted label access",
			selector: ".metadata.labels.app",
			expected: "test",
		},
		{
			name:     "array index access",
			selector: ".spec.template.spec.containers[0].name",
			expected: "web",
		},
		{
			name:     "second array element",
			selector: ".spec.template.spec.containers[1].name",
			expected: "sidecar",
		},
		{
			name:     "port access in nested array",
			selector: ".spec.template.spec.containers[0].ports[0].containerPort",
			expected: "80",
		},
		{
			name:      "nonexistent field",
			selector:  ".nonexistent",
			shouldErr: true,
		},
		{
			name:      "array index out of bounds",
			selector:  ".spec.template.spec.containers[5]",
			shouldErr: true,
		},
		{
			name:      "invalid field on non-mapping",
			selector:  ".spec.replicas.invalid",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.EvaluateSelector(&node, tt.selector)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected evaluation to fail for selector %q, but it succeeded with result: %q", tt.selector, result)
				}
				return
			}

			if err != nil {
				t.Errorf("Failed to evaluate selector %q: %v", tt.selector, err)
				return
			}

			// For complex objects, just check that we got some content
			if tt.selector == "." {
				if !strings.Contains(result, "apiVersion") {
					t.Errorf("Expected root result to contain 'apiVersion', got: %q", result)
				}
				return
			}

			if result != tt.expected {
				t.Errorf("Selector %q: expected %q, got %q", tt.selector, tt.expected, result)
			}
		})
	}
}

func TestEvaluateArraySlice(t *testing.T) {
	t.Parallel()
	// YAML with an array for slicing tests
	yamlContent := `
items:
  - name: item0
  - name: item1
  - name: item2
  - name: item3
  - name: item4
`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &node)
	if err != nil {
		t.Fatalf("Failed to parse test YAML: %v", err)
	}

	evaluator := NewEvaluator()

	tests := []struct {
		name     string
		selector string
		expected int // Expected number of items in result
	}{
		{
			name:     "slice middle elements",
			selector: ".items[1:3]",
			expected: 2,
		},
		{
			name:     "slice from start",
			selector: ".items[:3]",
			expected: 3,
		},
		{
			name:     "slice to end",
			selector: ".items[2:]",
			expected: 3,
		},
		{
			name:     "slice all",
			selector: ".items[:]",
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := evaluator.EvaluateSelector(&node, tt.selector)
			if err != nil {
				t.Errorf("Failed to evaluate selector %q: %v", tt.selector, err)
				return
			}

			// Count the number of items in the YAML result
			lines := strings.Split(result, "\n")
			itemCount := 0
			for _, line := range lines {
				if strings.Contains(line, "- name:") {
					itemCount++
				}
			}

			if itemCount != tt.expected {
				t.Errorf("Selector %q: expected %d items, got %d in result: %q", tt.selector, tt.expected, itemCount, result)
			}
		})
	}
}
