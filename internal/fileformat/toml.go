package fileformat

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

// TOMLParser implements the Parser interface for TOML files.
type TOMLParser struct{}

// Format returns FormatTOML.
func (p *TOMLParser) Format() FileFormat {
	return FormatTOML
}

// Parse converts TOML data into generic AST nodes.
// Note: TOML doesn't support multiple documents, so this always returns a single node.
func (p *TOMLParser) Parse(data []byte) ([]*Node, error) {
	// For now, use the simpler approach with regular toml.Unmarshal
	var rawData interface{}
	err := toml.Unmarshal(data, &rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	genericNode, err := p.convertInterfaceToNode(rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert TOML data: %w", err)
	}

	return []*Node{genericNode}, nil
}

// Serialize converts generic AST nodes back to TOML format.
// Note: Only the first node is used since TOML doesn't support multiple documents.
func (p *TOMLParser) Serialize(nodes []*Node) ([]byte, error) {
	if len(nodes) == 0 {
		return []byte{}, nil
	}

	if len(nodes) > 1 {
		return nil, fmt.Errorf("TOML format does not support multiple documents")
	}

	// Convert the generic node to a Go interface{} structure that go-toml can handle
	data, err := p.convertGenericNodeToInterface(nodes[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert node to interface: %w", err)
	}

	return toml.Marshal(data)
}

// convertInterfaceToNode converts a generic interface{} to a Node.
func (p *TOMLParser) convertInterfaceToNode(data interface{}) (*Node, error) {
	if data == nil {
		return NewScalarNode(nil), nil
	}

	switch v := data.(type) {
	case map[string]interface{}:
		mapNode := NewMapNode()
		for key, value := range v {
			childNode, err := p.convertInterfaceToNode(value)
			if err != nil {
				return nil, fmt.Errorf("failed to convert value for key %q: %w", key, err)
			}
			err = mapNode.SetMapValue(key, childNode)
			if err != nil {
				return nil, fmt.Errorf("failed to set map value: %w", err)
			}
		}
		return mapNode, nil

	case []interface{}:
		arrayNode := NewArrayNode()
		for i, value := range v {
			childNode, err := p.convertInterfaceToNode(value)
			if err != nil {
				return nil, fmt.Errorf("failed to convert array item %d: %w", i, err)
			}
			err = arrayNode.AppendArrayValue(childNode)
			if err != nil {
				return nil, fmt.Errorf("failed to append array value: %w", err)
			}
		}
		return arrayNode, nil

	default:
		// All scalar types (string, int, float, bool, etc.)
		return NewScalarNode(v), nil
	}
}

// convertGenericNodeToInterface converts a generic Node to interface{} for TOML marshaling.
func (p *TOMLParser) convertGenericNodeToInterface(node *Node) (interface{}, error) {
	switch node.Type {
	case NodeScalar:
		return node.Value, nil

	case NodeMap:
		result := make(map[string]interface{})

		for i, key := range node.Keys {
			value, err := p.convertGenericNodeToInterface(node.Children[i])
			if err != nil {
				return nil, fmt.Errorf("failed to convert map value for key %q: %w", key, err)
			}
			result[key] = value
		}

		return result, nil

	case NodeArray:
		result := make([]interface{}, len(node.Children))

		for i, child := range node.Children {
			value, err := p.convertGenericNodeToInterface(child)
			if err != nil {
				return nil, fmt.Errorf("failed to convert array item %d: %w", i, err)
			}
			result[i] = value
		}

		return result, nil

	default:
		return nil, fmt.Errorf("unsupported generic node type: %v", node.Type)
	}
}
