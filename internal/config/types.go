package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the data structure that is built by looking through all the
// directories under the root folder named "genifest.yaml". The configuration
// of these files is merged as follows:
//
// Metadata path lists are merged.
//
// Files and Changes are merged across all configurations.
//
// Functions are merged across all configurations. It is not permitted for a
// function to have the same name as another function in the same configuration
// file.
type Config struct {
	// Metadata is metadata about genifest configuration.
	Metadata MetaConfig `yaml:"metadata"`

	// Files is the list of files genifest manages and has access to. Paths are
	// local to the current directory.
	Files []string `yaml:"files"`

	// Changes are a list of change orders that can be applied to modify the
	// managed files.
	Changes []ChangeOrder `yaml:"changes"`

	// Functions defines the functions that are usable by changes. A function
	// is only usable by changes defined in the same path or changes defined
	// at an inner path.
	Functions []FunctionDefinition `yaml:"functions"`
}

// PathContext represents a path with context about where it was defined.
// It combines a relative path with information about the configuration file
// where the path was originally specified.
type PathContext struct {
	// contextPath is the directory path where this configuration was found
	contextPath string

	// Path is the relative path as defined in the configuration file
	Path string
}

// ContextPath returns the directory path where this configuration was found.
func (pc PathContext) ContextPath() string {
	return pc.contextPath
}

// SetContextPath sets the directory path where this configuration was found.
func (pc *PathContext) SetContextPath(path string) {
	pc.contextPath = path
}

// MarshalYAML implements yaml.Marshaler for PathContext.
func (pc PathContext) MarshalYAML() (interface{}, error) {
	return pc.Path, nil
}

// UnmarshalYAML implements yaml.Unmarshaler for PathContext.
func (pc *PathContext) UnmarshalYAML(value *yaml.Node) error {
	var path string
	if err := value.Decode(&path); err != nil {
		return err
	}
	pc.Path = path
	// contextPath will be filled in by the loader
	return nil
}

// PathContexts is a slice of PathContext that implements custom YAML marshalling.
// It appears as a simple string array in YAML files but maintains context information.
type PathContexts []PathContext

// MarshalYAML implements yaml.Marshaler for PathContexts.
func (pcs PathContexts) MarshalYAML() (interface{}, error) {
	paths := make([]string, len(pcs))
	for i, pc := range pcs {
		paths[i] = pc.Path
	}
	return paths, nil
}

// UnmarshalYAML implements yaml.Unmarshaler for PathContexts.
func (pcs *PathContexts) UnmarshalYAML(value *yaml.Node) error {
	var paths []string
	if err := value.Decode(&paths); err != nil {
		return err
	}

	*pcs = make(PathContexts, len(paths))
	for i, path := range paths {
		(*pcs)[i] = PathContext{Path: path}
		// contextPath will be filled in by the loader
	}
	return nil
}

// MetaConfig contains metadata about the genifest configuration including
// directory paths for scripts, manifests, and files, as well as the cloudHome boundary.
type MetaConfig struct {
	// CloudHome is the path to the root of the configuration. No genifest.yaml
	// configuration or work will be done outside of this folder. This is always
	// set by the loader based on the working directory of the genifest command.
	CloudHome string `yaml:"cloudHome"`

	// Scripts is a list of folders that may contain scripts. These are relative
	// to the folder holding the configuration file.
	Scripts PathContexts `yaml:"scripts"`

	// Manifests is a list of folders that may contain manifests. These folders
	// are structured with sub-folders identifying applications. This is usual
	// only defined in the top-level genifest.yaml, but in any case, the path is
	// relative to the directory containing the configuration file.
	Manifests PathContexts `yaml:"manifests"`

	// Files is a list of folders holding other related files, which are used with
	// the FileInclusion ValueFrom values. These are usually defined in the top-level
	// genifest.yaml file, but are defined relative to the configuration file
	// holding it. Similar to Manifests, these folders are arranged using sub-folders
	// to identify the app that the files belong to.
	Files PathContexts `yaml:"files"`
}

