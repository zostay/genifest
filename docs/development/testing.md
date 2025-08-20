# Testing

Testing guidelines and strategies for Genifest development.

!!! note "Work in Progress"
    This documentation page is being developed. Please check back soon for complete content.

## Testing Philosophy

Genifest uses a multi-layered testing approach:

- **Unit tests** for individual components
- **Integration tests** for end-to-end workflows
- **Example-based testing** using the guestbook project

## Running Tests

### Basic Testing

```bash
# Run all tests
make test

# Run with verbose output
go test -v ./...

# Run specific package
go test ./internal/changes
```

### Advanced Testing

```bash
# Test with race detection
make test-race

# Generate coverage report
make test-coverage

# Short tests only (skip slow tests)
make test-short

# Run benchmarks
make benchmark
```

## Test Organization

### Directory Structure

```
internal/
├── changes/
│   ├── eval_test.go           # ValueFrom evaluator tests
│   └── integration_test.go    # End-to-end integration tests
├── cmd/
│   └── root_test.go          # CLI command tests
└── config/
    ├── load_test.go          # Configuration loading tests
    ├── types_test.go         # Type validation tests
    └── validation_test.go    # Configuration validation tests
```

### Test Categories

**Unit Tests** (`*_test.go`):

- Test individual functions and methods
- Focus on business logic and edge cases
- Use table-driven tests where appropriate

**Integration Tests** (`integration_test.go`):

- Test complete workflows
- Use real configuration files
- Validate end-to-end behavior

## Writing Tests

### Unit Test Example

```go
func TestDefaultValue(t *testing.T) {
    t.Parallel()
    
    ctx := NewEvalContext(".", "", "", nil)
    
    vf := config.ValueFrom{
        DefaultValue: &config.DefaultValue{
            Value: "test-value",
        },
    }
    
    result, err := ctx.Evaluate(vf)
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    
    expected := "test-value"
    if result != expected {
        t.Errorf("Expected %q, got %q", expected, result)
    }
}
```

### Integration Test Example

```go
func TestGuestbookIntegration(t *testing.T) {
    t.Parallel()
    
    // Load the guestbook configuration
    projectRoot := getProjectRoot(t)
    guestbookDir := filepath.Join(projectRoot, "examples", "guestbook")
    
    cfg, err := config.LoadFromDirectory(guestbookDir)
    if err != nil {
        t.Fatalf("Failed to load guestbook configuration: %v", err)
    }
    
    // Test function evaluation
    applier := NewApplier(cfg)
    // ... test logic
}
```

### Table-Driven Tests

```go
func TestMatchesGlob(t *testing.T) {
    t.Parallel()
    
    tests := []struct {
        name     string
        pattern  string
        path     string
        expected bool
    }{
        {"exact match", "file.yaml", "file.yaml", true},
        {"wildcard", "*.yaml", "test.yaml", true},
        {"no match", "*.yaml", "test.txt", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := matchesGlob(tt.pattern, tt.path)
            if result != tt.expected {
                t.Errorf("Expected %v, got %v", tt.expected, result)
            }
        })
    }
}
```

## Test Utilities

### Test Fixtures

Use the guestbook example as a test fixture:

```go
func getProjectRoot(t *testing.T) string {
    // Find project root by looking for go.mod
    cwd, err := filepath.Abs(".")
    if err != nil {
        t.Fatalf("Failed to get current directory: %v", err)
    }
    
    // Walk up to find go.mod
    dir := cwd
    for {
        goModPath := filepath.Join(dir, "go.mod")
        if _, err := os.Stat(goModPath); err == nil {
            return dir
        }
        
        parent := filepath.Dir(dir)
        if parent == dir {
            break
        }
        dir = parent
    }
    
    // Fallback
    return filepath.Join(cwd, "..", "..", "..")
}
```

### Temporary Directories

```go
func TestWithTempDir(t *testing.T) {
    tmpDir := t.TempDir() // Automatically cleaned up
    
    // Create test files
    configPath := filepath.Join(tmpDir, "genifest.yaml")
    err := os.WriteFile(configPath, []byte("metadata:\n  cloudHome: ."), 0644)
    if err != nil {
        t.Fatalf("Failed to create test file: %v", err)
    }
    
    // Run test
    cfg, err := config.LoadFromDirectory(tmpDir)
    // ... test logic
}
```

## Test Best Practices

### Isolation

- Use `t.Parallel()` for parallel test execution
- Use temporary directories for file operations
- Don't depend on external services

### Clarity

- Use descriptive test names
- Test one thing at a time
- Include both positive and negative test cases

### Coverage

- Test error conditions
- Test edge cases and boundary conditions
- Verify error messages are helpful

### Performance

- Use `testing.Short()` for long-running tests
- Profile tests when needed
- Avoid unnecessary setup in tight loops

## Testing Guidelines

### What to Test

**Always test**:

- Public API functions
- Error conditions
- Edge cases and boundary values
- Complex business logic

**Consider testing**:

- Internal functions with complex logic
- Configuration parsing
- File operations

**Don't test**:

- Trivial getters/setters
- Third-party library behavior
- Generated code

### Test Data

- Use realistic test data
- Test with various input sizes
- Include malformed input
- Test with different file permissions

### Error Testing

```go
func TestErrorConditions(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        expectedErr string
    }{
        {
            name:        "empty input",
            input:       "",
            expectedErr: "input cannot be empty",
        },
        {
            name:        "invalid format",
            input:       "invalid",
            expectedErr: "invalid format",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := ParseInput(tt.input)
            if err == nil {
                t.Fatal("Expected error, got nil")
            }
            if !strings.Contains(err.Error(), tt.expectedErr) {
                t.Errorf("Expected error containing %q, got %q", tt.expectedErr, err.Error())
            }
        })
    }
}
```

## Continuous Integration

Tests run automatically on:

- Pull requests
- Pushes to main branch
- Release creation

CI includes:

- All test suites
- Race condition detection
- Code coverage reporting
- Linting validation

## See Also

- [Contributing](contributing.md) - Development workflow
- [Architecture](architecture.md) - System design
- [Release Process](releases.md) - Release procedures