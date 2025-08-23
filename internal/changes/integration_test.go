package changes

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/zostay/genifest/internal/config"
)

// TestGuestbookIntegration tests the evaluation system with the guestbook example configuration.
func TestGuestbookIntegration(t *testing.T) {
	t.Parallel()
	// Load the guestbook configuration
	projectRoot := getProjectRoot(t)
	guestbookDir := filepath.Join(projectRoot, "examples", "guestbook")

	cfg, err := config.LoadFromDirectory(guestbookDir)
	if err != nil {
		t.Fatalf("Failed to load guestbook configuration: %v", err)
	}

	// Create an applier
	applier := NewApplier(cfg)

	// Test function evaluation
	t.Run("EvaluateFunctions", func(t *testing.T) {
		t.Parallel()
		// Test get-replicas function
		replicasChange := config.ChangeOrder{
			Tag: "production",
			DocumentRef: config.DocumentRef{
				FileSelector: "manifests/*/deployment.yaml",
				KeySelector:  ".spec.replicas",
			},
			ValueFrom: config.ValueFrom{
				FunctionCall: &config.FunctionCall{
					Name: "get-replicas",
					Arguments: []config.Argument{
						{
							Name: "environment",
							ValueFrom: config.ValueFrom{
								DefaultValue: &config.DefaultValue{Value: "production"},
							},
						},
					},
				},
			},
		}

		// Load the document for context
		deploymentPath := filepath.Join(guestbookDir, "manifests/guestbook/frontend-deployment.yaml")
		doc, err := loadYAMLDocument(t, deploymentPath)
		if err != nil {
			t.Fatalf("Failed to load deployment document: %v", err)
		}

		value, err := applier.EvaluateChangeValue(replicasChange, "manifests/guestbook/frontend-deployment.yaml", doc)
		if err != nil {
			t.Fatalf("Failed to evaluate get-replicas function: %v", err)
		}

		expected := "2" // The function returns "2" for all environments in our simplified version
		if value != expected {
			t.Errorf("Expected replicas value %q, got %q", expected, value)
		}

		// Test get-image-tag function
		imageChange := config.ChangeOrder{
			DocumentRef: config.DocumentRef{
				FileSelector: "manifests/guestbook/*-deployment.yaml",
				KeySelector:  ".spec.template.spec.containers[0].image",
			},
			ValueFrom: config.ValueFrom{
				FunctionCall: &config.FunctionCall{
					Name: "get-image-tag",
					Arguments: []config.Argument{
						{
							Name: "service",
							ValueFrom: config.ValueFrom{
								DefaultValue: &config.DefaultValue{Value: "guestbook-frontend"},
							},
						},
						{
							Name: "environment",
							ValueFrom: config.ValueFrom{
								DefaultValue: &config.DefaultValue{Value: "dev"},
							},
						},
					},
				},
			},
		}

		value, err = applier.EvaluateChangeValue(imageChange, "manifests/guestbook/frontend-deployment.yaml", doc)
		if err != nil {
			t.Fatalf("Failed to evaluate get-image-tag function: %v", err)
		}

		expected = "guestbook-frontend:dev-latest"
		if value != expected {
			t.Errorf("Expected image tag %q, got %q", expected, value)
		}

		// Test get-database-host function
		dbChange := config.ChangeOrder{
			DocumentRef: config.DocumentRef{
				FileSelector: "manifests/guestbook/backend-deployment.yaml",
				KeySelector:  ".spec.template.spec.containers[0].env[0].value",
			},
			ValueFrom: config.ValueFrom{
				FunctionCall: &config.FunctionCall{
					Name: "get-database-host",
					Arguments: []config.Argument{
						{
							Name: "environment",
							ValueFrom: config.ValueFrom{
								DefaultValue: &config.DefaultValue{Value: "staging"},
							},
						},
					},
				},
			},
		}

		// Load the backend deployment document for the database host test
		backendPath := filepath.Join(guestbookDir, "manifests/guestbook/backend-deployment.yaml")
		backendDoc, err := loadYAMLDocument(t, backendPath)
		if err != nil {
			t.Fatalf("Failed to load backend deployment document: %v", err)
		}

		value, err = applier.EvaluateChangeValue(dbChange, "manifests/guestbook/backend-deployment.yaml", backendDoc)
		if err != nil {
			t.Fatalf("Failed to evaluate get-database-host function: %v", err)
		}

		expected = "postgres-service"
		if value != expected {
			t.Errorf("Expected database host %q, got %q", expected, value)
		}
	})

	// Test applying changes with tags
	t.Run("ApplyChangesWithTags", func(t *testing.T) {
		t.Parallel()
		// Apply production changes to guestbook deployment (matches "*-deployment.yaml" in guestbook dir)
		results, err := applier.ApplyChanges("manifests/guestbook/backend-deployment.yaml", []string{"production"})
		if err != nil {
			t.Fatalf("Failed to apply production changes: %v", err)
		}

		// Should have at least one change for production tag
		found := false
		for _, result := range results {
			if result.Change.Tag == "production" {
				found = true
				t.Logf("Production change applied: %s", result.String())
			}
		}
		if !found {
			t.Error("Expected to find production-tagged changes")
		}

		// Apply staging changes
		results, err = applier.ApplyChanges("manifests/guestbook/backend-deployment.yaml", []string{"staging"})
		if err != nil {
			t.Fatalf("Failed to apply staging changes: %v", err)
		}

		// Should have at least one change for staging tag
		found = false
		for _, result := range results {
			if result.Change.Tag == "staging" {
				found = true
				t.Logf("Staging change applied: %s", result.String())
			}
		}
		if !found {
			t.Error("Expected to find staging-tagged changes")
		}
	})

	// Test applying untagged changes
	t.Run("ApplyUntaggedChanges", func(t *testing.T) {
		t.Parallel()
		results, err := applier.ApplyChanges("manifests/guestbook/frontend-deployment.yaml", []string{})
		if err != nil {
			t.Fatalf("Failed to apply untagged changes: %v", err)
		}

		// Should have untagged changes (image updates)
		found := false
		for _, result := range results {
			if result.Change.Tag == "" {
				found = true
				t.Logf("Untagged change applied: %s", result.String())
			}
		}
		if !found {
			t.Error("Expected to find untagged changes")
		}
	})
}