// ChangeOrder represents a modification to be applied to managed files.
// It specifies which file and key to change, along with the new value.
type ChangeOrder struct {
	// path is the path where this change order was discovered before config
	// merger was performed.
	path string

	// DocumentRef defines the file, document, and key to change.
	DocumentRef `yaml:",inline"`

	// Tag is used to select which change orders to run.
	Tag string `yaml:"tag"`

	// ValueFrom is the value to apply to the DocumentRef
	ValueFrom ValueFrom `yaml:"valueFrom"`
}

// FunctionDefinition defines a reusable function that can be called from change orders.
// Functions have parameters and return values computed from ValueFrom expressions.
type FunctionDefinition struct {
	// path is the path within which this function definition is available
	path string

	// Name is the name of the function.
	Name string `yaml:"name"`

	// Params defines the function parameters.
	Params []Parameter `yaml:"params"`

	// ValueFrom defines the value returned by the function.
	ValueFrom ValueFrom `yaml:"valueFrom"`
}

// Parameter defines a function parameter with a name, optional default value,
// and whether it's required.
type Parameter struct {
	// Name is the name of the parameter.
	Name string `yaml:"name"`

	// Requires is true when the parameter is required.
	Required bool `yaml:"required"`

	// Default is the value to provide when no value is provided for this
	// parameter. Default may not be given when Required.
	Default string `yaml:"default"`
}

// DocumentSelector is a map from YAML keys to values the document must have.
// This is intended to be a very simple selection criteria for identifying
// specific YAML documents within a file.
type DocumentSelector map[string]string

// ValueFrom defines a value from one of the definitions within. Only one
// kind of value is permitted per ValueFrom. This is a union type that supports
// various ways of computing or retrieving values.
type ValueFrom struct {
	// FunctionCall calls the named function with the named arguments. The value
	// is the result of the function call.
	FunctionCall *FunctionCall `yaml:"call"`

	// CallPipeline runs a function call. The output from each pipe in the
	// pipeline is fed as an input to the next pipeline.
	CallPipeline *CallPipeline `yaml:"pipeline"`

	// FileInclusion loads the contents of a  file from the files directory.
	FileInclusion *FileInclusion `yaml:"file"`

	// BasicTemplate outputs a string after replacing $style variables with
	// specified values.
	BasicTemplate *BasicTemplate `yaml:"template"`

	// ScriptExec executes the given script from the scripts directory and uses
	// the standard output as the value.
	ScriptExec *ScriptExec `yaml:"script"`

	// ArgumentRef looks up the argument from the current context. This is only
	// available within a function definition or pipeline.
	ArgumentRef *ArgumentRef `yaml:"argRef"`

	// DefaultValue uses the given literal value.
	DefaultValue *DefaultValue `yaml:"default"`

	// DocumentRef looks up a key in the YAML document that is being changed.
	DocumentRef *DocumentRef `yaml:"documentRef"`
}

// FunctionCall looks up a function in the functions list and executes the
// ValueFrom for that function with the provided arguments.
type FunctionCall struct {
	// Name of the function to execute.
	Name string `yaml:"function"`

	// Arguments to pass to the function.
	Arguments Arguments `yaml:"args"`
}

// Argument defines an argument to pass to a ValueFrom expression.
// It consists of a name and a value computed from another ValueFrom.
type Argument struct {
	// Name of the argument to set.
	Name string `yaml:"name"`

	// ValueFrom resolves to the value to pass.
	ValueFrom ValueFrom `yaml:"valueFrom"`
}

// Arguments is a list of Argument values for function calls and templates.
type Arguments []Argument

// CallPipeline defines a list of functions or scripts to call. The output of
// the first feeds into an argument to the second. The second feeds into the
// third and so-on until the final output, which is the value to use.
type CallPipeline []CallPipe

