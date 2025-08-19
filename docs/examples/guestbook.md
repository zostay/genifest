# Guestbook Tutorial

Complete walkthrough of the Genifest guestbook example application.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Overview

The guestbook example demonstrates a complete Genifest setup with:
- Frontend and backend applications
- PostgreSQL database
- Environment-specific configurations
- Tag-based deployments

## Project Structure

```
examples/guestbook/
├── genifest.yaml              # Root configuration
├── manifests/
│   ├── guestbook/
│   │   ├── genifest.yaml     # Guestbook-specific config
│   │   ├── backend-deployment.yaml
│   │   ├── frontend-deployment.yaml
│   │   └── ...
│   └── postgres/
│       ├── deployment.yaml
│       ├── service.yaml
│       └── ...
├── files/
│   └── guestbook/
│       └── app.yaml
└── scripts/
```

## Key Features Demonstrated

- **Function definitions** for reusable value generation
- **Tag-based filtering** for environment-specific deployments
- **Template generation** for dynamic image tags
- **Multi-environment support** (development, staging, production)

## Running the Example

```bash
cd examples/guestbook

# Validate configuration
genifest validate

# Apply all changes
genifest run

# Apply only production changes
genifest run --include-tags production
```

## See Also

- [Quick Start Guide](../getting-started/quickstart.md) - Step-by-step tutorial
- [Common Patterns](patterns.md) - Reusable patterns
- [GitOps Workflows](gitops.md) - Integration examples