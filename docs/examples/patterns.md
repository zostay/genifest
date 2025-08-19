# Common Patterns

Collection of common patterns and best practices for using Genifest.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Environment-Specific Values

Pattern for managing different values across environments:

```yaml
functions:
  - name: "get-replicas"
    params:
      - name: "environment"
        required: true
    valueFrom:
      template:
        string: '{{ if eq .environment "production" }}5{{ else }}2{{ end }}'

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

## Image Tag Management

Dynamic image tag generation:

```yaml
functions:
  - name: "get-image-tag"
    params:
      - name: "service"
        required: true
      - name: "environment"
        required: true
    valueFrom:
      template:
        string: "${service}:${environment}-${BUILD_NUMBER}"
        
changes:
  - fileSelector: "*-deployment.yaml"
    keySelector: ".spec.template.spec.containers[0].image"
    valueFrom:
      call:
        function: "get-image-tag"
```

## Secret Management

Including secrets from external files:

```yaml
changes:
  - fileSelector: "*-secret.yaml"
    keySelector: ".data.config"
    valueFrom:
      file:
        app: "secrets"
        source: "config.yaml"
```

## See Also

- [Guestbook Tutorial](guestbook.md) - Complete example
- [Value Generation](../user-guide/value-generation.md) - ValueFrom reference