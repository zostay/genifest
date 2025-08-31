package fileformat

import (
	"fmt"
	"strconv"
)

// FileFormat represents the supported file formats.
type FileFormat int

const (
	// FormatUnknown indicates an unknown or unsupported format.
	FormatUnknown FileFormat = iota
	// FormatYAML indicates YAML format.
	FormatYAML
	// FormatTOML indicates TOML format.
	FormatTOML
)

// String returns the string representation of the file format.
func (f FileFormat) String() string {
	switch f {
	case FormatYAML:
		return "yaml"
	case FormatTOML:
		return "toml"
	case FormatUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// ParseFormat parses a string into a FileFormat.
func ParseFormat(s string) FileFormat {
	switch s {
	case "yaml", "yml":
		return FormatYAML
	case "toml", "tml":
		return FormatTOML
	default:
		return FormatUnknown
	}
}

// NodeType represents the type of a node in the generic AST.
type NodeType int

const (
	// NodeMap represents a map/object node.
	NodeMap NodeType = iota
	// NodeArray represents an array/list node.
	NodeArray
	// NodeScalar represents a scalar/primitive value node.
	NodeScalar
)

// Node represents a generic AST node that can represent both YAML and TOML structures.
type Node struct {
	Type     NodeType
	Value    interface{} // The actual value for scalar nodes
	Children []*Node     // Children for map/array nodes
	Keys     []string    // Keys for map nodes (corresponds to Children indices)
}

// NewScalarNode creates a new scalar node with the given value.
func NewScalarNode(value interface{}) *Node {
	return &Node{
		Type:  NodeScalar,
		Value: value,
	}
}

// NewMapNode creates a new map node.
func NewMapNode() *Node {
	return &Node{
		Type:     NodeMap,
		Children: make([]*Node, 0),
		Keys:     make([]string, 0),
	}
}

// NewArrayNode creates a new array node.
func NewArrayNode() *Node {
	return &Node{
		Type:     NodeArray,
		Children: make([]*Node, 0),
	}
}

// SetMapValue sets a key-value pair in a map node.
func (n *Node) SetMapValue(key string, value *Node) error {
	if n.Type != NodeMap {
		return fmt.Errorf("cannot set map value on non-map node")
	}

	// Check if key already exists
	for i, existingKey := range n.Keys {
		if existingKey == key {
			n.Children[i] = value
			return nil
		}
	}

	// Add new key-value pair
	n.Keys = append(n.Keys, key)
	n.Children = append(n.Children, value)
	return nil
}

// GetMapValue retrieves a value by key from a map node.
func (n *Node) GetMapValue(key string) (*Node, bool) {
	if n.Type != NodeMap {
		return nil, false
	}

	for i, existingKey := range n.Keys {
		if existingKey == key {
			return n.Children[i], true
		}
	}

	return nil, false
}

// AppendArrayValue appends a value to an array node.
func (n *Node) AppendArrayValue(value *Node) error {
	if n.Type != NodeArray {
		return fmt.Errorf("cannot append to non-array node")
	}

	n.Children = append(n.Children, value)
	return nil
}

// GetArrayValue retrieves a value by index from an array node.
func (n *Node) GetArrayValue(index int) (*Node, bool) {
	if n.Type != NodeArray {
		return nil, false
	}

	if index < 0 || index >= len(n.Children) {
		return nil, false
	}

	return n.Children[index], true
}

// SetArrayValue sets a value at a specific index in an array node.
func (n *Node) SetArrayValue(index int, value *Node) error {
	if n.Type != NodeArray {
		return fmt.Errorf("cannot set array value on non-array node")
	}

	if index < 0 || index >= len(n.Children) {
		return fmt.Errorf("array index %d out of bounds (length: %d)", index, len(n.Children))
	}

	n.Children[index] = value
	return nil
}

// String returns a string representation of the node value.
func (n *Node) String() string {
	if n.Type != NodeScalar {
		return ""
	}

	if n.Value == nil {
		return ""
	}

	return fmt.Sprintf("%v", n.Value)
}

// StringValue returns the string value of a scalar node.
func (n *Node) StringValue() (string, error) {
	if n.Type != NodeScalar {
		return "", fmt.Errorf("cannot get string value from non-scalar node")
	}

	if n.Value == nil {
		return "", nil
	}

	if str, ok := n.Value.(string); ok {
		return str, nil
	}

	return fmt.Sprintf("%v", n.Value), nil
}

// IntValue returns the integer value of a scalar node.
func (n *Node) IntValue() (int64, error) {
	if n.Type != NodeScalar {
		return 0, fmt.Errorf("cannot get int value from non-scalar node")
	}

	if n.Value == nil {
		return 0, nil
	}

	switch v := n.Value.(type) {
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case int32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}

// BoolValue returns the boolean value of a scalar node.
func (n *Node) BoolValue() (bool, error) {
	if n.Type != NodeScalar {
		return false, fmt.Errorf("cannot get bool value from non-scalar node")
	}

	if n.Value == nil {
		return false, nil
	}

	switch v := n.Value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("cannot convert %T to bool", v)
	}
}

// Clone creates a deep copy of the node.
func (n *Node) Clone() *Node {
	clone := &Node{
		Type:  n.Type,
		Value: n.Value,
	}

	if n.Keys != nil {
		clone.Keys = make([]string, len(n.Keys))
		copy(clone.Keys, n.Keys)
	}

	if n.Children != nil {
		clone.Children = make([]*Node, len(n.Children))
		for i, child := range n.Children {
			clone.Children[i] = child.Clone()
		}
	}

	return clone
}
