package changes

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/zostay/genifest/internal/config"
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

	// ScriptsDir is the directory containing scripts
	ScriptsDir string

	// FilesDir is the directory containing files
	FilesDir string
}

// NewEvalContext creates a new evaluation context.
func NewEvalContext(cloudHome, scriptsDir, filesDir string, functions []config.FunctionDefinition) *EvalContext {
	return &EvalContext{
		Variables:  make(map[string]string),
		Functions:  functions,
		CloudHome:  cloudHome,
		ScriptsDir: scriptsDir,
		FilesDir:   filesDir,
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

	// Create new context with function arguments
	functionCtx := ctx.WithVariables(functionVars)

	// Evaluate the function's ValueFrom
	return functionCtx.Evaluate(function.ValueFrom)
}

// evaluateScriptExec executes a script and returns its stdout.
func (ctx *EvalContext) evaluateScriptExec(se config.ScriptExec) (string, error) {
	// Resolve script path
	scriptPath := filepath.Join(ctx.ScriptsDir, se.ExecCommand)

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("script %q not found at %s", se.ExecCommand, scriptPath)
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

	// Execute and capture output
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("script execution failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
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

	// Build file path
	var filePath string
	if appDir != "" {
		filePath = filepath.Join(ctx.FilesDir, appDir, fi.Source)
	} else {
		filePath = filepath.Join(ctx.FilesDir, fi.Source)
	}

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return string(content), nil
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
