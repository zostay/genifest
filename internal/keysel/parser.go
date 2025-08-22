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
	{Name: "LParen", Pattern: `\(`},
	{Name: "RParen", Pattern: `\)`},
	{Name: "Pipe", Pattern: `\|`},
	{Name: "Colon", Pattern: `:`},
	{Name: "Comma", Pattern: `,`},
	{Name: "Equals", Pattern: `==`},
	{Name: "NotEquals", Pattern: `!=`},
	{Name: "Number", Pattern: `-?\d+`},
	{Name: "String", Pattern: `"([^"\\]|\\.)*"`},
	{Name: "SingleString", Pattern: `'([^'\\]|\\.)*'`},
	{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_-]*`},
	{Name: "Whitespace", Pattern: `\s+`},
})

// Expression represents the root of a yq-style expression with pipes.
type Expression struct {
	Pipeline []*PipelineStep `parser:"@@ ( Pipe @@ )*"`
}

// PipelineStep represents one step in a pipeline.
type PipelineStep struct {
	Path     *Path     `parser:"@@"`
	Function *Function `parser:"| @@"`
}

// Path represents a path expression (the basic navigation).
type Path struct {
	Components []*Component `parser:"@@+"`
}

// Component represents a single component in a path.
type Component struct {
	Field     *Field     `parser:"@@"`
	Bracket   *Bracket   `parser:"| @@"`
	ArrayIter *ArrayIter `parser:"| @@"`
}

// Field represents field access (.fieldname).
type Field struct {
	Name string `parser:"Dot @Ident"`
}

// Bracket represents array/map access or slicing ([...]).
type Bracket struct {
	Content string `parser:"Dot? LBracket @( Number ( Colon Number? )? | Colon Number? | ( String | SingleString ) | Colon ) RBracket"`
}

// ArrayIter represents array iteration ([]).
type ArrayIter struct {
	Token string `parser:"Dot? LBracket RBracket"`
}

// Function represents a function call like select().
type Function struct {
	Name string     `parser:"@Ident"`
	Args []*FuncArg `parser:"LParen ( @@ ( Comma @@ )* )? RParen"`
}

// FuncArg represents a function argument.
type FuncArg struct {
	Comparison *Comparison `parser:"@@"`
	Path       *Path       `parser:"| @@"`
	Literal    *Literal    `parser:"| @@"`
}

// Comparison represents a comparison expression like .name == "value".
type Comparison struct {
	Left     *Path    `parser:"@@"`
	Operator string   `parser:"( @Equals | @NotEquals )"`
	Right    *Literal `parser:"@@"`
}

// Literal represents a literal value.
type Literal struct {
	String *string `parser:"( @String | @SingleString )"`
	Number *int    `parser:"| @Number"`
}

// Parser wraps the participle parser.
type Parser struct {
	parser *participle.Parser[Expression]
}

// NewParser creates a new yq-style expression parser using participle/v2.
func NewParser() (*Parser, error) {
	parser, err := participle.Build[Expression](
		participle.Lexer(selectorLexer),
		participle.Unquote("String", "SingleString"),
		participle.Elide("Whitespace"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build parser: %w", err)
	}

	return &Parser{parser: parser}, nil
}

// ParseSelector parses a yq-style expression string using participle/v2 grammar.
func (p *Parser) ParseSelector(selectorStr string) (*Expression, error) {
	if selectorStr == "" {
		return &Expression{Pipeline: []*PipelineStep{{Path: &Path{Components: []*Component{}}}}}, nil
	}

	// Handle special case of root selector "."
	if selectorStr == "." {
		return &Expression{Pipeline: []*PipelineStep{{Path: &Path{Components: []*Component{}}}}}, nil
	}

	expression, err := p.parser.ParseString("", selectorStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse selector %q: %w", selectorStr, err)
	}

	return expression, nil
}

// Evaluate evaluates the expression against a YAML node using the new pipeline AST.
func (e *Expression) Evaluate(node *yaml.Node, evaluator *Evaluator) (*yaml.Node, error) {
	// Start with the input node
	current := node

	// If we start with a document node, navigate to its content
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}

	// Process the pipeline - handle array iteration specially
	for i, step := range e.Pipeline {
		var err error
		current, err = evaluator.evaluatePipelineStepWithIteration(current, step, e.Pipeline[i+1:])
		if err != nil {
			return nil, fmt.Errorf("pipeline step %d failed: %w", i, err)
		}

		// If we processed iteration, we're done
		if current == nil {
			return nil, fmt.Errorf("no matching results from pipeline")
		}

		// Check if this step involved iteration - if so, the remaining steps were processed
		if evaluator.hasArrayIteration(step) {
			return current, nil
		}
	}

	return current, nil
}

// evaluatePipelineStep evaluates a single step in the pipeline.
func (e *Evaluator) evaluatePipelineStep(node *yaml.Node, step *PipelineStep) (*yaml.Node, error) {
	switch {
	case step.Path != nil:
		return e.evaluatePath(node, step.Path)
	case step.Function != nil:
		return e.evaluateFunction(node, step.Function)
	default:
		return nil, fmt.Errorf("unknown pipeline step type")
	}
}

// evaluatePath evaluates a path expression.
func (e *Evaluator) evaluatePath(node *yaml.Node, path *Path) (*yaml.Node, error) {
	current := node

	// Handle empty path (root access)
	if len(path.Components) == 0 {
		return current, nil
	}

	// Evaluate each component
	for _, component := range path.Components {
		var err error
		current, err = e.evaluateComponent(current, component)
		if err != nil {
			return nil, err
		}
	}

	return current, nil
}

// evaluateComponent evaluates a single component.
func (e *Evaluator) evaluateComponent(node *yaml.Node, component *Component) (*yaml.Node, error) {
	switch {
	case component.Field != nil:
		return e.evaluateField(node, component.Field)
	case component.Bracket != nil:
		return e.evaluateBracket(node, component.Bracket)
	case component.ArrayIter != nil:
		return e.evaluateArrayIter(node, component.ArrayIter)
	default:
		return nil, fmt.Errorf("unknown component type")
	}
}

// evaluateField evaluates a field access against a YAML node.
func (e *Evaluator) evaluateField(node *yaml.Node, field *Field) (*yaml.Node, error) {
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

// evaluateBracket evaluates a bracket access (index or slice) against a YAML node.
func (e *Evaluator) evaluateBracket(node *yaml.Node, bracket *Bracket) (*yaml.Node, error) {
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

// evaluateArrayIter evaluates array iteration ([]) - returns all elements as individual nodes.
func (e *Evaluator) evaluateArrayIter(node *yaml.Node, arrayIter *ArrayIter) (*yaml.Node, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("cannot iterate over non-sequence node")
	}

	// For array iteration, we need to return a special "iteration" node
	// that will be processed by the next pipeline step for each element
	// For now, let's return the array itself and handle iteration in the pipeline
	return node, nil
}

// evaluateFunction evaluates a function call like select().
func (e *Evaluator) evaluateFunction(node *yaml.Node, function *Function) (*yaml.Node, error) {
	switch function.Name {
	case "select":
		return e.evaluateSelect(node, function)
	default:
		return nil, fmt.Errorf("unknown function: %s", function.Name)
	}
}

// evaluateSelect evaluates the select() function for filtering.
func (e *Evaluator) evaluateSelect(node *yaml.Node, function *Function) (*yaml.Node, error) {
	if len(function.Args) != 1 {
		return nil, fmt.Errorf("select() function requires exactly 1 argument, got %d", len(function.Args))
	}

	arg := function.Args[0]
	if arg.Comparison == nil {
		return nil, fmt.Errorf("select() function requires a comparison argument")
	}

	// Evaluate the comparison against the current node
	matches, err := e.evaluateComparison(node, arg.Comparison)
	if err != nil {
		return nil, fmt.Errorf("select() comparison failed: %w", err)
	}

	if matches {
		return node, nil
	}

	// Return nil to indicate this node should be filtered out
	return nil, nil
}

// evaluateComparison evaluates a comparison expression.
func (e *Evaluator) evaluateComparison(node *yaml.Node, comparison *Comparison) (bool, error) {
	// Evaluate the left side (path) against the current node
	leftNode, err := e.evaluatePath(node, comparison.Left)
	if err != nil {
		return false, fmt.Errorf("left side evaluation failed: %w", err)
	}

	// Get the string value from the left side
	leftValue := ""
	if leftNode.Kind == yaml.ScalarNode {
		leftValue = leftNode.Value
	}

	// Get the right side literal value
	rightValue := ""
	if comparison.Right.String != nil {
		rightValue = *comparison.Right.String
	} else if comparison.Right.Number != nil {
		rightValue = strconv.Itoa(*comparison.Right.Number)
	}

	// Perform the comparison
	switch comparison.Operator {
	case "==":
		return leftValue == rightValue, nil
	case "!=":
		return leftValue != rightValue, nil
	default:
		return false, fmt.Errorf("unknown comparison operator: %s", comparison.Operator)
	}
}

// evaluatePipelineStepWithIteration handles pipeline steps with potential array iteration.
func (e *Evaluator) evaluatePipelineStepWithIteration(node *yaml.Node, step *PipelineStep, remainingSteps []*PipelineStep) (*yaml.Node, error) {
	// Check if this step has array iteration
	if step.Path != nil && e.pathHasArrayIteration(step.Path) {
		return e.evaluateWithArrayIteration(node, step.Path, remainingSteps)
	}

	// Regular evaluation
	return e.evaluatePipelineStep(node, step)
}

// hasArrayIteration checks if a pipeline step involves array iteration.
func (e *Evaluator) hasArrayIteration(step *PipelineStep) bool {
	return step.Path != nil && e.pathHasArrayIteration(step.Path)
}

// pathHasArrayIteration checks if a path contains array iteration.
func (e *Evaluator) pathHasArrayIteration(path *Path) bool {
	for _, component := range path.Components {
		if component.ArrayIter != nil {
			return true
		}
	}
	return false
}

// evaluateWithArrayIteration handles expressions with array iteration.
func (e *Evaluator) evaluateWithArrayIteration(node *yaml.Node, path *Path, remainingSteps []*PipelineStep) (*yaml.Node, error) {
	// Find the array iteration component
	var beforeIter []*Component
	var afterIter []*Component
	iterIndex := -1

	for i, component := range path.Components {
		if component.ArrayIter != nil {
			iterIndex = i
			beforeIter = path.Components[:i]
			if i+1 < len(path.Components) {
				afterIter = path.Components[i+1:]
			}
			break
		}
	}

	if iterIndex == -1 {
		return nil, fmt.Errorf("no array iteration found")
	}

	// Navigate to the array
	current := node
	for _, component := range beforeIter {
		var err error
		current, err = e.evaluateComponent(current, component)
		if err != nil {
			return nil, err
		}
	}

	// Ensure we have an array
	if current.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("array iteration requires a sequence node")
	}

	// Iterate over array elements
	for _, element := range current.Content {
		// Apply remaining path components to this element
		elementResult := element
		for _, component := range afterIter {
			var err error
			elementResult, err = e.evaluateComponent(elementResult, component)
			if err != nil {
				continue // Skip elements that don't match
			}
		}

		// Apply remaining pipeline steps
		for _, step := range remainingSteps {
			var err error
			elementResult, err = e.evaluatePipelineStep(elementResult, step)
			if err != nil {
				continue // Skip elements that don't match
			}
			if elementResult == nil {
				continue // Element was filtered out
			}
		}

		// If we found a match, return it
		if elementResult != nil {
			return elementResult, nil
		}
	}

	return nil, fmt.Errorf("no matching elements found")
}

// GetSimplePath extracts a simple path for write operations (backwards compatibility).
func (e *Expression) GetSimplePath() ([]*Component, error) {
	if len(e.Pipeline) != 1 {
		return nil, fmt.Errorf("write operations require simple paths, not complex pipelines")
	}

	step := e.Pipeline[0]
	if step.Path == nil {
		return nil, fmt.Errorf("write operations require path expressions, not functions")
	}

	// Check for complex features not supported in writes
	for _, component := range step.Path.Components {
		if component.ArrayIter != nil {
			return nil, fmt.Errorf("write operations do not support array iteration")
		}
	}

	return step.Path.Components, nil
}