// CallPipe is an individual element of a CallPipeline that defines
// a single step in a processing pipeline.
type CallPipe struct {
	// ValueFrom is the value to pull in for the pipeline. The first CallPipe
	// in a pipeline may be any type of value. However, subsequent pipelines
	// must either be a FunctionCall or ScriptExec.
	ValueFrom ValueFrom `yaml:"valueFrom"`

	// Output is the name to give the output. This is available as an argument
	// in the next pipeline.
	Output string `yaml:"output"`
}

// FileInclusion looks up a file in the files directory. The content of the file
// becomes the value. Files are organized by application subdirectories.
type FileInclusion struct {
	// App is the application sub-directory to use. If not specified, it will
	// ue the same app folder as the change.
	App string `yaml:"app"`

	// Source is the name of the file to read.
	Source string `yaml:"source"`
}

// BasicTemplate turns a string with $style variables into a string value.
// Variables are replaced with values from the provided arguments.
type BasicTemplate struct {
	// String is the template with $style variables that must match the names
	// of arguments. If $style is ambiguous, you may use ${style}. If you need
	// a $, then $$ escapes to a single $.
	String string `yaml:"string"`

	// Variables is the list of variables available in the template.
	Variables Arguments `yaml:"variables"`
}

// ScriptExec executes a program, usually a script, from the scripts folder.
// It supports passing arguments, environment variables, and stdin data.
type ScriptExec struct {
	// ExecCommand is the name of the script to execute. The path is relative
	// to the scripts folder.
	ExecCommand string `yaml:"exec"`

	// Stdin is the value ot use to pass as stdin to script. If this is not set,
	// nothing will be sent to stdin.
	Stdin *ValueFrom `yaml:"stdin"`

	// Args is the list of arguments to pass to the script.
	Args Arguments `yaml:"args"`

	// Env is the list of environment variables to set for the script.
	Env Arguments `yaml:"env"`
}

// ArgumentRef is permitted inside of a CallPipeline to refer to the output
// variable of a previous CallPipe or within a function definition to refer
// to a parameter. It is an error to use this in other contexts.
type ArgumentRef struct {
	// Name is the name of the parameter to use.
	Name string `yaml:"name"`
}

// DefaultValue is a literal value that provides a static string result.
type DefaultValue struct {
	// Value is the literal value to set.
	Value string `yaml:"value"`
}

// DocumentRef looks up a key in a document. The FileSelector and
// DocumentSelector are optional. The KeySelector is required and uses yq syntax.
type DocumentRef struct {
	// FileSelector may be omitted. In a ChangeOrder, omitting this value means
	// that the DocumentSelector and KeySelector will be applied to all files in
	// the current folder, so the change may be applied multiple times. In the
	// case of a ValueFrom field, this indicates that the current file will be
	// used.
	FileSelector string `yaml:"fileSelector"`

	// DocumentSelector may be omitted. In a ChangeOrder, omitting this value
	// means that the KeySelector will be applied to as many documents in the
	// files identified by the FileSelector as it matches, so it may apply to
	// multiple files. In a ValueFrom field, this indicates that the current
	// document will be used.
	DocumentSelector DocumentSelector `yaml:"documentSelector"`

	// KeySelector identifies the specific field to select. This is in the form
	// of a yq expression that will identify a specific field.
	KeySelector string `yaml:"keySelector"`
}

// Validation patterns.
var (
	identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$|^[a-z]$`)
	kebabPattern      = regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]$|^[a-z]$`)
)

// isValidIdentifier checks if a string is a valid kebab-case identifier for names.
// Valid identifiers start with a lowercase letter, contain only lowercase letters,
// numbers, and hyphens, and end with a letter or number.
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	return identifierPattern.MatchString(s)
}

// isValidKebabTag checks if a string is a valid kebab-case tag (looser than identifier).
// Tags follow the same rules as identifiers but are optional (empty string is valid).
func isValidKebabTag(s string) bool {
	if s == "" {
		return true // tags are optional
	}
	return kebabPattern.MatchString(s)
}

// ValidationContext provides context for validation including available functions
// and the current path for function scope resolution.
type ValidationContext struct {
	Functions   []FunctionDefinition
	CurrentPath string
}

