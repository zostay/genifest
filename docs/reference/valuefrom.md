# ValueFrom Types

Detailed reference for all ValueFrom expression types in Genifest.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Overview

ValueFrom expressions are the core mechanism for generating dynamic values in Genifest. Each type provides different capabilities for value generation.

## DefaultValue

Returns a literal string value.

```yaml
valueFrom:
  default:
    value: "literal-string"
```

**Use cases:**
- Static configuration values
- Default fallback values
- Simple string constants

## ArgumentRef

References a variable from the current evaluation context.

```yaml
valueFrom:
  argRef:
    name: "variable-name"
```

**Use cases:**
- Function parameters
- Pipeline step outputs
- Context variables

## BasicTemplate

Template string with variable substitution.

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

**Features:**
- `${variable}` syntax
- Nested ValueFrom for variables
- String interpolation

## FunctionCall

Calls a named function with arguments.

```yaml
valueFrom:
  call:
    function: "function-name"
    args:
      - name: "param1"
        valueFrom:
          default:
            value: "value"
```

**Features:**
- Reusable logic
- Parameter passing
- Function scoping

## ScriptExec

Executes external scripts.

```yaml
valueFrom:
  script:
    exec: "script.sh"
    args:
      - name: "arg1"
        valueFrom:
          default:
            value: "value"
    env:
      - name: "ENV_VAR"
        valueFrom:
          default:
            value: "value"
```

**Features:**
- External script execution
- Argument passing
- Environment variables
- Security isolation

## FileInclusion

Includes content from external files.

```yaml
valueFrom:
  file:
    app: "subdirectory"    # Optional
    source: "file.yaml"    # Required
```

**Features:**
- File content inclusion
- Subdirectory support
- Path security validation

## CallPipeline

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

**Features:**
- Multi-step processing
- Output passing between steps
- Complex value transformations

## See Also

- [Value Generation](../user-guide/value-generation.md) - Usage guide
- [Configuration Schema](schema.md) - Complete schema reference