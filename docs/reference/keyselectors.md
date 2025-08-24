# KeySelector Reference

Complete reference for keySelector expressions used in Genifest change definitions.

## Overview

KeySelectors use **yq-style path expressions** to specify which parts of YAML documents to modify. This implementation provides a robust, grammar-based parser that supports a carefully chosen subset of the expression syntax used by tools like `yq` and `jq`.

## Basic Syntax

### Field Access

Access fields in YAML objects using dot notation:

```yaml
# Simple field access
keySelector: ".metadata.name"
keySelector: ".spec.replicas"
keySelector: ".data.config"

# Nested field access  
keySelector: ".spec.template.spec"
keySelector: ".metadata.labels.app"
keySelector: ".spec.template.metadata.annotations"
```

### Array Indexing

Access array elements using bracket notation with numeric indices:

```yaml
# Positive indexing (0-based)
keySelector: ".spec.containers[0]"        # First container
keySelector: ".spec.ports[1]"             # Second port
keySelector: ".items[5]"                  # Sixth item

# Negative indexing (from end)
keySelector: ".spec.containers[-1]"       # Last container
keySelector: ".items[-2]"                 # Second-to-last item
```

### Map Key Access with Quotes

Access map keys that contain special characters using quoted strings:

```yaml
# Double quotes
keySelector: ".data.[\"app.yaml\"]"
keySelector: ".labels.[\"app.kubernetes.io/name\"]"
keySelector: ".annotations.[\"deployment.kubernetes.io/revision\"]"

# Single quotes  
keySelector: ".data.['config-file']"
keySelector: ".labels.['custom-key']"
keySelector: ".annotations.['build-timestamp']"
```

### Array Slicing

Extract ranges of elements from arrays using slice notation:

```yaml
# Range slicing [start:end] - excludes end index
keySelector: ".items[1:3]"                # Elements at indices 1 and 2
keySelector: ".spec.containers[0:2]"      # First two containers

# Open-ended slicing
keySelector: ".items[2:]"                 # From index 2 to end
keySelector: ".items[:3]"                 # First 3 elements (indices 0,1,2)
keySelector: ".items[:]"                  # All elements (full copy)

# Negative indices in slicing
keySelector: ".items[:-1]"                # All except last element
keySelector: ".items[-3:]"                # Last 3 elements
```

### Array Iteration and Pipeline Operations

Process all elements in an array and chain operations:

```yaml
# Array iteration - process all containers
keySelector: ".spec.containers[]"

# Filter with select() function
keySelector: ".spec.containers[] | select(.name == \"frontend\")"
keySelector: ".spec.containers[] | select(.name != \"sidecar\")"

# Complete pipeline operations
keySelector: ".spec.containers[] | select(.name == \"frontend\") | .image"
keySelector: ".spec.template.spec.containers[] | select(.name == \"backend\") | .image"

# Filter and access nested properties
keySelector: ".spec.containers[] | select(.name == \"app\") | .ports[0].containerPort"
keySelector: ".spec.volumes[] | select(.name == \"config\") | .configMap.name"
```

### Comparison Operators

Use comparison operators in select() functions:

```yaml
# Equality comparison
keySelector: ".spec.containers[] | select(.name == \"frontend\")"
keySelector: ".metadata.labels[] | select(.key == \"app.kubernetes.io/name\")"

# Inequality comparison  
keySelector: ".spec.containers[] | select(.name != \"sidecar\")"
keySelector: ".spec.volumes[] | select(.name != \"tmp\")"
```

### Alternative/Default Operator

Use the `//` operator to provide fallback values when paths don't exist or are empty:

```yaml
# Basic alternative values
keySelector: ".metadata.annotations[\"missing-annotation\"] // \"default-value\""
keySelector: ".spec.replicas // \"1\""
keySelector: ".data.config // \"fallback-config\""

# Complex alternatives with nested paths
keySelector: ".spec.template.spec.containers[0].resources.limits.memory // \"256Mi\""
keySelector: ".metadata.labels[\"version\"] // \"unknown\""

# Combined with pipelines (alternatives evaluated last)
keySelector: ".spec.containers[] | select(.name == \"app\") | .image // \"default:latest\""
```

## Complex Examples

### Deep Nested Access

```yaml
# Kubernetes-style deep navigation
keySelector: ".spec.template.spec.containers[0].image"
keySelector: ".spec.template.spec.containers[0].ports[0].containerPort"
keySelector: ".spec.template.spec.volumes[1].configMap.items[0].key"

# Complex metadata access
keySelector: ".metadata.annotations.[\"kubectl.kubernetes.io/last-applied-configuration\"]"
keySelector: ".spec.template.metadata.labels.[\"app.kubernetes.io/version\"]"
```

### Mixed Array and Object Operations

```yaml
# Array of objects with field access
keySelector: ".spec.rules[0].host"
keySelector: ".spec.containers[1].env[2].value"
keySelector: ".status.conditions[-1].type"

# Complex ConfigMap operations
keySelector: ".data.[\"application.properties\"]"
keySelector: ".data.[\"nginx.conf\"]" 
keySelector: ".data.[\"config.json\"]"
```

### Real-World Scenarios