// LookupFunction finds the best available function for the given name from the current path.
// It returns the function definition and true if found, or nil and false if not found.
// Functions are available if they are defined in the same path or in a parent path.
// When multiple functions with the same name exist, the one from the deepest (closest) path is chosen.
func (ctx *ValidationContext) LookupFunction(name string) (*FunctionDefinition, bool) {
	var bestMatch *FunctionDefinition
	var bestDepth = -1

	for i := range ctx.Functions {
		fn := &ctx.Functions[i]
		if fn.Name == name {
			// Check if this function is available from the current path
			if ctx.isFunctionAvailable(fn.path) {
				// Calculate depth (shallower paths have lower depth)
				depth := strings.Count(fn.path, string(filepath.Separator))

				// Choose the function from the closest (deepest valid) path
				if bestMatch == nil || depth > bestDepth {
					bestMatch = fn
					bestDepth = depth
				}
			}
		}
	}

	return bestMatch, bestMatch != nil
}

// isFunctionAvailable checks if a function defined at functionPath is available from currentPath.
// A function is available if it's defined in the same path or in a parent path.
// This implements the scoping rules where functions can only be called from the same
// directory or subdirectories where they are defined.
func (ctx *ValidationContext) isFunctionAvailable(functionPath string) bool {
	// Normalize paths
	currentPath := filepath.Clean(ctx.CurrentPath)
	funcPath := filepath.Clean(functionPath)

	// Function is available if it's in the same path or a parent path
	if currentPath == funcPath {
		return true
	}

	// Check if functionPath is a parent of currentPath
	rel, err := filepath.Rel(funcPath, currentPath)
	if err != nil {
		return false
	}

	// If the relative path doesn't start with "..", then funcPath is a parent
	return !strings.HasPrefix(rel, "..")
}

// Validate methods

// Validate validates the entire configuration including metadata, changes, and functions.
// It sets up a validation context with function definitions and validates all components.
func (c *Config) Validate() error {
	ctx := &ValidationContext{
		Functions: c.Functions,
	}

	if err := c.Metadata.ValidateWithContext(ctx); err != nil {
		return fmt.Errorf("metadata validation failed: %w", err)
	}

	for i, change := range c.Changes {
		ctx.CurrentPath = change.path
		if err := change.ValidateWithContext(ctx); err != nil {
			return fmt.Errorf("change %d validation failed: %w", i, err)
		}
	}

	for i, fn := range c.Functions {
		ctx.CurrentPath = fn.path
		if err := fn.ValidateWithContext(ctx); err != nil {
			return fmt.Errorf("function %d validation failed: %w", i, err)
		}
	}

	return nil
}

// Validate validates the metadata configuration without context.
func (m *MetaConfig) Validate() error {
	return m.ValidateWithContext(nil)
}

// ValidateWithContext validates the metadata configuration, ensuring all paths
// are within the cloudHome boundary to prevent path traversal attacks.
func (m *MetaConfig) ValidateWithContext(_ *ValidationContext) error {
	// Validate that all paths are within cloudHome
	if m.CloudHome != "" {
		for _, scriptCtx := range m.Scripts {
			if err := m.validatePathWithinHome(scriptCtx.Path, "script"); err != nil {
				return err
			}
		}

		for _, manifestCtx := range m.Manifests {
			if err := m.validatePathWithinHome(manifestCtx.Path, "manifest"); err != nil {
				return err
			}
		}

		for _, fileCtx := range m.Files {
			if err := m.validatePathWithinHome(fileCtx.Path, "file"); err != nil {
				return err
			}
		}
	}

	return nil
}

