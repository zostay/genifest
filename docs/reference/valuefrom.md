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

## EnvironmentRef

Reads values from environment variables with optional default values.

```yaml
valueFrom:
  envRef:
    name: "DATABASE_URL"
    default: "postgresql://localhost:5432/myapp"  # Optional
```

**Configuration:**

- `name` (required) - The environment variable name to read
- `default` (optional) - Fallback value if the environment variable is not set or empty

**Use cases:**
- Reading database connection strings
- Accessing API keys and secrets
- Environment-specific configuration values
- CI/CD pipeline variables

**Examples:**

```yaml
# Required environment variable (fails if not set)
valueFrom:
  envRef:
    name: "API_KEY"
```
```yaml
# Optional with default
valueFrom:
  envRef:
    name: "LOG_LEVEL"
    default: "info"
```
```yaml
# Database configuration
valueFrom:
  envRef:
    name: "DB_HOST"
    default: "localhost"
```

**Best Practices:**
- Use descriptive environment variable names: `DB_HOST` instead of `HOST`
- Provide sensible defaults for non-sensitive configuration
- Never set defaults for secrets or API keys
- Use uppercase with underscores for environment variable names

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

Includes content from external files with optional transient modifications.

```yaml
valueFrom:
  file:
    app: "subdirectory"    # Optional
    source: "file.yaml"    # Required
    changes:               # Optional transient changes
      - keySelector: ".spec.replicas"
        valueFrom:
          default:
            value: "3"
      - documentSelector:  # Optional document filtering
          kind: "Secret"
        keySelector: ".data.password"
        valueFrom:
          default:
            value: "temp-password"
```

**Features:**
- File content inclusion
- Subdirectory support
- Path security validation
- **Transient changes** - Apply modifications to file content in memory without persisting to disk
- **Document selection** - Target specific documents in multi-document files
- **Key selector support** - Modify specific fields using path expressions

**Transient Changes:**
Transient changes allow you to modify file content dynamically without altering the original files. This is useful for scenarios like:
- Injecting secrets that will be encrypted by external tools (e.g., kubeseal)
- Environment-specific modifications during pipeline execution
- Testing configurations without affecting source files

Changes are applied in memory only and are never written back to the source files.

## DocumentRef

References values from a selected document or from other locations within the 
current document.

```yaml
valueFrom:
  documentRef:
    fileSelector: other-file.yaml
    documentSelector:
      metadata.name: the-document
    keySelector: ".spec.key-info"
```

**Configuration:**

- `fileSelector` (optional) - Relative path to the file to work with. If omitted, the current file is used. By "current file", it means the file that is the context of the existing operation. For example, if a change is being applied to a file named `deployment.yaml`, that is the file to which the `documentSelector` and `keySelector` will refer.
- `documentSelector` (optional) - In cases where a file contains multiple documents (in YAML files, documents are separated by `---` separators). This optionally selects the document to work with. If no documentSelector is used, then the current document is the one that will be referred to by via this document reference.
- `keySelector` (required) - Path expression to the value within the current document

**Use cases:**
- Referencing values in other fields, documents, and files
- Creating consistent naming across document sections
- Dynamic field generation based on existing values
- Cross-referencing within complex Kubernetes manifests

**Examples:**

```yaml
# Reference the deployment name in a service
apiVersion: v1
kind: Service
metadata:
  name: my-app-service
spec:
  selector:
    app: "VALUE_TO_BE_REPLACED"  # Will be replaced with documentRef
```

```yaml
# Configuration that replaces the selector
changes:
  - keySelector: ".spec.selector.app"
    valueFrom:
      documentRef:
        keySelector: ".metadata.name"  # References "my-app-service"
```

```yaml
# Reference namespace in multiple places
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: production
spec:
  template:
    spec:
      serviceAccountName: "VALUE_TO_BE_REPLACED"
```

```yaml
# Configuration
changes:
  - keySelector: ".spec.template.spec.serviceAccountName"
    valueFrom:
      template:
        string: "my-app-${namespace}"
        variables:
          - name: "namespace"
            valueFrom:
              documentRef:
                keySelector: ".metadata.namespace"
```

```yaml
# deployment configuration
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: production
spec:
  template:
    spec:
      serviceAccountName: "VALUE_TO_BE_REPLACED"
```

```yaml
# configuration output from an external script
environment: production
myapp-account: the-account
---
environment: staging
myapp-account: the-stager
```

```yaml
# Configuration
changes:
  - keySelector: ".spec.template.spec.serviceAccountName"
    valueFrom:
      documentRef:
        fileSelector: output.yaml
        documentSelector: 
          environment: production
        keySelector: ".myapp-account"
```

**Document Selector Syntax:**

If `documentSelector` is used, it must be provided as a simple map. This is a
very simple matching system where a document matches if all the keys named are
matched.

```yaml
documentSelector:
  metadata.name: Foo
  metadata.labels.pick-me: Bar
```

Dots in the selector will be interpreted as nested items in a map. No special
escaping or advanced matching is provided. Value matching is by exact string
match only.

**Key Selector Syntax:**

DocumentRef supports the same key selector syntax as changes:

- `.metadata.name` - Simple field access
- `.spec.containers[0].image` - Array index access
- `.spec.containers[] | select(.name == "app") | .image` - Advanced filtering

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
- [KeySelector Syntax](keyselectors.md) - Path expression reference