// TestEvalContextWithRealConfig tests evaluation context with real configuration data.
func TestEvalContextWithRealConfig(t *testing.T) {
	t.Parallel()
	// Load the guestbook configuration
	projectRoot := getProjectRoot(t)
	guestbookDir := filepath.Join(projectRoot, "examples", "guestbook")

	cfg, err := config.LoadFromDirectory(guestbookDir)
	if err != nil {
		t.Fatalf("Failed to load guestbook configuration: %v", err)
	}

	// Create evaluation context
	ctx := NewEvalContext(
		guestbookDir,
		filepath.Join(guestbookDir, "scripts"),
		filepath.Join(guestbookDir, "files"),
		cfg.Functions,
	)

	// Test that functions are available
	if len(ctx.Functions) == 0 {
		t.Error("Expected functions to be loaded from configuration")
	}

	// Test function lookup
	var foundFunc *config.FunctionDefinition
	for _, fn := range ctx.Functions {
		if fn.Name == "get-replicas" {
			foundFunc = &fn
			break
		}
	}
	if foundFunc == nil {
		t.Error("Expected to find get-replicas function")
	} else {
		t.Logf("Found function: %s with %d parameters", foundFunc.Name, len(foundFunc.Params))
	}

	// Test variable management
	ctx.SetVariable("test", "value")
	if val, exists := ctx.GetVariable("test"); !exists || val != "value" {
		t.Error("Variable management not working correctly")
	}

	// Test context immutability
	newCtx := ctx.WithVariables(map[string]string{"new": "variable"})
	if _, exists := ctx.GetVariable("new"); exists {
		t.Error("Original context was modified")
	}
	if val, exists := newCtx.GetVariable("new"); !exists || val != "variable" {
		t.Error("New context doesn't have new variable")
	}
}

// loadYAMLDocument loads a YAML document from a file for testing.
func loadYAMLDocument(t *testing.T, filePath string) (*yaml.Node, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	err = yaml.Unmarshal(content, &doc)
	if err != nil {
		return nil, err
	}

	// Return the first document if it's a document node, otherwise return the root
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0], nil
	}
	return &doc, nil
}

// getProjectRoot finds the project root directory for testing.
func getProjectRoot(t *testing.T) string {
	// Start from the current working directory and walk up to find the project root
	cwd, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Look for go.mod file to identify project root
	dir := cwd
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := filepath.Glob(goModPath); err == nil {
			if matches, _ := filepath.Glob(goModPath); len(matches) > 0 {
				return dir
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	// Fallback: assume we're in internal/changes and go up three levels
	return filepath.Join(cwd, "..", "..", "..")
}
