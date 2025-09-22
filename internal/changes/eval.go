package changes

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/zostay/genifest/internal/config"
	"github.com/zostay/genifest/internal/fileformat"
	"github.com/zostay/genifest/internal/keysel"
)

// EvalContext provides the context for evaluating ValueFrom expressions.
// It contains the current file and document being processed, as well as
// a scratchpad of variables available for argument resolution.
type EvalContext struct {
	// CurrentFile is the path to the file being processed
	CurrentFile string

	// CurrentDocument is the YAML document being processed (can be nil)
	CurrentDocument *yaml.Node

	// Variables is a scratchpad of variables in scope for argument resolution
	Variables map[string]string

	// Functions is the list of available functions
	Functions []config.FunctionDefinition

	// CloudHome is the base directory for resolving relative paths
	CloudHome string

	// ScriptsDirs are the directories containing scripts (search path)
	ScriptsDirs []string

	// FilesDirs are the directories containing files (search path)
	FilesDirs []string
}

// NewEvalContext creates a new evaluation context with single directories (legacy).
func NewEvalContext(cloudHome, scriptsDir, filesDir string, functions []config.FunctionDefinition) *EvalContext {
	var scriptsDirs, filesDirs []string
	if scriptsDir != "" {
		scriptsDirs = []string{scriptsDir}
	}
	if filesDir != "" {
		filesDirs = []string{filesDir}
	}
	return NewEvalContextWithPaths(cloudHome, scriptsDirs, filesDirs, functions)
}

// NewEvalContextWithPaths creates a new evaluation context with multiple search paths.
func NewEvalContextWithPaths(cloudHome string, scriptsDirs, filesDirs []string, functions []config.FunctionDefinition) *EvalContext {
	return &EvalContext{
		Variables:   make(map[string]string),
		Functions:   functions,
		CloudHome:   cloudHome,
		ScriptsDirs: scriptsDirs,
		FilesDirs:   filesDirs,
	}
}

// WithFile returns a new context with the specified file set.
func (ctx *EvalContext) WithFile(filePath string) *EvalContext {
	newCtx := *ctx
	newCtx.CurrentFile = filePath
	return &newCtx
}

// WithDocument returns a new context with the specified document set.
func (ctx *EvalContext) WithDocument(doc *yaml.Node) *EvalContext {
	newCtx := *ctx
	newCtx.CurrentDocument = doc
	return &newCtx
}

// WithVariables returns a new context with additional variables.
func (ctx *EvalContext) WithVariables(vars map[string]string) *EvalContext {
	newCtx := *ctx
	newCtx.Variables = make(map[string]string)
	// Copy existing variables
	for k, v := range ctx.Variables {
		newCtx.Variables[k] = v
	}
	// Add new variables
	for k, v := range vars {
		newCtx.Variables[k] = v
	}
	return &newCtx
}

// SetVariable sets a variable in the current context.
func (ctx *EvalContext) SetVariable(name, value string) {
	ctx.Variables[name] = value
}

// GetVariable retrieves a variable from the current context.
func (ctx *EvalContext) GetVariable(name string) (string, bool) {
	value, exists := ctx.Variables[name]
	return value, exists
}

