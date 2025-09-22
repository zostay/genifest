package fileformat

import (
	"testing"
)

func TestTOMLDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		filename string
		expected FileFormat
	}{
		{"config.toml", FormatTOML},
		{"app.tml", FormatTOML},
		{"config.yaml", FormatYAML},
		{"config.yml", FormatYAML},
		{"config.json", FormatUnknown},
		{"README.md", FormatUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			t.Parallel()
			result := DetectFormat(tt.filename)
			if result != tt.expected {
				t.Errorf("DetectFormat(%q) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestTOMLParser(t *testing.T) {
	t.Parallel()

	tomlContent := `
title = "Test App"

[database]
host = "localhost" 
port = 5432

[[services]]
name = "web"
port = 3000
`

	parser := &TOMLParser{}

	// Test parsing
	nodes, err := parser.Parse([]byte(tomlContent))
	if err != nil {
		t.Fatalf("Failed to parse TOML: %v", err)
	}

	if len(nodes) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(nodes))
	}

	root := nodes[0]
	if root.Type != NodeMap {
		t.Fatalf("Expected root to be a map, got %v", root.Type)
	}

	// Check title
	titleNode, found := root.GetMapValue("title")
	if !found {
		t.Error("Expected title field not found")
	} else {
		title, err := titleNode.StringValue()
		if err != nil {
			t.Errorf("Failed to get title as string: %v", err)
		} else if title != "Test App" {
			t.Errorf("Expected title 'Test App', got %q", title)
		}
	}

	// Test serialization
	serialized, err := parser.Serialize(nodes)
	if err != nil {
		t.Fatalf("Failed to serialize TOML: %v", err)
	}

	if len(serialized) == 0 {
		t.Error("Serialized content is empty")
	}

	t.Logf("Serialized TOML: %s", string(serialized))
}
