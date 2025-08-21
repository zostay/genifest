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

## Advanced KeySelector Patterns

Examples of complex YAML path expressions using the enhanced keySelector syntax:

### Container Configuration

```yaml
changes:
  # Update specific container image
  - fileSelector: "*-deployment.yaml"
    keySelector: ".spec.template.spec.containers[0].image"
    valueFrom:
      template:
        string: "myapp:${BUILD_TAG}"
        
  # Set resource limits for second container  
  - fileSelector: "*-deployment.yaml"
    keySelector: ".spec.template.spec.containers[1].resources.limits.memory"
    valueFrom:
      default:
        value: "512Mi"
        
  # Update last container's port
  - fileSelector: "*-deployment.yaml"  
    keySelector: ".spec.template.spec.containers[-1].ports[0].containerPort"
    valueFrom:
      default:
        value: "8080"
```

### ConfigMap and Secret Management

```yaml
changes:
  # Update configuration files with special characters in names
  - fileSelector: "configmap.yaml"
    keySelector: ".data.[\"app.yaml\"]"
    valueFrom:
      file:
        source: "application-config.yaml"
        
  # Update nginx configuration
  - fileSelector: "configmap.yaml"
    keySelector: ".data.[\"nginx.conf\"]"
    valueFrom:
      file:
        source: "nginx.conf"
        
  # Complex Kubernetes annotations
  - fileSelector: "*-deployment.yaml"
    keySelector: ".metadata.annotations.[\"deployment.kubernetes.io/revision\"]" 
    valueFrom:
      script:
        exec: "get-revision.sh"
```

### Service and Ingress Configuration

```yaml
changes:
  # Update multiple service ports using array slicing
  - fileSelector: "service.yaml"
    keySelector: ".spec.ports[0].port"
    valueFrom:
      default:
        value: "80"
        
  # Configure ingress host for production
  - tag: "production"
    fileSelector: "ingress.yaml"
    keySelector: ".spec.rules[0].host"
    valueFrom:
      default:
        value: "api.production.example.com"
        
  # Update TLS configuration
  - fileSelector: "ingress.yaml"
    keySelector: ".spec.tls[0].hosts[0]"
    valueFrom:
      template:
        string: "${ENVIRONMENT}.example.com"
        variables:
          - name: "ENVIRONMENT"
            valueFrom:
              default:
                value: "staging"
```

### Complex Nested Structures  

```yaml
changes:
  # Volume mount configuration
  - fileSelector: "*-deployment.yaml"
    keySelector: ".spec.template.spec.volumes[0].configMap.items[0].key"
    valueFrom:
      default:
        value: "application.properties"
        
  # Update init container command
  - fileSelector: "*-deployment.yaml"
    keySelector: ".spec.template.spec.initContainers[0].command[1]"
    valueFrom:
      template:
        string: "--config=/etc/config/${ENV}.yaml"
        
  # Security context configuration
  - fileSelector: "*-deployment.yaml"
    keySelector: ".spec.template.spec.securityContext.runAsUser"
    valueFrom:
      default:
        value: "1000"
```

### Array Slicing Operations

```yaml
changes:
  # Copy environment variables (all elements)
  - fileSelector: "*-deployment.yaml"
    keySelector: ".spec.template.spec.containers[0].env[:]"
    valueFrom:
      template:
        string: "${BASE_ENV_VARS}"
        
  # Update first three ports
  - fileSelector: "service.yaml"  
    keySelector: ".spec.ports[:3]"
    valueFrom:
      pipeline:
        - valueFrom:
            script:
              exec: "generate-ports.sh"
          output: "ports"
          
  # Remove last environment variable by replacing with empty slice
  - fileSelector: "*-deployment.yaml"
    keySelector: ".spec.template.spec.containers[0].env[:-1]"
    valueFrom:
      template:
        string: "${FILTERED_ENV_VARS}"
```

## See Also

- [Guestbook Tutorial](guestbook.md) - Complete example
- [Value Generation](../user-guide/value-generation.md) - ValueFrom reference