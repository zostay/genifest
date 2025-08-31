package fileformat

import (
	"path/filepath"
	"strings"
)

// DetectFormat detects the file format based on the file extension.
func DetectFormat(filename string) FileFormat {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".yaml", ".yml":
		return FormatYAML
	case ".toml", ".tml":
		return FormatTOML
	default:
		return FormatUnknown
	}
}

// Parser is the interface that file format parsers must implement.
type Parser interface {
	// Parse parses the given data into a slice of generic AST nodes.
	// Multiple nodes represent multiple documents (like YAML's --- separator).
	Parse(data []byte) ([]*Node, error)

	// Serialize converts a slice of generic AST nodes back to the format's byte representation.
	Serialize(nodes []*Node) ([]byte, error)

	// Format returns the format this parser handles.
	Format() FileFormat
}

// GetParser returns a parser for the given format.
func GetParser(format FileFormat) (Parser, error) {
	switch format {
	case FormatYAML:
		return &YAMLParser{}, nil
	case FormatTOML:
		return &TOMLParser{}, nil
	case FormatUnknown:
		return nil, &UnsupportedFormatError{Format: format}
	default:
		return nil, &UnsupportedFormatError{Format: format}
	}
}

// UnsupportedFormatError represents an error for unsupported file formats.
type UnsupportedFormatError struct {
	Format FileFormat
}

func (e *UnsupportedFormatError) Error() string {
	return "unsupported file format: " + e.Format.String()
}
