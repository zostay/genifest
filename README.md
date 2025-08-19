# genifest

⚠️ **Alpha Software**: This is alpha software under active development. APIs and configuration formats may change without notice.

Genifest is a Kubernetes manifest generation tool that creates deployment manifests from templates for GitOps workflows. It processes configuration files to generate Kubernetes resources with dynamic value substitution, designed for use with GitOps continuous deployment processes like ArgoCD.

## How It Works

Genifest uses a metadata-driven approach to discover and process YAML configuration files. It applies dynamic changes to Kubernetes manifests and related configuration files based on configurable rules, allowing you to maintain a single set of manifests that evolve over time. This is useful for:
- Generating environment-specific deployments
- Embed secrets
- Incrementing image tags according to business rules
- Managing variations in configuration files

This is all done without templating and aiming at idempotency so there is a single source of truth.

### Core Concepts

- **Configuration Discovery**: Starts with a root `genifest.yaml` file and discovers additional configurations through metadata-defined paths
- **Dynamic Value Generation**: Uses `ValueFrom` expressions to generate values from functions, templates, scripts, files, and more  
- **Tag-Based Filtering**: Apply different sets of changes based on tags (based on environment, e.g., `production`, `staging`, `development`, or change type, e.g., `secrets`, `config`, `image-version`, depending your needs)

### Configuration Structure

The system uses a hierarchical configuration approach:

1. **Root Configuration**: A `genifest.yaml` file in your project root defines metadata paths and global settings
2. **Distributed Configurations**: Additional `genifest.yaml` files in subdirectories provide scoped configurations
3. **Automatic Discovery**: Directories without explicit configurations get synthetic configs containing all YAML files

### Recommended Usage Pattern

For optimal organization and maintainability, use multiple `genifest.yaml` files throughout your project to keep changes close to the YAML files being managed:

```
project-root/
├── genifest.yaml                 # Root configuration
├── k8s/
│   ├── app1/
│   │   ├── genifest.yaml        # App1-specific changes
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   └── app2/
│       ├── genifest.yaml        # App2-specific changes  
│       ├── deployment.yaml
│       └── configmap.yaml
├── scripts/
│   └── build-image.sh
├── files/
│   └── app1/                   # App1-specific files
        └── config-file.yaml
```

## Primary Use Case

Genifest is designed for managing YAML files that are directly deployed via GitOps CD processes such as ArgoCD. In this workflow:

1. Developers define base Kubernetes manifests
2. Genifest generates environment-specific variations  
3. GitOps tools detect changes and deploy automatically
4. No manual kubectl or deployment steps required

While this is the primary use case, genifest can be adapted for other YAML processing workflows where dynamic value substitution is needed.

## Value Generation System

Genifest provides multiple ways to generate dynamic values:

### DefaultValue
Returns literal string values:
```yaml
valueFrom:
  default:
    value: "literal-string"
```

### ArgumentRef  
References variables from the current evaluation context:
```yaml
valueFrom:
  argRef:
    name: "variable-name"
```

### BasicTemplate
Template strings with `${variable}` substitution:
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
Calls named functions with arguments:
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

### ScriptExec
Executes scripts from the scripts directory:
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

### FileInclusion
Includes content from files in the files directory:
```yaml
valueFrom:
  file:
    app: "myapp"  # optional subdirectory
    source: "config.yaml"
```

### CallPipeline
Chains multiple operations together:
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

## Security and Safety

- All script execution uses the configured `CloudHome` as working directory
- File inclusion is restricted to the configured files directory  
- Path traversal is prevented through path validation
- Environment variables are isolated for script execution

## Installation

To install the tool, run the following command:

```bash
curl -L https://raw.githubusercontent.com/zostay/genifest/master/tools/install.sh | sh
```

Or to install from source, you'll need go 1.22 installed:

```bash
go install github.com/zostay/genifest/cmd/genifest@latest
```

## Usage

Genifest provides several subcommands for different operations. All commands can optionally specify a directory argument to operate from a location other than the current working directory.

### Core Commands

```bash
# Apply all changes (run from directory containing genifest.yaml)
genifest run

# Apply only production changes  
genifest run --include-tags production

# Apply all except staging changes
genifest run --exclude-tags staging

# Apply changes from a different directory
genifest run path/to/project

# Show version information
genifest version

# List all available tags in configuration
genifest tags

# Validate configuration without applying changes
genifest validate

# Display merged configuration 
genifest config
```

### Enhanced Output

The run command provides detailed progress reporting:
- Shows total change definitions found and how many will be processed
- Reports each change with full context: `file -> document[index] -> key: old → new`
- Distinguishes between changes applied vs actual modifications made
- Summarizes final results with file modification counts

### Tag Filtering

- **No flags**: All changes applied (tagged and untagged)
- **Include only**: Only changes matching include patterns
- **Exclude only**: All changes except those matching exclude patterns
- **Both flags**: Changes matching include but not exclude patterns
- **Glob patterns supported**: `prod*`, `test-*`, etc.

## Contributing

We welcome contributions! To get started:

### Development Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/zostay/genifest.git
   cd genifest
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Run tests**:
   ```bash
   go test ./...
   ```

4. **Build locally**:
   ```bash
   go build -o genifest ./cmd/genifest
   ```

5. **Run linting**:
   ```bash
   golangci-lint run --timeout=5m
   ```

### Development Workflow

1. Fork the repository on GitHub
2. Create a feature branch from `master`
3. Make your changes with tests
4. Ensure all tests pass and linting is clean
5. Submit a pull request with a clear description

### Testing

The project includes comprehensive tests covering:
- Individual ValueFrom evaluator functionality
- Error handling and edge cases  
- Integration with real configuration files (guestbook example)
- Context immutability and variable scoping
- Script execution with different argument and environment configurations

## LICENSE

Copyright © 2025 Qubling LLC

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.