// validatePathWithinHome checks if a relative path would resolve to a location within cloudHome.
// This security validation prevents path traversal attacks by ensuring paths don't escape
// the cloudHome boundary using ".." directory references.
func (m *MetaConfig) validatePathWithinHome(relativePath, pathType string) error {
	if relativePath == "" {
		return nil // empty paths are allowed
	}

	// Clean the relative path
	cleanPath := filepath.Clean(relativePath)

	// Check for absolute paths (not allowed)
	if filepath.IsAbs(cleanPath) {
		return fmt.Errorf("%s path '%s' must be relative, not absolute", pathType, relativePath)
	}

	// Check for parent directory references that would escape cloudHome
	if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "/..") || strings.Contains(cleanPath, "\\..") {
		return fmt.Errorf("%s path '%s' attempts to reference parent directories outside of cloudHome", pathType, relativePath)
	}

	return nil
}

// Validate validates a change order without context.
func (c *ChangeOrder) Validate() error {
	return c.ValidateWithContext(nil)
}

// ValidateWithContext validates a change order including its document reference,
// tag format, and valueFrom expression using the provided validation context.
func (c *ChangeOrder) ValidateWithContext(ctx *ValidationContext) error {
	if err := c.DocumentRef.ValidateWithContext(ctx); err != nil {
		return fmt.Errorf("document ref validation failed: %w", err)
	}

	if !isValidKebabTag(c.Tag) {
		return fmt.Errorf("tag '%s' is not a valid kebab-case tag", c.Tag)
	}

	if err := c.ValueFrom.ValidateWithContext(ctx); err != nil {
		return fmt.Errorf("valueFrom validation failed: %w", err)
	}

	return nil
}

// Validate validates a function definition without context.
func (f *FunctionDefinition) Validate() error {
	return f.ValidateWithContext(nil)
}

// ValidateWithContext validates a function definition including its name,
// parameters, and valueFrom expression using the provided validation context.
func (f *FunctionDefinition) ValidateWithContext(ctx *ValidationContext) error {
	if !isValidIdentifier(f.Name) {
		return fmt.Errorf("function name '%s' is not a valid identifier", f.Name)
	}

	for i, param := range f.Params {
		if err := param.ValidateWithContext(ctx); err != nil {
			return fmt.Errorf("parameter %d validation failed: %w", i, err)
		}
	}

	if err := f.ValueFrom.ValidateWithContext(ctx); err != nil {
		return fmt.Errorf("valueFrom validation failed: %w", err)
	}

	return nil
}

// Validate validates a parameter without context.
func (p *Parameter) Validate() error {
	return p.ValidateWithContext(nil)
}

// ValidateWithContext validates a parameter ensuring the name is a valid identifier
// and that required parameters don't have default values.
func (p *Parameter) ValidateWithContext(_ *ValidationContext) error {
	if !isValidIdentifier(p.Name) {
		return fmt.Errorf("parameter name '%s' is not a valid identifier", p.Name)
	}
	if p.Required && p.Default != "" {
		return fmt.Errorf("parameter %s is required and cannot have a default", p.Name)
	}
	return nil
}

// Validate validates a ValueFrom expression without context.
func (v *ValueFrom) Validate() error {
	return v.ValidateWithContext(nil)
}

// ValidateWithContext validates a ValueFrom expression ensuring exactly one field is set
// and that the chosen field is valid according to its own validation rules.
func (v *ValueFrom) ValidateWithContext(ctx *ValidationContext) error {
	count := 0

	if v.FunctionCall != nil {
		count++
		if err := v.FunctionCall.ValidateWithContext(ctx); err != nil {
			return fmt.Errorf("function call validation failed: %w", err)
		}
	}
	if v.CallPipeline != nil {
		count++
		if err := v.CallPipeline.ValidateWithContext(ctx); err != nil {
			return fmt.Errorf("call pipeline validation failed: %w", err)
		}
	}
	if v.FileInclusion != nil {
		count++
		if err := v.FileInclusion.ValidateWithContext(ctx); err != nil {
			return fmt.Errorf("file inclusion validation failed: %w", err)
		}
	}
	if v.BasicTemplate != nil {
		count++
		if err := v.BasicTemplate.ValidateWithContext(ctx); err != nil {
			return fmt.Errorf("basic template validation failed: %w", err)
		}
	}
	if v.ScriptExec != nil {
		count++
		if err := v.ScriptExec.ValidateWithContext(ctx); err != nil {
			return fmt.Errorf("script exec validation failed: %w", err)
		}
	}
	if v.ArgumentRef != nil {
		count++
		if err := v.ArgumentRef.ValidateWithContext(ctx); err != nil {
			return fmt.Errorf("argument ref validation failed: %w", err)
		}
	}
	if v.DefaultValue != nil {
		count++
		if err := v.DefaultValue.ValidateWithContext(ctx); err != nil {
			return fmt.Errorf("default value validation failed: %w", err)
		}
	}
	if v.DocumentRef != nil {
		count++
		if err := v.DocumentRef.ValidateWithContext(ctx); err != nil {
			return fmt.Errorf("document ref validation failed: %w", err)
		}
	}

	if count != 1 {
		return fmt.Errorf("exactly one field must be set in ValueFrom, but %d fields are set", count)
	}

	return nil
}

