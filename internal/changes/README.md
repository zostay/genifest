# Changes Package

The `changes` package provides the evaluation system for `ValueFrom` configurations in genifest. It allows dynamic value generation through various sources like functions, templates, scripts, files, and more.

## Core Components

### EvalContext

The `EvalContext` provides the runtime environment for evaluating `ValueFrom` expressions. It contains:

- **CurrentFile**: Path to the file being processed
- **CurrentDocument**: YAML document being processed (for document references)
- **Variables**: Scratchpad of variables available for argument resolution
- **Functions**: Available function definitions
- **CloudHome**, **ScriptsDir**, **FilesDir**: Base directories for resolving paths

### ValueFrom Evaluators

The system supports evaluation of all `ValueFrom` types:

#### DefaultValue
Returns literal string values.
```yaml
valueFrom:
  default:
    value: "literal-string"
```

#### ArgumentRef
References variables from the current evaluation context.
```yaml
valueFrom:
  argRef:
    name: "variable-name"
```

#### BasicTemplate
Template strings with `$variable` substitution.
```yaml
valueFrom:
  template:
    string: "Hello ${name}!"
    variables:
      - name: "name"
        valueFrom:
          default:
            value: "World"
```

#### FunctionCall
Calls named functions with arguments.
```yaml
valueFrom:
  call:
    function: "get-replicas"
    args:
      - name: "environment"
        valueFrom:
          default:
            value: "production"
```

#### ScriptExec
Executes scripts from the scripts directory.
```yaml
valueFrom:
  script:
    exec: "build-image.sh"
    args:
      - name: "tag"
        valueFrom:
          default:
            value: "latest"
    env:
      - name: "BUILD_ENV"
        valueFrom:
          default:
            value: "production"
```

#### FileInclusion
Includes content from files in the files directory.
```yaml
valueFrom:
  file:
    app: "myapp"  # optional subdirectory
    source: "config.yaml"
```

#### CallPipeline
Chains multiple operations together.
```yaml
valueFrom:
  pipeline:
    - valueFrom:
        default:
          value: "initial"
      output: "step1"
    - valueFrom:
        template:
          string: "${step1}-processed"
          variables:
            - name: "step1"
              valueFrom:
                argRef:
                  name: "step1"
```

#### DocumentRef
References values from the current YAML document (not yet fully implemented).

## Applier

The `Applier` provides a higher-level interface for applying changes to configurations:

```go
applier := NewApplier(config)

// Evaluate a single change
value, err := applier.EvaluateChangeValue(change, "path/to/file.yaml")

// Apply all matching changes for specific tags
results, err := applier.ApplyChanges("path/to/file.yaml", []string{"production"})
```

## Security and Safety

- All script execution uses the configured CloudHome as working directory
- File inclusion is restricted to the configured files directory
- Path traversal attacks are prevented through path validation
- Environment variables are properly isolated for script execution

## Testing

Comprehensive tests cover:
- Individual ValueFrom evaluator functionality
- Error handling and edge cases
- Integration with real configuration files (guestbook example)
- Context immutability and variable scoping
- Script execution with different argument and environment configurations

## Example Usage

```go
// Create evaluation context
ctx := NewEvalContext(
    "/path/to/project",
    "/path/to/scripts", 
    "/path/to/files",
    functions,
)

// Set variables
ctx.SetVariable("environment", "production")

// Evaluate a ValueFrom expression
valueFrom := config.ValueFrom{
    FunctionCall: &config.FunctionCall{
        Name: "get-replicas",
        Arguments: []config.Argument{
            {
                Name: "env",
                ValueFrom: config.ValueFrom{
                    ArgumentRef: &config.ArgumentRef{Name: "environment"},
                },
            },
        },
    },
}

result, err := ctx.Evaluate(valueFrom)
```