```yaml
# Update specific container image by name (modern approach)
keySelector: ".spec.template.spec.containers[] | select(.name == \"frontend\") | .image"
keySelector: ".spec.template.spec.containers[] | select(.name == \"backend\") | .image"

# Update container image by index (legacy approach)
keySelector: ".spec.template.spec.containers[0].image"

# Modify resource limits for specific container with defaults
keySelector: ".spec.template.spec.containers[] | select(.name == \"app\") | .resources.limits.memory // \"256Mi\""
keySelector: ".spec.template.spec.containers[0].resources.limits.memory // \"128Mi\""

# Update environment variables with fallback values
keySelector: ".spec.template.spec.containers[] | select(.name == \"app\") | .env[0].value // \"default-value\""

# Update ConfigMap data with alternatives
keySelector: ".data.[\"app.properties\"] // \"# Default configuration\""
keySelector: ".data.config // \"default: true\""

# Configuration with fallbacks
keySelector: ".spec.replicas // \"3\""                    # Default replica count
keySelector: ".spec.ports[0].port // \"8080\""           # Default port
keySelector: ".spec.rules[0].host // \"localhost\""      # Default host

# Secrets with alternatives
keySelector: ".data.[\"database-password\"] // \"default-password\""
keySelector: ".metadata.annotations[\"backup.policy\"] // \"daily\""

# Volume mounts with defaults  
keySelector: ".spec.template.spec.containers[] | select(.name == \"app\") | .volumeMounts[0].mountPath // \"/data\""
```

### Common Errors

```yaml
# ❌ Invalid syntax
keySelector: ".spec[containers"           # Missing closing bracket
keySelector: ".spec..replicas"            # Double dots not supported  
keySelector: ".spec[0:1:2]"              # Step slicing not supported

# ❌ Runtime errors (detected during execution)
keySelector: ".spec.containers[999]"     # Index out of bounds
keySelector: ".nonexistent.field"        # Field doesn't exist
keySelector: ".spec.replicas[0]"         # Can't index scalar value
```

## Supported vs yq/jq

### ✅ Fully Supported Features

- **Object field access**: `.field`, `.nested.field`
- **Array indexing**: `[0]`, `[-1]`, positive and negative indices
- **Array slicing**: `[1:3]`, `[2:]`, `[:3]`, `[:]`  
- **Array iteration**: `[]` for processing all elements
- **Quoted key access**: `["key"]`, `['key']`, handling special characters
- **Pipeline operations**: `|` chaining multiple operations
- **Alternative operator**: `//` for fallback values when paths don't exist
- **Filtering with select()**: `select(.name == "value")` for conditional filtering
- **Comparison operators**: `==`, `!=` for equality/inequality tests
- **Complex nested paths**: mixing all above operations
- **Grammar-based parsing**: robust expression handling
- **Parse-time validation**: syntax checking before execution

### ❌ Not Supported (by design)

- **Advanced filtering functions**: `map()`, `has()`, `contains()`, `keys()`, `values()`
- **Conditional expressions**: `if-then-else` constructs
- **Arithmetic operations**: `+`, `-`, `*`, `/`, `%`
- **String functions**: `split()`, `join()`, `length()`, regex operations
- **Recursive descent**: `..` (find anywhere)
- **Variable assignment**: setting temporary variables
- **Complex queries**: SQL-like operations with multiple conditions
- **Step slicing**: `[start:end:step]` with step parameter
- **Advanced comparison operators**: `<`, `>`, `<=`, `>=`

## Best Practices

### Clarity and Maintainability
```yaml
# ✅ Good: Clear, specific selectors with reasonable defaults
keySelector: ".spec.template.spec.containers[] | select(.name == \"frontend\") | .image // \"nginx:latest\""
keySelector: ".data.[\"application.yaml\"] // \"default-config\""

# ✅ Good: Modern approach using names instead of indices
keySelector: ".spec.containers[] | select(.name == \"app\") | .image"

# ✅ Good: Provide sensible fallbacks for optional configuration
keySelector: ".spec.replicas // \"3\""
keySelector: ".metadata.annotations[\"backup.policy\"] // \"daily\""

# ⚠️ Acceptable but less maintainable: Index-based access
keySelector: ".spec.template.spec.containers[0].image"

# ❌ Avoid: Overly complex nested expressions
keySelector: ".spec.template.spec.volumes[2].configMap.items[1].path"
```

### Pipeline Best Practices
```yaml
# ✅ Good: Use descriptive container names for filtering
keySelector: ".spec.containers[] | select(.name == \"frontend\") | .image"
keySelector: ".spec.containers[] | select(.name == \"sidecar\") | .env[0].value"

# ✅ Good: Simple pipeline with clear intent
keySelector: ".spec.volumes[] | select(.name == \"config\") | .configMap.name"

# ❌ Avoid: Chaining too many operations
keySelector: ".spec.containers[] | select(.name == \"app\") | .volumeMounts[] | select(.name == \"data\") | .mountPath"
```

### Error Prevention
```yaml
# ✅ Good: Use quoted keys for special characters
keySelector: ".labels.[\"app.kubernetes.io/name\"]"
keySelector: ".data.[\"nginx.conf\"]"

# ❌ Risky: Special characters without quotes (may fail)
keySelector: ".labels.app.kubernetes.io/name"     # Fails: dots in key
```

### Performance Considerations
```yaml
# ✅ Good: Direct access is fastest
keySelector: ".spec.replicas"
keySelector: ".data.config"

# ⚠️ Moderate: Array iteration and filtering (requires processing multiple elements)
keySelector: ".spec.containers[] | select(.name == \"frontend\") | .image"

# ⚠️ Note: Deep nesting is supported but slower
keySelector: ".spec.template.spec.containers[0].env[5].value"

# ⚠️ Slower: Complex pipelines with multiple operations
keySelector: ".spec.containers[] | select(.name == \"app\") | .volumeMounts[0].mountPath"
```

## Testing KeySelectors

Use the `genifest validate` command to test your keySelectors without applying changes:

```bash
# Validate configuration and keySelectors
genifest validate

# See the parsed configuration structure  
genifest config
```

## See Also

- [Configuration Schema](schema.md) - Complete configuration reference
- [Core Concepts](../user-guide/concepts.md) - Understanding the system
- [Examples](../examples/patterns.md) - Real-world usage patterns