// Validate validates a function call without context.
func (f *FunctionCall) Validate() error {
	return f.ValidateWithContext(nil)
}

// ValidateWithContext validates a function call including checking that the
// referenced function exists and is accessible from the current path.
func (f *FunctionCall) ValidateWithContext(ctx *ValidationContext) error {
	if !isValidIdentifier(f.Name) {
		return fmt.Errorf("function name '%s' is not a valid identifier", f.Name)
	}

	// Check if the function exists and is available
	if ctx != nil {
		if _, found := ctx.LookupFunction(f.Name); !found {
			return fmt.Errorf("function '%s' is not defined or not accessible from current path", f.Name)
		}
	}

	if err := f.Arguments.ValidateWithContext(ctx); err != nil {
		return fmt.Errorf("arguments validation failed: %w", err)
	}

	return nil
}

// Validate validates an argument without context.
func (a *Argument) Validate() error {
	return a.ValidateWithContext(nil)
}

// ValidateWithContext validates an argument ensuring the name is a valid identifier
// and the valueFrom expression is valid.
func (a *Argument) ValidateWithContext(ctx *ValidationContext) error {
	if !isValidIdentifier(a.Name) {
		return fmt.Errorf("argument name '%s' is not a valid identifier", a.Name)
	}

	if err := a.ValueFrom.ValidateWithContext(ctx); err != nil {
		return fmt.Errorf("valueFrom validation failed: %w", err)
	}

	return nil
}

// Validate validates a list of arguments without context.
func (a Arguments) Validate() error {
	return a.ValidateWithContext(nil)
}

// ValidateWithContext validates all arguments in the list using the provided context.
func (a Arguments) ValidateWithContext(ctx *ValidationContext) error {
	for i, arg := range a {
		if err := arg.ValidateWithContext(ctx); err != nil {
			return fmt.Errorf("argument %d validation failed: %w", i, err)
		}
	}
	return nil
}

// Validate validates a call pipeline without context.
func (c CallPipeline) Validate() error {
	return c.ValidateWithContext(nil)
}

// ValidateWithContext validates a call pipeline ensuring it's not empty and that
// subsequent pipes after the first are limited to FunctionCall or ScriptExec.
func (c CallPipeline) ValidateWithContext(ctx *ValidationContext) error {
	if len(c) == 0 {
		return fmt.Errorf("call pipeline cannot be empty")
	}

	for i, pipe := range c {
		if err := pipe.ValidateWithContext(ctx); err != nil {
			return fmt.Errorf("pipe %d validation failed: %w", i, err)
		}

		// Subsequent pipes must be FunctionCall or ScriptExec
		if i > 0 {
			if err := pipe.validateSubsequentPipe(); err != nil {
				return fmt.Errorf("pipe %d validation failed: %w", i, err)
			}
		}
	}
	return nil
}

// Validate validates a file inclusion without context.
func (f *FileInclusion) Validate() error {
	return f.ValidateWithContext(nil)
}

// ValidateWithContext validates a file inclusion ensuring the source field is provided.
func (f *FileInclusion) ValidateWithContext(_ *ValidationContext) error {
	// App is optional - if not specified, uses same app folder as the change
	if f.Source == "" {
		return fmt.Errorf("source field is required")
	}
	return nil
}

