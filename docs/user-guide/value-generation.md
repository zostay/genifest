# Value Generation

Deep dive into Genifest's ValueFrom expression system for dynamic value generation.

## Overview

ValueFrom expressions are the core of Genifest's dynamic value generation system. They provide multiple ways to generate values for configuration changes, from simple literals to complex pipeline operations.

## Core ValueFrom Types

### DefaultValue

The simplest ValueFrom type for literal string values.

```yaml
valueFrom:
  default:
    value: "literal-string"
```

**Use cases:**
- Static configuration values
- Default fallbacks
- Simple string constants

### EnvironmentRef

Read values from environment variables with optional defaults.

```yaml
valueFrom:
  envRef:
    name: "DATABASE_URL"
    default: "postgresql://localhost:5432/myapp"  # Optional
```

**Use cases:**
- Database connection strings
- API keys and secrets
- Environment-specific configuration
- CI/CD pipeline variables

**Examples:**
```yaml
# Database host with fallback
changes:
  - keySelector: ".spec.template.spec.containers[0].env[0].value"
    valueFrom:
      envRef:
        name: "DB_HOST"
        default: "localhost"
```
```yaml
# Required API key (no default)
changes:
  - keySelector: ".spec.template.spec.containers[0].env[1].value"
    valueFrom:
      envRef:
        name: "API_KEY"
```

### DocumentRef

Reference values from other locations within the document:

```yaml
valueFrom:
  documentRef:
    keySelector: ".metadata.name"
```

Or in an external document:

```yaml
valueFrom:
  documentRef:
    fileSelector: outputs.yaml
    documentSelector:
      environment: production
    keySelector: .outputs.deploy.image-name
```

**Use cases:**
- Consistent naming across document sections
- Dynamic field generation based on existing values
- Cross-referencing within Kubernetes manifests
- Referencing external configuration

**Examples:**
```yaml
# Use deployment name as app selector
changes:
  - keySelector: ".spec.selector.matchLabels.app"
    valueFrom:
      documentRef:
        keySelector: ".metadata.name"
```
```yaml
# Reference namespace in service account name
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

### ArgumentRef

Reference variables from the current evaluation context.

```yaml
valueFrom:
  argRef:
    name: "variable-name"
```

**Use cases:**
- Function parameters
- Pipeline step outputs
- Template variables

### BasicTemplate

Template strings with variable substitution using `${variable}` syntax.

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

**Advanced templating:**
```yaml
valueFrom:
  template:
    string: "${app}-${env}-${component}"
    variables:
      - name: "app"
        valueFrom:
          documentRef:
            keySelector: ".metadata.labels.app"
      - name: "env"
        valueFrom:
          envRef:
            name: "ENVIRONMENT"
            default: "dev"
      - name: "component"
        valueFrom:
          default:
            value: "backend"
```

## Advanced ValueFrom Types

### FunctionCall

Call reusable functions with parameters.

```yaml
valueFrom:
  call:
    function: "get-image-tag"
    args:
      - name: "environment"
        valueFrom:
          envRef:
            name: "DEPLOY_ENV"
            default: "dev"
```

Functions are defined in the `functions` section:
```yaml
functions:
  - name: "get-image-tag"
    params:
      - name: "environment"
        required: true
    valueFrom:
      template:
        string: "myapp:${environment}-latest"
        variables:
          - name: "environment"
            valueFrom:
              argRef:
                name: "environment"
```

### ScriptExec

Execute external scripts for dynamic value generation.

```yaml
valueFrom:
  script:
    exec: "get-commit-hash.sh"
    args:
      - name: "branch"
        valueFrom:
          envRef:
            name: "GIT_BRANCH"
            default: "main"
    env:
      - name: "REPO_PATH"
        valueFrom:
          default:
            value: "/workspace"
```

### FileInclusion

Include content from external files with optional transient modifications.

```yaml
valueFrom:
  file:
    app: "templates"         # Optional subdirectory
    source: "base.yaml"      # Required filename
    changes:                 # Optional transient changes
      - keySelector: ".metadata.name"
        valueFrom:
          template:
            string: "${name}-${env}"
            variables:
              - name: "name"
                valueFrom:
                  documentRef:
                    keySelector: ".metadata.name"
              - name: "env"
                valueFrom:
                  envRef:
                    name: "ENVIRONMENT"
```

**Transient changes** allow you to modify included files without persisting changes to disk.

### CallPipeline

Chain multiple operations together, passing outputs between steps.

```yaml
valueFrom:
  pipeline:
    - valueFrom:
        envRef:
          name: "BASE_IMAGE"
          default: "nginx"
      output: "base"
    - valueFrom:
        template:
          string: "${base}:${tag}"
          variables:
            - name: "base"
              valueFrom:
                argRef:
                  name: "base"
            - name: "tag"
              valueFrom:
                script:
                  exec: "get-latest-tag.sh"
                  args:
                    - name: "image"
                      valueFrom:
                        argRef:
                          name: "base"
```

## Combining ValueFrom Types

ValueFrom expressions can be combined to create sophisticated value generation:

```yaml
changes:
  - keySelector: ".spec.template.spec.containers[0].image"
    valueFrom:
      pipeline:
        - valueFrom:
            envRef:
              name: "REGISTRY_URL"
              default: "docker.io"
          output: "registry"
        - valueFrom:
            template:
              string: "${registry}/${app}:${tag}"
              variables:
                - name: "registry"
                  valueFrom:
                    argRef:
                      name: "registry"
                - name: "app"
                  valueFrom:
                    documentRef:
                      keySelector: ".metadata.name"
                - name: "tag"
                  valueFrom:
                    call:
                      function: "get-version"
                      args:
                        - name: "environment"
                          valueFrom:
                            envRef:
                              name: "DEPLOY_ENV"
```

## Best Practices

### Value Generation Strategy

1. **Start simple**: Use `default` for static values
2. **Add environment awareness**: Use `envRef` for environment-specific values
3. **Enable reusability**: Use `call` for common patterns
4. **Build complexity**: Use `pipeline` for multi-step transformations

### Error Handling

- Use `default` values in `envRef` for non-critical configuration
- Never set defaults for secrets or API keys
- Test complex pipelines thoroughly
- Validate function parameters

### Performance Considerations

- Minimize script execution in pipelines
- Cache function results where possible
- Use `documentRef` instead of re-reading files
- Keep template complexity reasonable

## See Also

- [ValueFrom Types](../reference/valuefrom.md) - Complete reference
- [Functions](../reference/functions.md) - Function definition guide
- [KeySelector Syntax](../reference/keyselectors.md) - Path expression reference
- [Core Concepts](concepts.md) - Understanding the system