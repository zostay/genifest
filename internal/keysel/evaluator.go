package keysel

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Evaluator provides functionality for evaluating key selectors against YAML nodes.
type Evaluator struct {
	// Can be extended with configuration options if needed
}

// NewEvaluator creates a new key selector evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// EvaluateSelector evaluates a selector string against a YAML node and returns the result as a string.
func (e *Evaluator) EvaluateSelector(node *yaml.Node, selectorStr string) (string, error) {
	parser, err := NewParser()
	if err != nil {
		return "", fmt.Errorf("failed to create parser: %w", err)
	}

	expression, err := parser.ParseSelector(selectorStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse selector %q: %w", selectorStr, err)
	}

	result, err := expression.Evaluate(node, e)
	if err != nil {
		return "", err
	}

	return e.nodeToString(result)
}

// literalToString converts a literal AST node to its string representation.
func (e *Evaluator) literalToString(literal *Literal) string {
	if literal.String != nil {
		// Remove surrounding quotes
		s := *literal.String
		if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
			return s[1 : len(s)-1]
		}
		return s
	}
	if literal.Number != nil {
		return fmt.Sprintf("%d", *literal.Number)
	}
	return ""
}

// nodeToString converts a YAML node to its string representation.
func (e *Evaluator) nodeToString(node *yaml.Node) (string, error) {
	if node.Kind == yaml.ScalarNode {
		return node.Value, nil
	}

	// For non-scalar values, return a YAML representation
	var result strings.Builder
	encoder := yaml.NewEncoder(&result)
	encoder.SetIndent(2)
	err := encoder.Encode(node)
	if err != nil {
		return "", fmt.Errorf("failed to encode result: %w", err)
	}
	err = encoder.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close encoder: %w", err)
	}
	return strings.TrimSpace(result.String()), nil
}