// Evaluate evaluates a ValueFrom expression and returns the resulting value.
func (ctx *EvalContext) Evaluate(valueFrom config.ValueFrom) (string, error) {
	// Check which type of ValueFrom is set and evaluate accordingly
	switch {
	case valueFrom.DefaultValue != nil:
		return ctx.evaluateDefaultValue(*valueFrom.DefaultValue)
	case valueFrom.ArgumentRef != nil:
		return ctx.evaluateArgumentRef(*valueFrom.ArgumentRef)
	case valueFrom.BasicTemplate != nil:
		return ctx.evaluateBasicTemplate(*valueFrom.BasicTemplate)
	case valueFrom.FunctionCall != nil:
		return ctx.evaluateFunctionCall(*valueFrom.FunctionCall)
	case valueFrom.ScriptExec != nil:
		return ctx.evaluateScriptExec(*valueFrom.ScriptExec)
	case valueFrom.FileInclusion != nil:
		return ctx.evaluateFileInclusion(*valueFrom.FileInclusion)
	case valueFrom.DocumentRef != nil:
		return ctx.evaluateDocumentRef(*valueFrom.DocumentRef)
	case valueFrom.CallPipeline != nil:
		return ctx.evaluateCallPipeline(*valueFrom.CallPipeline)
	case valueFrom.EnvironmentRef != nil:
		return ctx.evaluateEnvironmentRef(*valueFrom.EnvironmentRef)
	default:
		return "", fmt.Errorf("no ValueFrom type specified")
	}
}

// evaluateDefaultValue returns the literal value.
func (ctx *EvalContext) evaluateDefaultValue(dv config.DefaultValue) (string, error) {
	return dv.Value, nil
}

// evaluateArgumentRef looks up an argument from the current context.
func (ctx *EvalContext) evaluateArgumentRef(ar config.ArgumentRef) (string, error) {
	value, exists := ctx.GetVariable(ar.Name)
	if !exists {
		return "", fmt.Errorf("argument %q not found in context", ar.Name)
	}
	return value, nil
}

// evaluateBasicTemplate replaces $style variables in a template string.
func (ctx *EvalContext) evaluateBasicTemplate(bt config.BasicTemplate) (string, error) {
	// First, evaluate all the template variables
	templateVars := make(map[string]string)
	for _, arg := range bt.Variables {
		value, err := ctx.Evaluate(arg.ValueFrom)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate template variable %q: %w", arg.Name, err)
		}
		templateVars[arg.Name] = value
	}

	// Replace variables in the template string
	result := bt.String
	for name, value := range templateVars {
		// Replace ${name} and $name patterns
		result = strings.ReplaceAll(result, "${"+name+"}", value)
		result = strings.ReplaceAll(result, "$"+name, value)
	}

	// Replace $$ with $
	result = strings.ReplaceAll(result, "$$", "$")

	return result, nil
}

// evaluateFunctionCall calls a function with the provided arguments.
func (ctx *EvalContext) evaluateFunctionCall(fc config.FunctionCall) (string, error) {
	// Find the function
	var function *config.FunctionDefinition
	for _, fn := range ctx.Functions {
		if fn.Name == fc.Name {
			function = &fn
			break
		}
	}
	if function == nil {
		return "", fmt.Errorf("function %q not found", fc.Name)
	}

	// Evaluate arguments and create a new context
	functionVars := make(map[string]string)

	// Evaluate provided arguments
	for _, arg := range fc.Arguments {
		value, err := ctx.Evaluate(arg.ValueFrom)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate argument %q: %w", arg.Name, err)
		}
		functionVars[arg.Name] = value
	}

	// Check for required parameters
	for _, param := range function.Params {
		if param.Required {
			if _, exists := functionVars[param.Name]; !exists {
				return "", fmt.Errorf("required parameter %q not provided for function %q", param.Name, fc.Name)
			}
		} else if param.Default != "" {
			// Set default value if parameter not provided
			if _, exists := functionVars[param.Name]; !exists {
				functionVars[param.Name] = param.Default
			}
		}
	}

	// Create new context with function arguments, preserving document context
	functionCtx := ctx.WithVariables(functionVars)

	// Evaluate the function's ValueFrom
	return functionCtx.Evaluate(function.ValueFrom)
}

