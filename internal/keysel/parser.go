package keysel

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/alecthomas/participle/v2"       //nolint:depguard // Required for grammar-based selector parsing
	"github.com/alecthomas/participle/v2/lexer" //nolint:depguard // Required for grammar-based selector parsing
	"gopkg.in/yaml.v3"
)

// Define lexer rules for yq-style selectors.
var selectorLexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "Dot", Pattern: `\.`},
	{Name: "LBracket", Pattern: `\[`},
	{Name: "RBracket", Pattern: `\]`},
	{Name: "Colon", Pattern: `:`},
	{Name: "Number", Pattern: `-?\d+`},
	{Name: "String", Pattern: `"([^"\\]|\\.)*"`},
	{Name: "SingleString", Pattern: `'([^'\\]|\\.)*'`},
	{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_-]*`},
	{Name: "Whitespace", Pattern: `\s+`},
})

// Selector represents the root of a yq-style selector.
type Selector struct {
	Components []*Component `parser:"@@*"`
}

// Component represents a single component in a selector path.
type Component struct {
	Field   *Field   `parser:"@@"`
	Bracket *Bracket `parser:"| @@"`
}

// Field represents field access (.fieldname).
type Field struct {
	Name string `parser:"Dot @Ident"`
}

// Bracket represents array/map access or slicing ([...]).
type Bracket struct {
	Content string `parser:"Dot? LBracket @( Number ( Colon Number? )? | Colon Number? | ( String | SingleString ) | Colon ) RBracket"`
}

// Parser wraps the participle parser.
type Parser struct {
	parser *participle.Parser[Selector]
}

// NewParser creates a new yq-style selector parser using participle/v2.
func NewParser() (*Parser, error) {
	parser, err := participle.Build[Selector](
		participle.Lexer(selectorLexer),
		participle.Unquote("String", "SingleString"),
		participle.Elide("Whitespace"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build parser: %w", err)
	}

	return &Parser{parser: parser}, nil
}

// ParseSelector parses a yq-style selector string using participle/v2 grammar.
func (p *Parser) ParseSelector(selectorStr string) (*Selector, error) {
	if selectorStr == "" {
		return &Selector{Components: []*Component{}}, nil
	}

	// Handle special case of root selector "."
	if selectorStr == "." {
		return &Selector{Components: []*Component{}}, nil
	}

	selector, err := p.parser.ParseString("", selectorStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse selector %q: %w", selectorStr, err)
	}

	return selector, nil
}

// Evaluate evaluates the selector against a YAML node using the participle AST.
func (s *Selector) Evaluate(node *yaml.Node, evaluator *Evaluator) (*yaml.Node, error) {
	current := node

	// If we start with a document node, navigate to its content
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}

	// Handle empty selector (root access)
	if len(s.Components) == 0 {
		return current, nil
	}

	// Evaluate each component
	for _, component := range s.Components {
		var err error
		current, err = evaluator.evaluateParticleComponent(current, component)
		if err != nil {
			return nil, err
		}
	}

	return current, nil
}

// evaluateParticleComponent evaluates a participle Component against a YAML node.
func (e *Evaluator) evaluateParticleComponent(node *yaml.Node, component *Component) (*yaml.Node, error) {
	switch {
	case component.Field != nil:
		return e.evaluateParticleField(node, component.Field)
	case component.Bracket != nil:
		return e.evaluateParticleBracket(node, component.Bracket)
	default:
		return nil, fmt.Errorf("unknown component type in AST")
	}
}

// evaluateParticleField evaluates a participle Field access against a YAML node.
func (e *Evaluator) evaluateParticleField(node *yaml.Node, field *Field) (*yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("cannot access field %q from non-mapping node", field.Name)
	}

	// Search for the field in the mapping
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 < len(node.Content) && node.Content[i].Value == field.Name {
			return node.Content[i+1], nil
		}
	}

	return nil, fmt.Errorf("field %q not found", field.Name)
}

// evaluateParticleBracket evaluates a participle Bracket access (index or slice) against a YAML node.
func (e *Evaluator) evaluateParticleBracket(node *yaml.Node, bracket *Bracket) (*yaml.Node, error) {
	content := bracket.Content

	// Check if it contains a colon (slice operation)
	if strings.Contains(content, ":") {
		return e.evaluateBracketSlice(node, content)
	}

	// Handle index operations
	return e.evaluateBracketIndex(node, content)
}

// evaluateBracketSlice handles slice operations like [1:3], [1:], [:3], [:].
func (e *Evaluator) evaluateBracketSlice(node *yaml.Node, content string) (*yaml.Node, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("cannot slice non-sequence node")
	}

	length := len(node.Content)
	start := 0
	end := length

	parts := strings.Split(content, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid slice format: %q", content)
	}

	// Parse start value
	if parts[0] != "" {
		startVal, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid slice start: %q", parts[0])
		}
		start = startVal
	}

	// Parse end value
	if parts[1] != "" {
		endVal, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid slice end: %q", parts[1])
		}
		end = endVal
	}

	// Handle negative indices
	if start < 0 {
		start = length + start
	}
	if end < 0 {
		end = length + end
	}

	// Clamp to valid bounds
	if start < 0 {
		start = 0
	}
	if start > length {
		start = length
	}
	if end < 0 {
		end = 0
	}
	if end > length {
		end = length
	}

	// Ensure start <= end
	if start > end {
		start = end
	}

	// Create a new sequence node with the sliced content
	result := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Tag:     node.Tag,
		Content: node.Content[start:end],
	}

	return result, nil
}

// evaluateBracketIndex handles index operations like [0], [-1], ["key"].
func (e *Evaluator) evaluateBracketIndex(node *yaml.Node, content string) (*yaml.Node, error) {
	// Try numeric index first
	if idx, err := strconv.Atoi(content); err == nil {
		if node.Kind == yaml.SequenceNode {
			if idx < 0 {
				// Negative indexing from the end
				idx = len(node.Content) + idx
			}
			if idx < 0 || idx >= len(node.Content) {
				return nil, fmt.Errorf("array index %d out of bounds (length %d)", idx, len(node.Content))
			}
			return node.Content[idx], nil
		}
		return nil, fmt.Errorf("cannot index non-sequence node with numeric index %d", idx)
	}

	// Handle string key indexing (remove quotes if present)
	key := content
	if (strings.HasPrefix(key, "\"") && strings.HasSuffix(key, "\"")) ||
		(strings.HasPrefix(key, "'") && strings.HasSuffix(key, "'")) {
		key = key[1 : len(key)-1]
	}

	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 < len(node.Content) && node.Content[i].Value == key {
				return node.Content[i+1], nil
			}
		}
		return nil, fmt.Errorf("key %q not found", key)
	}
	return nil, fmt.Errorf("cannot index non-mapping node with string key %q", key)
}
