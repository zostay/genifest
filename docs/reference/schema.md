# Configuration Schema

Complete schema reference for Genifest configuration files.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Root Configuration Schema

```yaml
# genifest.yaml
metadata:
  cloudHome: string              # Optional: security boundary
  scripts: [string]              # Optional: script directories
  manifests: [string]            # Optional: manifest directories  
  files: [string]                # Optional: file directories

functions:                       # Optional: function definitions
  - name: string                 # Required: function name
    params:                      # Optional: parameters
      - name: string             # Required: parameter name
        required: boolean        # Optional: default false
    valueFrom: ValueFrom         # Required: value generation

files: [string]                  # Optional: managed files

changes:                         # Optional: change definitions
  - tag: string                  # Optional: filter tag
    fileSelector: string         # Required: file pattern
    keySelector: string          # Required: YAML path
    valueFrom: ValueFrom         # Required: value generation
```

## KeySelector Syntax

The `keySelector` field uses **yq-style path expressions** to specify which parts of YAML documents to modify. This syntax is a subset of the expression language used by tools like `yq` and `jq`.

### Basic Syntax

| Syntax | Description | Example |
|--------|-------------|---------|
| `.field` | Access object field | `.metadata.name` |
| `.field.nested` | Access nested field | `.spec.template.spec` |
| `[index]` | Array element by index | `[0]`, `[1]`, `[-1]` |
| `[]` | Array iteration | `.containers[]` |
| `[start:end]` | Array slice | `[1:3]`, `[2:]`, `[:3]` |
| `["key"]` | Quoted key access | `["app.yaml"]` |
| `['key']` | Single-quoted key | `['key-name']` |
| `\|` | Pipeline operator | `\| select(.name == "app")` |
| `//` | Alternative operator | `// "default-value"` |
| `select()` | Filter function | `select(.name == "frontend")` |
| `==`, `!=` | Comparison operators | `.name == "app"` |

### Complete Examples

```yaml
# Field access
keySelector: ".metadata.name"
keySelector: ".spec.replicas" 
keySelector: ".spec.template.spec.containers"

# Array indexing  
keySelector: ".spec.containers[0]"           # First container
keySelector: ".items[-1]"                    # Last item
keySelector: ".spec.ports[1]"                # Second port

# Map key access
keySelector: ".data.config"                  # Simple key
keySelector: ".data.[\"app.yaml\"]"          # Key with dots
keySelector: ".labels.[\"app.kubernetes.io/name\"]"  # Complex key

# Array slicing
keySelector: ".items[1:3]"                   # Elements 1 and 2
keySelector: ".items[2:]"                    # From index 2 to end
keySelector: ".items[:3]"                    # First 3 elements
keySelector: ".items[:]"                     # All elements

# Array iteration and pipeline operations
keySelector: ".spec.containers[]"                                              # Iterate over containers
keySelector: ".spec.containers[] | select(.name == \"frontend\")"              # Filter containers
keySelector: ".spec.containers[] | select(.name == \"frontend\") | .image"     # Pipeline with field access

# Alternative values for fallbacks
keySelector: ".metadata.annotations[\"missing\"] // \"default-value\""          # Fallback if annotation missing
keySelector: ".spec.replicas // \"3\""                                         # Default replica count
keySelector: ".data.config // \"fallback-config\""                             # Default configuration

# Complex nested access
keySelector: ".spec.template.spec.containers[0].image"
keySelector: ".spec.volumes[0].configMap.items[1].key"
keySelector: ".metadata.annotations.[\"deployment.kubernetes.io/revision\"]"
```

### Features

- **Grammar-based parsing**: Robust expression parsing using formal grammar
- **Array iteration**: Process all elements in arrays with `[]` syntax
- **Pipeline operations**: Chain operations with `|` operator
- **Filtering functions**: Built-in `select()` function for conditional filtering
- **Comparison operators**: Support for `==` and `!=` in filter conditions
- **Negative indexing**: Use negative numbers to access from array end
- **Quoted keys**: Handle keys with special characters like dots, dashes, slashes
- **Mixed access**: Combine field access, array indexing, and key access
- **Path validation**: Expressions are validated at parse time

### Supported vs yq/jq

This implementation focuses on **path navigation** and supports a subset of yq/jq syntax:

✅ **Supported Operations:**
- Object field access (`.field`, `.nested.field`)
- Array indexing with positive/negative indices (`[0]`, `[-1]`)  
- Array iteration (`[]`) for processing all elements
- Array slicing (`[start:end]`, `[start:]`, `[:end]`)
- Quoted key access (`["key"]`, `['key']`) for special characters
- Pipeline operations (`|`) for chaining expressions
- Alternative operator (`//`) for fallback values when paths don't exist
- Filtering with `select()` function
- Comparison operators (`==`, `!=`) for equality tests
- Complex nested paths combining all above features

❌ **Not Supported:**
- Advanced filtering functions (`map()`, `has()`, `contains()`, etc.)
- Arithmetic and logical operations (`+`, `-`, `*`, `/`, `and`, `or`)
- String manipulation functions (`split()`, `join()`, `length()`)
- Conditional expressions (`if-then-else`)
- Recursive descent operator (`..`)
- Advanced comparison operators (`<`, `>`, `<=`, `>=`)
- Variable assignment and complex queries

!!! info "Complete KeySelector Reference"
    For comprehensive documentation of keySelector syntax, grammar details, and examples, see the [KeySelector Reference](keyselectors.md).

## ValueFrom Schema

```yaml
# Default value
valueFrom:
  default:
    value: string

# Argument reference  
valueFrom:
  argRef:
    name: string

# Template
valueFrom:
  template:
    string: string
    variables:
      - name: string
        valueFrom: ValueFrom

# Function call
valueFrom:
  call:
    function: string
    args:
      - name: string
        valueFrom: ValueFrom

# Script execution
valueFrom:
  script:
    exec: string
    args:
      - name: string
        valueFrom: ValueFrom
    env:
      - name: string
        valueFrom: ValueFrom

# File inclusion
valueFrom:
  file:
    app: string                  # Optional: subdirectory
    source: string               # Required: filename

# Pipeline
valueFrom:
  pipeline:
    - valueFrom: ValueFrom
      output: string
```

## Data Types

- `string` - Text value
- `boolean` - true/false
- `[type]` - Array of type
- `ValueFrom` - Value generation expression

## See Also

- [Configuration Files](../user-guide/configuration.md) - Usage guide
- [ValueFrom Types](valuefrom.md) - Detailed ValueFrom reference