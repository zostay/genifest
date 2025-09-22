package fileformat

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// YAMLParser implements the Parser interface for YAML files.
type YAMLParser struct{}

// Format returns FormatYAML.
func (p *YAMLParser) Format() FileFormat {
	return FormatYAML
}

// Parse converts YAML data into generic AST nodes.
func (p *YAMLParser) Parse(data []byte) ([]*Node, error) {
	var nodes []*Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	for {
		var yamlNode yaml.Node
		err := decoder.Decode(&yamlNode)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}

		genericNode, err := p.convertYAMLNode(&yamlNode)
		if err != nil {
			return nil, fmt.Errorf("failed to convert YAML node: %w", err)
		}

		nodes = append(nodes, genericNode)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no YAML documents found")
	}

	return nodes, nil
}

// Serialize converts generic AST nodes back to YAML format.
func (p *YAMLParser) Serialize(nodes []*Node) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)

	for i, node := range nodes {
		yamlNode, err := p.convertGenericNode(node)
		if err != nil {
			return nil, fmt.Errorf("failed to convert node %d: %w", i, err)
		}

		err = encoder.Encode(yamlNode)
		if err != nil {
			return nil, fmt.Errorf("failed to encode YAML node %d: %w", i, err)
		}
	}

	encoder.Close()
	return buffer.Bytes(), nil
}

// convertYAMLNode converts a yaml.Node to a generic Node.
func (p *YAMLParser) convertYAMLNode(yamlNode *yaml.Node) (*Node, error) {
	// Handle document node by converting its content
	if yamlNode.Kind == yaml.DocumentNode {
		if len(yamlNode.Content) != 1 {
			return nil, fmt.Errorf("document node should have exactly one content node, got %d", len(yamlNode.Content))
		}
		return p.convertYAMLNode(yamlNode.Content[0])
	}

	switch yamlNode.Kind {
	case yaml.DocumentNode:
		// This should never be reached due to the check above, but added for exhaustive switch
		return nil, fmt.Errorf("document node should have been handled above")

	case yaml.ScalarNode:
		return NewScalarNode(yamlNode.Value), nil

	case yaml.MappingNode:
		mapNode := NewMapNode()

		// YAML mapping nodes have alternating key-value pairs in Content
		for i := 0; i < len(yamlNode.Content); i += 2 {
			if i+1 >= len(yamlNode.Content) {
				return nil, fmt.Errorf("mapping node has odd number of content nodes")
			}

			keyNode := yamlNode.Content[i]
			valueNode := yamlNode.Content[i+1]

			if keyNode.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("mapping key must be scalar")
			}

			key := keyNode.Value
			value, err := p.convertYAMLNode(valueNode)
			if err != nil {
				return nil, fmt.Errorf("failed to convert mapping value for key %q: %w", key, err)
			}

			err = mapNode.SetMapValue(key, value)
			if err != nil {
				return nil, fmt.Errorf("failed to set map value: %w", err)
			}
		}

		return mapNode, nil

	case yaml.SequenceNode:
		arrayNode := NewArrayNode()

		for i, contentNode := range yamlNode.Content {
			value, err := p.convertYAMLNode(contentNode)
			if err != nil {
				return nil, fmt.Errorf("failed to convert sequence item %d: %w", i, err)
			}

			err = arrayNode.AppendArrayValue(value)
			if err != nil {
				return nil, fmt.Errorf("failed to append array value: %w", err)
			}
		}

		return arrayNode, nil

	case yaml.AliasNode:
		return nil, fmt.Errorf("YAML alias nodes are not supported")

	default:
		return nil, fmt.Errorf("unsupported YAML node kind: %v", yamlNode.Kind)
	}
}

// convertGenericNode converts a generic Node back to a yaml.Node.
func (p *YAMLParser) convertGenericNode(node *Node) (*yaml.Node, error) {
	switch node.Type {
	case NodeScalar:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: fmt.Sprintf("%v", node.Value),
		}, nil

	case NodeMap:
		yamlNode := &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: make([]*yaml.Node, 0, len(node.Keys)*2),
		}

		for i, key := range node.Keys {
			keyNode := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: key,
			}

			valueNode, err := p.convertGenericNode(node.Children[i])
			if err != nil {
				return nil, fmt.Errorf("failed to convert map value for key %q: %w", key, err)
			}

			yamlNode.Content = append(yamlNode.Content, keyNode, valueNode)
		}

		return yamlNode, nil

	case NodeArray:
		yamlNode := &yaml.Node{
			Kind:    yaml.SequenceNode,
			Content: make([]*yaml.Node, 0, len(node.Children)),
		}

		for i, child := range node.Children {
			childNode, err := p.convertGenericNode(child)
			if err != nil {
				return nil, fmt.Errorf("failed to convert array item %d: %w", i, err)
			}

			yamlNode.Content = append(yamlNode.Content, childNode)
		}

		return yamlNode, nil

	default:
		return nil, fmt.Errorf("unsupported generic node type: %v", node.Type)
	}
}