// evaluateScriptExec executes a script and returns its stdout.
func (ctx *EvalContext) evaluateScriptExec(se config.ScriptExec) (string, error) {
	// Resolve script path
	// Find script in search path
	var scriptPath string
	var found bool
	for _, dir := range ctx.ScriptsDirs {
		candidatePath := filepath.Join(dir, se.ExecCommand)
		if _, err := os.Stat(candidatePath); err == nil {
			scriptPath = candidatePath
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("script %q not found in any of the script directories: %v", se.ExecCommand, ctx.ScriptsDirs)
	}

	// Prepare command
	cmd := exec.Command(scriptPath)

	// Set working directory to CloudHome
	cmd.Dir = ctx.CloudHome

	// Evaluate and set environment variables
	if len(se.Env) > 0 {
		env := os.Environ()
		for _, envArg := range se.Env {
			value, err := ctx.Evaluate(envArg.ValueFrom)
			if err != nil {
				return "", fmt.Errorf("failed to evaluate environment variable %q: %w", envArg.Name, err)
			}
			env = append(env, fmt.Sprintf("%s=%s", envArg.Name, value))
		}
		cmd.Env = env
	}

	// Evaluate and set arguments
	if len(se.Args) > 0 {
		args := make([]string, len(se.Args))
		for i, arg := range se.Args {
			value, err := ctx.Evaluate(arg.ValueFrom)
			if err != nil {
				return "", fmt.Errorf("failed to evaluate script argument %d: %w", i, err)
			}
			args[i] = value
		}
		cmd.Args = append(cmd.Args, args...)
	}

	// Set stdin if provided
	if se.Stdin != nil {
		stdinValue, err := ctx.Evaluate(*se.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate stdin: %w", err)
		}
		cmd.Stdin = strings.NewReader(stdinValue)
	}

	// Execute and capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Include both stdout and stderr in error message for debugging
		var errorMsg strings.Builder
		errorMsg.WriteString(fmt.Sprintf("script execution failed: %v", err))

		if stderr.Len() > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nstderr: %s", strings.TrimSpace(stderr.String())))
		}

		if stdout.Len() > 0 {
			errorMsg.WriteString(fmt.Sprintf("\nstdout: %s", strings.TrimSpace(stdout.String())))
		}

		return "", fmt.Errorf("%s", errorMsg.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// evaluateFileInclusion reads a file and returns its contents.
func (ctx *EvalContext) evaluateFileInclusion(fi config.FileInclusion) (string, error) {
	// Determine the app directory
	appDir := ""
	if fi.App != "" {
		appDir = fi.App
	} else if ctx.CurrentFile != "" {
		// Extract app from current file path if possible
		parts := strings.Split(ctx.CurrentFile, string(filepath.Separator))
		if len(parts) >= 2 {
			appDir = parts[len(parts)-2]
		}
	}

	// Search for file in all file directories
	var filePath string
	var found bool
	for _, dir := range ctx.FilesDirs {
		var candidatePath string
		if appDir != "" {
			candidatePath = filepath.Join(dir, appDir, fi.Source)
		} else {
			candidatePath = filepath.Join(dir, fi.Source)
		}
		if _, err := os.Stat(candidatePath); err == nil {
			filePath = candidatePath
			found = true
			break
		}
	}

	if !found {
		return "", fmt.Errorf("file %q not found in any of the file directories: %v", fi.Source, ctx.FilesDirs)
	}

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Apply transient changes if any are defined
	if len(fi.Changes) > 0 {
		modifiedContent, err := ctx.applyTransientChanges(content, fi.Changes, fi.Source)
		if err != nil {
			return "", fmt.Errorf("failed to apply transient changes to file %s: %w", filePath, err)
		}
		content = modifiedContent
	}

	return string(content), nil
}

// applyTransientChanges applies a list of transient changes to file content in memory.
func (ctx *EvalContext) applyTransientChanges(content []byte, changes []config.TransientChange, filename string) ([]byte, error) {
	// Detect or use specified file format
	format := fileformat.FormatUnknown
	if len(changes) > 0 && changes[0].Format != "" {
		format = fileformat.ParseFormat(changes[0].Format)
	}
	if format == fileformat.FormatUnknown {
		format = fileformat.DetectFormat(filename)
	}
	if format == fileformat.FormatUnknown {
		return nil, fmt.Errorf("cannot determine file format for %s", filename)
	}

	// Get parser for the format
	parser, err := fileformat.GetParser(format)
	if err != nil {
		return nil, fmt.Errorf("failed to get parser for format %s: %w", format, err)
	}

	// Parse the content
	documents, err := parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file content: %w", err)
	}

	// Apply each transient change
	for _, change := range changes {
		for docIndex, doc := range documents {
			// Check if document selector matches this document (if specified)
			if len(change.DocumentSelector) > 0 {
				matches := ctx.matchesDocumentSelector(doc, change.DocumentSelector)
				if !matches {
					continue
				}
			}

			// Create a new context with the target file and document for evaluation
			// This ensures documentRef operations work on the file being modified
			changeCtx := ctx.WithFile(filename)

			// Convert generic document to YAML node for document context
			yamlParser := &fileformat.YAMLParser{}
			yamlContent, err := yamlParser.Serialize([]*fileformat.Node{doc})
			if err != nil {
				return nil, fmt.Errorf("failed to convert document to YAML for context: %w", err)
			}

			var yamlDoc yaml.Node
			err = yaml.Unmarshal(yamlContent, &yamlDoc)
			if err != nil {
				return nil, fmt.Errorf("failed to parse document for context: %w", err)
			}

			changeCtx = changeCtx.WithDocument(&yamlDoc)

			// Evaluate the new value using the updated context
			newValue, err := changeCtx.Evaluate(change.ValueFrom)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate value for transient change %d: %w", docIndex, err)
			}

			// Apply the change to the document
			//nolint:staticcheck // SA4006 false positive - changed is used on line 429
			changed, err := ctx.setValueInGenericDocument(doc, change.KeySelector, newValue)
			if err != nil {
				return nil, fmt.Errorf("failed to set value in document %d: %w", docIndex, err)
			}

			if !changed {
				// Value was already the same, no change needed
				continue
			}
		}
	}

	// Serialize the modified documents back
	return parser.Serialize(documents)
}

