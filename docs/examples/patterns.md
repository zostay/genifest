# Common Patterns

Collection of common patterns and best practices for using Genifest.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Environment-Specific Values

Pattern for managing different values across environments:

```yaml
functions:
  - name: get-replicas
    params:
      - name: environment
        required: true
    valueFrom:
      script:
        exec: get-replicas
        args:
          - name: environment
            valueFrom:
              argRef:
                name: environment

changes:
  - tag: production
    fileSelector: "*-deployment.yaml"
    keySelector: .spec.replicas
    valueFrom:
      call:
        function: get-replicas
        args:
          - name: environment
            valueFrom:
              default:
                value: production
```

## Image Tag Management

Dynamic image tag generation:

```yaml
functions:
  - name: get-image-tag
    params:
      - name: service
        required: true
      - name: environment
        required: true
    valueFrom:
      template:
        string: "${service}:${environment}-${BUILD_NUMBER}"
        
changes:
  - fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.containers[] | select(.name == "app") | .image
    valueFrom:
      call:
        function: get-image-tag
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

## Advanced Pipeline Patterns

Examples using the new array iteration, filtering, and pipeline capabilities:

### Container-Specific Updates

```yaml
changes:
  # Update frontend container image using pipeline
  - fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.containers[] | select(.name == "frontend") | .image
    valueFrom:
      template:
        string: "frontend:${BUILD_TAG}"
        variables:
          - name: BUILD_TAG
            valueFrom:
              script:
                exec: "get-build-tag.sh"
                
  # Update backend container resources
  - fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.containers[] | select(.name == "backend") | .resources.limits.memory
    valueFrom:
      default:
        value: "1Gi"
        
  # Set environment variable for specific container
  - fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.containers[] | select(.name == "api") | .env[0].value
    valueFrom:
      template:
        string: "https://api.${ENVIRONMENT}.example.com"
```

### Multi-Container Deployments

```yaml
changes:
  # Update sidecar container image
  - tag: sidecar-update
    fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.containers[] | select(.name == "istio-proxy") | .image
    valueFrom:
      default:
        value: "istio/proxyv2:1.16.0"
        
  # Configure logging sidecar
  - fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.containers[] | select(.name == "fluentd") | .volumeMounts[0].mountPath
    valueFrom:
      default:
        value: "/var/log/app"
        
  # Update init container command
  - fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.initContainers[] | select(.name == "migration") | .command[1]
    valueFrom:
      template:
        string: "migrate --env=${ENVIRONMENT}"
```

### Service and Volume Management

```yaml
changes:
  # Update specific volume configuration
  - fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.volumes[] | select(.name == "config-volume") | .configMap.name
    valueFrom:
      template:
        string: "${SERVICE_NAME}-config"
        
  # Configure persistent volume claims
  - fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.volumes[] | select(.name == "data-volume") | .persistentVolumeClaim.claimName
    valueFrom:
      template:
        string: "${SERVICE_NAME}-data-pvc"
        
  # Update service account for specific containers
  - fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.containers[] | select(.name == "worker") | .securityContext.runAsUser
    valueFrom:
      default:
        value: "1001"
```

### Complex Filtering Examples

```yaml
changes:
  # Update all non-sidecar containers
  - fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.containers[] | select(.name != "istio-proxy") | .imagePullPolicy
    valueFrom:
      default:
        value: "IfNotPresent"
        
  # Configure specific port for named container
  - fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.containers[] | select(.name == "web-server") | .ports[0].containerPort
    valueFrom:
      default:
        value: "8080"
        
  # Set resource requests for application containers (not sidecars)
  - fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.containers[] | select(.name == "app") | .resources.requests.cpu
    valueFrom:
      template:
        string: "${CPU_REQUEST}"
        variables:
          - name: CPU_REQUEST
            valueFrom:
              script:
                exec: "calculate-cpu-request.sh"
                args:
                  - name: "environment"
                    valueFrom:
                      default:
                        value: "production"
```

### Environment-Specific Container Configuration

```yaml
changes:
  # Production-specific container settings
  - tag: production
    fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.containers[] | select(.name == "app") | .env[0].value
    valueFrom:
      default:
        value: "production"
        
  # Staging-specific container settings
  - tag: staging
    fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.containers[] | select(.name == "app") | .env[0].value
    valueFrom:
      default:
        value: "staging"
        
  # Debug containers only in development
  - tag: development
    fileSelector: "*-deployment.yaml"
    keySelector: .spec.template.spec.containers[] | select(.name == "debug-helper") | .image
    valueFrom:
      default:
        value: "debug-tools:latest"
```

### Service Configuration with Pipelines

```yaml
changes:
  # Update service ports based on container configuration
  - fileSelector: "*-service.yaml"
    keySelector: .spec.ports[] | select(.name == "http") | .port
    valueFrom:
      default:
        value: "80"
        
  # Configure ingress for specific services
  - fileSelector: "*-ingress.yaml"
    keySelector: .spec.rules[] | select(.host == "api.example.com") | .http.paths[0].backend.service.name
    valueFrom:
      template:
        string: "${SERVICE_NAME}-api"
        
  # Update load balancer configuration
  - fileSelector: "*-service.yaml"
    keySelector: .metadata.annotations["service.beta.kubernetes.io/aws-load-balancer-type"]
    valueFrom:
      default:
        value: "nlb"
```

## Best Practices for Pipeline Patterns

### Naming Conventions
- Use descriptive container names for reliable filtering
- Prefer semantic names over positional indices
- Use consistent naming across deployments

### Performance Considerations
- Simple field access: `.spec.replicas` (fastest)
- Container filtering: `.containers[] | select(.name == "app")` (moderate)
- Complex pipelines: `.containers[] | select(.name == "app") | .env[0].value` (slower)

### Maintainability
- Use pipeline expressions for container selection by name
- Fall back to index-based access only when names aren't available
- Keep pipeline expressions focused and readable

## See Also

- [Guestbook Tutorial](guestbook.md) - Complete example
- [Value Generation](../user-guide/value-generation.md) - ValueFrom reference