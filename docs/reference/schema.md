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