// matchesDocumentSelector checks if a document matches the given selector using the generic AST.
func (ctx *EvalContext) matchesDocumentSelector(doc *fileformat.Node, selector config.DocumentSelector) bool {
	if doc.Type != fileformat.NodeMap {
		return false
	}

	// Check each selector key-value pair
	for key, expectedValue := range selector {
		switch key {
		case "kind":
			kindNode, found := doc.GetMapValue("kind")
			if !found {
				return false
			}
			kind, err := kindNode.StringValue()
			if err != nil || kind != expectedValue {
				return false
			}
		case "metadata.name":
			metadataNode, found := doc.GetMapValue("metadata")
			if !found {
				return false
			}
			if metadataNode.Type != fileformat.NodeMap {
				return false
			}
			nameNode, found := metadataNode.GetMapValue("name")
			if !found {
				return false
			}
			name, err := nameNode.StringValue()
			if err != nil || name != expectedValue {
				return false
			}
		default:
			// Handle other keys like metadata.namespace, spec.template.spec.name, etc.
			// Use the key selector parsing logic to navigate to the field
			actualValue, err := ctx.getValueFromGenericDocument(doc, key)
			if err != nil {
				return false // Field not found or error accessing
			}
			if actualValue != expectedValue {
				return false
			}
		}
	}

	return true
}

