# Quick Start

This guide will get you up and running with Genifest in just a few minutes using the included guestbook example.

## Prerequisites

- Genifest installed ([Installation Guide](installation.md))
- Basic familiarity with Kubernetes YAML files
- Text editor of your choice

## Your First Genifest Project

### Step 1: Explore the Example

Genifest comes with a complete guestbook example. Let's explore it:

```bash
# Navigate to the example
cd examples/guestbook

# Look at the project structure
tree
```

You'll see a structure like this:

```
examples/guestbook/
├── genifest.yaml          # Root configuration
├── files/                 # Template files
│   └── guestbook/
│       └── app.yaml
├── manifests/            # Kubernetes manifests
│   ├── guestbook/
│   │   ├── genifest.yaml
│   │   ├── backend-deployment.yaml
│   │   ├── frontend-deployment.yaml
│   │   └── ...
│   └── postgres/
│       ├── deployment.yaml
│       ├── service.yaml
│       └── ...
└── scripts/              # Custom scripts
```

### Step 2: Examine the Configuration

Look at the root configuration:

```bash
cat genifest.yaml
```

```yaml
metadata:
  cloudHome: "."
  scripts: ["scripts"]
  manifests: ["manifests"]
  files: ["files"]

functions:
  - name: "get-replicas"
    params:
      - name: "environment"
        required: true
    valueFrom:
      default:
        value: "2"
```

This defines:
- **Metadata**: Where to find scripts, manifests, and files
- **Functions**: Reusable value generators

### Step 3: Validate the Configuration

Before applying changes, validate everything is correct:

```bash
genifest validate
```

Expected output:
```
✅ Configuration validation successful
  • 3 function definitions validated
  • 13 files found and accessible
  • 3 change definitions validated
```

### Step 4: Explore Available Commands

```bash
# Show all available tags
genifest tags

# Display the merged configuration
genifest config

# Show help for any command
genifest run --help
```

### Step 5: Apply Changes

Now apply the changes to see Genifest in action:

```bash
genifest run
```

You'll see detailed output like:
```
🔍 Configuration loaded:
  • 3 total change definition(s) found
  • 3 change definition(s) will be processed (all changes)
  • 13 file(s) to examine

  ✏️  manifests/guestbook/backend-deployment.yaml -> document[0] -> .spec.replicas: 1 → 2
  ✓  manifests/guestbook/frontend-deployment.yaml -> document[0] -> .spec.replicas: 2 (no change)
📝 Updated file: manifests/guestbook/backend-deployment.yaml (1 changes)

✅ Successfully completed processing:
  • 2 change application(s) processed
  • 1 change application(s) resulted in actual modifications
  • 1 file(s) were updated
```

### Step 6: Try Tag-Based Filtering

Apply only production-tagged changes:

```bash
genifest run --include-tags production
```

This will only apply changes marked with the "production" tag.

## Understanding What Happened

### Configuration Discovery

Genifest discovered configurations in this order:

1. **Root config** (`genifest.yaml`) - Defined metadata and functions
2. **Subdirectory configs** - Found `manifests/guestbook/genifest.yaml` with specific changes
3. **Synthetic configs** - Created automatic configs for directories without `genifest.yaml`

### Value Generation

The changes used the `get-replicas` function to set replica counts:

```yaml
changes:
  - tag: "production"
    fileSelector: "*-deployment.yaml"
    keySelector: ".spec.replicas"
    valueFrom:
      call:
        function: "get-replicas"
        args:
          - name: "environment"
            valueFrom:
              default:
                value: "production"
```

## Next Steps: Create Your Own Project

### 1. Create a Basic Project

```bash
mkdir my-k8s-project
cd my-k8s-project

# Create the root configuration
cat > genifest.yaml << EOF
metadata:
  cloudHome: "."
  manifests: ["k8s"]

functions:
  - name: "get-replicas"
    params:
      - name: "env"
        required: true
    valueFrom:
      template:
        string: '{{ if eq .env "prod" }}5{{ else }}2{{ end }}'

changes:
  - fileSelector: "*-deployment.yaml"
    keySelector: ".spec.replicas"
    valueFrom:
      call:
        function: "get-replicas"
        args:
          - name: "env"
            valueFrom:
              default:
                value: "dev"
EOF
```

### 2. Add Kubernetes Manifests

```bash
mkdir -p k8s
cat > k8s/app-deployment.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 1  # This will be updated by genifest
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: app
        image: nginx:latest
EOF
```

### 3. Test Your Configuration

```bash
# Validate
genifest validate

# See what would change
genifest run

# Check the result
cat k8s/app-deployment.yaml
```

## Common Patterns

### Environment-Specific Values

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

changes:
  - tag: "staging"
    fileSelector: "*-deployment.yaml"
    keySelector: ".spec.template.spec.containers[0].image"
    valueFrom:
      call:
        function: "get-image-tag"
        args:
          - name: "environment"
            valueFrom:
              default:
                value: "staging"
```

### Multiple Environments

Use tags to target specific environments:

```bash
# Development
genifest run --include-tags dev

# Staging  
genifest run --include-tags staging

# Production
genifest run --include-tags prod
```

## Troubleshooting

### Common Issues

1. **"Configuration file not found"**
   ```bash
   # Make sure you're in the directory with genifest.yaml
   ls genifest.yaml
   ```

2. **"No changes applied"**
   ```bash
   # Check if your file selectors match your files
   genifest config  # Shows merged config
   genifest validate  # Validates everything
   ```

3. **"Function not found"**
   ```bash
   # Verify function definitions in config
   genifest config | grep -A5 functions
   ```

## What's Next?

- 📖 **[User Guide](../user-guide/concepts.md)** - Learn core concepts in depth
- 🔧 **[Configuration Reference](../user-guide/configuration.md)** - Complete configuration options
- 💡 **[Examples](../examples/patterns.md)** - Real-world patterns and use cases
- 🚀 **[GitOps Integration](../examples/gitops.md)** - Setting up with ArgoCD/Flux

---

Next: [Configuration Guide →](configuration.md)