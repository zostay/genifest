# GitOps Workflows

Integration patterns for using Genifest with GitOps tools like ArgoCD and Flux.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Overview

Genifest is designed to work seamlessly with GitOps continuous deployment workflows. This page covers integration patterns and best practices.

## ArgoCD Integration

Basic setup for using Genifest with ArgoCD:

```yaml
# .argocd/application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-app
spec:
  source:
    repoURL: https://github.com/myorg/my-app
    targetRevision: HEAD
    path: k8s
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

## Flux Integration

Example Flux configuration:

```yaml
# .flux/kustomization.yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: my-app
spec:
  sourceRef:
    kind: GitRepository
    name: my-app
  path: "./k8s"
  prune: true
```

## CI/CD Pipeline

Example GitHub Actions workflow for GitOps:

```yaml
name: Update Manifests
on:
  push:
    branches: [main]
    
jobs:
  update-manifests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Update manifests
        run: |
          genifest run --include-tags production
          git add .
          git commit -m "Update manifests [skip ci]"
          git push
```

## Best Practices

- Use tags to separate environments
- Keep configuration close to manifests
- Version control all generated changes
- Use automated validation in CI

## See Also

- [Guestbook Tutorial](guestbook.md) - Complete example
- [Common Patterns](patterns.md) - Reusable patterns