// setValueInGenericDocument sets a value in a generic document using keySelector.
func (ctx *EvalContext) setValueInGenericDocument(doc *fileformat.Node, keySelector, value string) (bool, error) {
	// For now, convert to YAML, apply change, and convert back
	// TODO: Implement native generic document modification
	yamlParser := &fileformat.YAMLParser{}
	yamlContent, err := yamlParser.Serialize([]*fileformat.Node{doc})
	if err != nil {
		return false, fmt.Errorf("failed to convert to YAML: %w", err)
	}

	// Parse as YAML node
	var yamlDoc yaml.Node
	err = yaml.Unmarshal(yamlContent, &yamlDoc)
	if err != nil {
		return false, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Apply the change using keysel evaluation
	changed, err := ctx.applyYAMLChange(&yamlDoc, keySelector, value)
	if err != nil {
		return false, err
	}

	if changed {
		// Convert back to generic format
		modifiedYAML, err := yaml.Marshal(&yamlDoc)
		if err != nil {
			return false, fmt.Errorf("failed to marshal modified YAML: %w", err)
		}

		modifiedNodes, err := yamlParser.Parse(modifiedYAML)
		if err != nil {
			return false, fmt.Errorf("failed to parse modified YAML: %w", err)
		}

		if len(modifiedNodes) > 0 {
			// Copy the modified structure back
			*doc = *modifiedNodes[0]
		}
	}

	return changed, nil
}

// getValueFromGenericDocument gets a value from a generic document using keySelector.
func (ctx *EvalContext) getValueFromGenericDocument(doc *fileformat.Node, keySelector string) (string, error) {
	// Convert to YAML for easier key selector navigation
	yamlParser := &fileformat.YAMLParser{}
	yamlContent, err := yamlParser.Serialize([]*fileformat.Node{doc})
	if err != nil {
		return "", fmt.Errorf("failed to convert to YAML: %w", err)
	}

	// Parse as YAML node
	var yamlDoc yaml.Node
	err = yaml.Unmarshal(yamlContent, &yamlDoc)
	if err != nil {
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Get the value using keysel evaluation
	return ctx.getYAMLValue(&yamlDoc, keySelector)
}

// getYAMLValue gets a value from a YAML document using keysel.
func (ctx *EvalContext) getYAMLValue(doc *yaml.Node, keySelector string) (string, error) {
	// Parse the keySelector
	parser, err := keysel.NewParser()
	if err != nil {
		return "", fmt.Errorf("failed to create keysel parser: %w", err)
	}

	expr, err := parser.ParseSelector(keySelector)
	if err != nil {
		return "", fmt.Errorf("failed to parse keySelector %q: %w", keySelector, err)
	}

	// Try to get a simple path for direct navigation
	if components, err := expr.GetSimplePath(); err == nil {
		return ctx.getValueAtSimplePath(doc, components)
	}

	// For complex expressions, not yet supported
	return "", fmt.Errorf("complex keySelector expressions not yet supported in transient changes")
}

// getValueAtSimplePath gets a value at a simple path in a YAML document.
func (ctx *EvalContext) getValueAtSimplePath(doc *yaml.Node, components []*keysel.Component) (string, error) {
	if len(components) == 0 {
		return "", fmt.Errorf("empty path")
	}

	// Navigate to the target node
	current := doc
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}

	for _, component := range components {
		next, err := ctx.navigateYAMLComponent(current, component)
		if err != nil {
			return "", fmt.Errorf("failed to navigate component: %w", err)
		}
		current = next
	}

	// Return the value as string
	if current.Kind == yaml.ScalarNode {
		return current.Value, nil
	}

	return "", fmt.Errorf("target field is not a scalar value")
}

// applyYAMLChange applies a change to a YAML document using keysel.
func (ctx *EvalContext) applyYAMLChange(doc *yaml.Node, keySelector, value string) (bool, error) {
	// Parse the keySelector
	parser, err := keysel.NewParser()
	if err != nil {
		return false, fmt.Errorf("failed to create keysel parser: %w", err)
	}

	expr, err := parser.ParseSelector(keySelector)
	if err != nil {
		return false, fmt.Errorf("failed to parse selector %q: %w", keySelector, err)
	}

	// Try to get a simple path for direct modification
	if components, err := expr.GetSimplePath(); err == nil {
		return ctx.setValueAtSimplePath(doc, components, value)
	}

	// For complex expressions, we need to evaluate and then set
	// This is a simplified version - full implementation would handle all cases
	return false, fmt.Errorf("complex keySelector expressions not yet supported in transient changes")
}

// setValueAtSimplePath sets a value at a simple path in a YAML document.
func (ctx *EvalContext) setValueAtSimplePath(doc *yaml.Node, components []*keysel.Component, value string) (bool, error) {
	if len(components) == 0 {
		return false, fmt.Errorf("empty path")
	}

	// Navigate to the parent of the target
	current := doc
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}
	for _, component := range components[:len(components)-1] {
		next, err := ctx.navigateYAMLComponent(current, component)
		if err != nil {
			return false, err
		}
		current = next
	}

	// Set the value at the final component
	lastComponent := components[len(components)-1]
	return ctx.setYAMLValueAtComponent(current, lastComponent, value)
}

