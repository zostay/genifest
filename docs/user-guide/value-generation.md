# Value Generation

Deep dive into Genifest's ValueFrom expression system for dynamic value generation.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Overview

ValueFrom expressions are the core of Genifest's dynamic value generation system. They provide multiple ways to generate values for configuration changes.

## ValueFrom Types

### DefaultValue
```yaml
valueFrom:
  default:
    value: "literal-string"
```

### ArgumentRef
```yaml
valueFrom:
  argRef:
    name: "variable-name"
```

### BasicTemplate
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

### FunctionCall
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

### ScriptExec
```yaml
valueFrom:
  script:
    exec: "script.sh"
    args:
      - name: "arg1"
        valueFrom:
          default:
            value: "value"
```

### FileInclusion
```yaml
valueFrom:
  file:
    app: "optional-subdirectory"
    source: "filename.yaml"
```

### CallPipeline
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
```

## See Also

- [Core Concepts](concepts.md) - Understanding the system
- [Configuration Files](configuration.md) - Configuration reference