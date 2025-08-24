# Genifest JSON Schema

This directory contains the JSON Schema for Genifest configuration files.

## Files

- `genifest-schema.json` - The complete JSON Schema for validating `genifest.yaml` files

## IDE Integration

### VS Code

To enable schema validation and autocompletion in VS Code, add this to your VS Code settings or workspace `.vscode/settings.json`:

```json
{
  "yaml.schemas": {
    "./schema/genifest-schema.json": ["genifest.yaml", "**/genifest.yaml"]
  }
}
```

### JetBrains IDEs (IntelliJ, GoLand, etc.)

1. Go to **File** → **Settings** → **Languages & Frameworks** → **Schemas and DTDs** → **JSON Schema Mappings**
2. Click **+** to add a new schema
3. Set **Schema file or URL** to the path of `schema/genifest-schema.json`
4. Add file patterns: `genifest.yaml` and `**/genifest.yaml`

## Schema Validation

The schema is automatically used by the `genifest validate` command in different modes:

### Validation Modes

- **Permissive (default)**: `genifest validate` - Ignores unknown fields
- **Warning mode**: `genifest validate --warn` - Shows warnings for unknown fields but continues
- **Strict mode**: `genifest validate --strict` - Fails validation on unknown fields

### Examples

```bash
# Default permissive validation
genifest validate examples/guestbook

# Show warnings for unknown fields
genifest validate examples/guestbook --warn

# Strict validation (fail on unknown fields)
genifest validate examples/guestbook --strict
```

## Schema Features

The schema provides:

- **Complete structure validation** - Validates all configuration sections
- **Field type checking** - Ensures correct data types
- **Pattern validation** - Validates identifiers, tags, and paths
- **Union type validation** - Validates ValueFrom expressions (exactly one field required)
- **IDE completion** - Provides autocompletion and documentation in IDEs
- **Documentation** - Rich descriptions for all fields and structures

## Keeping Schema in Sync

The schema is embedded into the Genifest binary and used for runtime validation. The Go types in `internal/config/types.go` should be kept in sync with this schema.

If you modify the configuration structure:

1. Update the Go types in `internal/config/types.go`
2. Update the JSON schema in `schema/genifest-schema.json`
3. Test validation with `genifest validate --strict` on example configurations
4. Update documentation if needed

## Schema Structure

The schema defines these main types:

- **Config** - Root configuration object
- **MetaConfig** - Metadata with directory paths and capabilities
- **PathConfig** - Unified directory configuration with depth
- **FilesConfig** - File selection with include/exclude patterns
- **FunctionDefinition** - Reusable function definitions
- **ChangeOrder** - Change definitions for file modifications
- **ValueFrom** - Union type for value generation (functions, templates, scripts, etc.)

Each type includes validation rules, documentation, and proper constraints.