// navigateYAMLComponent navigates to a single component in a YAML document.
func (ctx *EvalContext) navigateYAMLComponent(node *yaml.Node, component *keysel.Component) (*yaml.Node, error) {
	if component.Field != nil {
		return ctx.findYAMLField(node, component.Field.Name)
	}

	if component.Bracket != nil {
		return ctx.navigateYAMLBracket(node, component.Bracket)
	}

	return nil, fmt.Errorf("unsupported component type")
}

// findYAMLField finds a field in a YAML mapping node.
func (ctx *EvalContext) findYAMLField(node *yaml.Node, fieldName string) (*yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("cannot access field %q on non-mapping node", fieldName)
	}

	// YAML mapping nodes have alternating key-value pairs
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 >= len(node.Content) {
			break
		}
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Kind == yaml.ScalarNode && keyNode.Value == fieldName {
			return valueNode, nil
		}
	}

	return nil, fmt.Errorf("field %q not found", fieldName)
}

// navigateYAMLBracket navigates a bracket expression in a YAML node.
func (ctx *EvalContext) navigateYAMLBracket(node *yaml.Node, bracket *keysel.Bracket) (*yaml.Node, error) {
	content := bracket.Content

	// Handle quoted strings as map keys
	if bracket.IsQuotedString() {
		if node.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("cannot use quoted key access on non-mapping node")
		}
		// Remove quotes
		key := content[1 : len(content)-1]
		return ctx.findYAMLField(node, key)
	}

	// Handle numeric indices as array access
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("cannot use numeric index on non-sequence node")
	}

	// Parse index
	var index int
	if _, err := fmt.Sscanf(content, "%d", &index); err != nil {
		return nil, fmt.Errorf("invalid array index %q", content)
	}

	// Handle negative indices
	if index < 0 {
		index = len(node.Content) + index
	}

	if index < 0 || index >= len(node.Content) {
		return nil, fmt.Errorf("array index %d out of bounds", index)
	}

	return node.Content[index], nil
}

// setYAMLValueAtComponent sets a value at a specific component in a YAML node.
func (ctx *EvalContext) setYAMLValueAtComponent(parent *yaml.Node, component *keysel.Component, value string) (bool, error) {
	if component.Field != nil {
		return ctx.setYAMLFieldValue(parent, component.Field.Name, value)
	}

	if component.Bracket != nil {
		return ctx.setYAMLBracketValue(parent, component.Bracket, value)
	}

	return false, fmt.Errorf("unsupported component type for value setting")
}

// setYAMLFieldValue sets a field value in a YAML mapping node.
func (ctx *EvalContext) setYAMLFieldValue(parent *yaml.Node, fieldName, value string) (bool, error) {
	if parent.Kind != yaml.MappingNode {
		return false, fmt.Errorf("cannot set field on non-mapping node")
	}

	// Find existing field
	for i := 0; i < len(parent.Content); i += 2 {
		if i+1 >= len(parent.Content) {
			break
		}
		keyNode := parent.Content[i]
		valueNode := parent.Content[i+1]

		if keyNode.Kind == yaml.ScalarNode && keyNode.Value == fieldName {
			// Check if value is already the same
			if valueNode.Kind == yaml.ScalarNode && valueNode.Value == value {
				return false, nil // No change needed
			}

			// Update the value
			valueNode.Kind = yaml.ScalarNode
			valueNode.Value = value
			return true, nil
		}
	}

	// Field doesn't exist, create it
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: fieldName}
	valueNode := &yaml.Node{Kind: yaml.ScalarNode, Value: value}
	parent.Content = append(parent.Content, keyNode, valueNode)
	return true, nil
}