// Validate validates a basic template without context.
func (b *BasicTemplate) Validate() error {
	return b.ValidateWithContext(nil)
}

// ValidateWithContext validates a basic template ensuring the string field is provided
// and all variables are valid.
func (b *BasicTemplate) ValidateWithContext(ctx *ValidationContext) error {
	if b.String == "" {
		return fmt.Errorf("string field is required")
	}

	if err := b.Variables.ValidateWithContext(ctx); err != nil {
		return fmt.Errorf("variables validation failed: %w", err)
	}

	return nil
}

// Validate validates a script execution without context.
func (s *ScriptExec) Validate() error {
	return s.ValidateWithContext(nil)
}

// ValidateWithContext validates a script execution ensuring the exec field is provided
// and all arguments, environment variables, and stdin are valid.
func (s *ScriptExec) ValidateWithContext(ctx *ValidationContext) error {
	if s.ExecCommand == "" {
		return fmt.Errorf("exec field is required")
	}

	if s.Stdin != nil {
		if err := s.Stdin.ValidateWithContext(ctx); err != nil {
			return fmt.Errorf("stdin validation failed: %w", err)
		}
	}

	if err := s.Args.ValidateWithContext(ctx); err != nil {
		return fmt.Errorf("args validation failed: %w", err)
	}

	if err := s.Env.ValidateWithContext(ctx); err != nil {
		return fmt.Errorf("env validation failed: %w", err)
	}

	return nil
}

// Validate validates an argument reference without context.
func (a *ArgumentRef) Validate() error {
	return a.ValidateWithContext(nil)
}

// ValidateWithContext validates an argument reference ensuring the name is a valid identifier.
func (a *ArgumentRef) ValidateWithContext(_ *ValidationContext) error {
	if !isValidIdentifier(a.Name) {
		return fmt.Errorf("argument ref name '%s' is not a valid identifier", a.Name)
	}
	return nil
}

// Validate validates a default value without context.
func (d *DefaultValue) Validate() error {
	return d.ValidateWithContext(nil)
}

// ValidateWithContext validates a default value ensuring the value field is provided.
func (d *DefaultValue) ValidateWithContext(_ *ValidationContext) error {
	if d.Value == "" {
		return fmt.Errorf("value field is required")
	}
	return nil
}

// Validate validates a document reference without context.
func (d *DocumentRef) Validate() error {
	return d.ValidateWithContext(nil)
}

// ValidateWithContext validates a document reference ensuring the keySelector is provided.
// FileSelector and DocumentSelector are optional per the documentation.
func (d *DocumentRef) ValidateWithContext(_ *ValidationContext) error {
	if d.KeySelector == "" {
		return fmt.Errorf("keySelector is required")
	}
	// FileSelector and DocumentSelector are optional per documentation
	return nil
}

// Validate validates a call pipe without context.
func (c *CallPipe) Validate() error {
	return c.ValidateWithContext(nil)
}

// ValidateWithContext validates a call pipe ensuring the valueFrom expression
// and output name are valid.
func (c *CallPipe) ValidateWithContext(ctx *ValidationContext) error {
	if err := c.ValueFrom.ValidateWithContext(ctx); err != nil {
		return fmt.Errorf("valueFrom validation failed: %w", err)
	}

	if !isValidIdentifier(c.Output) {
		return fmt.Errorf("output name '%s' is not a valid identifier", c.Output)
	}

	return nil
}

// validateSubsequentPipe checks that subsequent pipes in a pipeline are FunctionCall or ScriptExec.
// This enforces the constraint that only the first pipe can use any ValueFrom type.
func (c *CallPipe) validateSubsequentPipe() error {
	if c.ValueFrom.FunctionCall == nil && c.ValueFrom.ScriptExec == nil {
		return fmt.Errorf("subsequent pipes must be either FunctionCall or ScriptExec")
	}
	return nil
}