// setYAMLBracketValue sets a bracket value in a YAML node.
func (ctx *EvalContext) setYAMLBracketValue(parent *yaml.Node, bracket *keysel.Bracket, value string) (bool, error) {
	content := bracket.Content

	// Handle quoted strings as map keys
	if bracket.IsQuotedString() {
		if parent.Kind != yaml.MappingNode {
			return false, fmt.Errorf("cannot use quoted key access on non-mapping node")
		}
		// Remove quotes
		key := content[1 : len(content)-1]
		return ctx.setYAMLFieldValue(parent, key, value)
	}

	// Handle numeric indices as array access
	if parent.Kind != yaml.SequenceNode {
		return false, fmt.Errorf("cannot use numeric index on non-sequence node")
	}

	// Parse index
	var index int
	if _, err := fmt.Sscanf(content, "%d", &index); err != nil {
		return false, fmt.Errorf("invalid array index %q", content)
	}

	// Handle negative indices
	if index < 0 {
		index = len(parent.Content) + index
	}

	if index < 0 || index >= len(parent.Content) {
		return false, fmt.Errorf("array index %d out of bounds", index)
	}

	// Check if value is already the same
	targetNode := parent.Content[index]
	if targetNode.Kind == yaml.ScalarNode && targetNode.Value == value {
		return false, nil // No change needed
	}

	// Update the value
	targetNode.Kind = yaml.ScalarNode
	targetNode.Value = value
	return true, nil
}

// evaluateDocumentRef extracts a value from the current document using yq-style selector.
func (ctx *EvalContext) evaluateDocumentRef(dr config.DocumentRef) (string, error) {
	if ctx.CurrentDocument == nil {
		return "", fmt.Errorf("no current document available for document reference")
	}

	// Handle file selector - if specified, we would need to load the target file
	// For now, we'll work with the current document
	if dr.FileSelector != "" {
		return "", fmt.Errorf("file selector in document reference not yet supported")
	}

	// Use the new keysel package to evaluate the key selector
	evaluator := keysel.NewEvaluator()
	value, err := evaluator.EvaluateSelector(ctx.CurrentDocument, dr.KeySelector)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate key selector %q: %w", dr.KeySelector, err)
	}

	return value, nil
}

// evaluateCallPipeline executes a pipeline of function calls.
func (ctx *EvalContext) evaluateCallPipeline(cp config.CallPipeline) (string, error) {
	if len(cp) == 0 {
		return "", fmt.Errorf("empty pipeline")
	}

	currentCtx := ctx
	var result string

	for i, pipe := range cp {
		value, err := currentCtx.Evaluate(pipe.ValueFrom)
		if err != nil {
			return "", fmt.Errorf("pipeline step %d failed: %w", i, err)
		}
		result = value

		// If this pipe has an output name, set it as a variable for the next step
		if pipe.Output != "" {
			currentCtx = currentCtx.WithVariables(map[string]string{
				pipe.Output: result,
			})
		}
	}

	return result, nil
}

// evaluateEnvironmentRef reads a value from an environment variable.
func (ctx *EvalContext) evaluateEnvironmentRef(er config.EnvironmentRef) (string, error) {
	value := os.Getenv(er.Name)

	// If the environment variable is not set or empty, use the default if provided
	if value == "" {
		if er.Default != "" {
			return er.Default, nil
		}
		return "", fmt.Errorf("environment variable %q is not set and no default value provided", er.Name)
	}

	return value, nil
}
