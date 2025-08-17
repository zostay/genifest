package changes

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zostay/genifest/internal/config"
)

// Applier applies changes to configurations using the evaluation system.
type Applier struct {
	config  *config.Config
	evalCtx *EvalContext
}

// NewApplier creates a new change applier for the given configuration.
func NewApplier(cfg *config.Config) *Applier {
	// Determine directories from metadata
	cloudHome := cfg.Metadata.CloudHome
	if cloudHome == "" {
		cloudHome = "."
	}

	// Build script and file directories from metadata
	var scriptsDir, filesDir string
	if len(cfg.Metadata.Scripts) > 0 {
		scriptsDir = filepath.Join(cloudHome, cfg.Metadata.Scripts[0].Path)
	}
	if len(cfg.Metadata.Files) > 0 {
		filesDir = filepath.Join(cloudHome, cfg.Metadata.Files[0].Path)
	}

	evalCtx := NewEvalContext(cloudHome, scriptsDir, filesDir, cfg.Functions)

	return &Applier{
		config:  cfg,
		evalCtx: evalCtx,
	}
}

// GetEvalContext returns the evaluation context for this applier.
func (a *Applier) GetEvalContext() *EvalContext {
	return a.evalCtx
}

// EvaluateChangeValue evaluates a change order's value in the context of a specific file.
func (a *Applier) EvaluateChangeValue(change config.ChangeOrder, filePath string) (string, error) {
	// Create context for this specific file
	ctx := a.evalCtx.WithFile(filePath)

	// Evaluate the change's ValueFrom
	return ctx.Evaluate(change.ValueFrom)
}

// ApplyChanges applies all matching changes to a file context
// This is a simplified version - a full implementation would need to:
// - Parse YAML files
// - Apply fileSelector and keySelector matching
// - Modify the YAML documents
// - Write back the results
// .
func (a *Applier) ApplyChanges(filePath string, tags []string) ([]ChangeResult, error) {
	var results []ChangeResult

	for _, change := range a.config.Changes {
		// Check if change applies based on tags
		if change.Tag != "" {
			found := false
			for _, tag := range tags {
				if tag == change.Tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check if file matches (simplified glob matching)
		if change.FileSelector != "" {
			matched := matchesGlobPattern(change.FileSelector, filePath)
			if !matched {
				continue
			}
		}

		// Evaluate the change value
		value, err := a.EvaluateChangeValue(change, filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate change for file %s: %w", filePath, err)
		}

		results = append(results, ChangeResult{
			Change:   change,
			FilePath: filePath,
			Value:    value,
			KeyPath:  change.KeySelector,
		})
	}

	return results, nil
}

// ChangeResult represents the result of applying a change.
type ChangeResult struct {
	Change   config.ChangeOrder
	FilePath string
	Value    string
	KeyPath  string
}

// String returns a string representation of the change result.
func (cr ChangeResult) String() string {
	return fmt.Sprintf("File: %s, Key: %s, Value: %s", cr.FilePath, cr.KeyPath, cr.Value)
}

// matchesGlobPattern provides simplified glob pattern matching
// This is a basic implementation - a full implementation would use a proper glob library.
func matchesGlobPattern(pattern, path string) bool {
	// Handle simple cases first
	if pattern == "" || pattern == "*" {
		return true
	}

	// Split pattern and path into segments
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	// If lengths don't match and there are no wildcards, no match
	if len(patternParts) != len(pathParts) {
		// Check if pattern has wildcards that could account for the difference
		hasWildcard := false
		for _, part := range patternParts {
			if strings.Contains(part, "*") {
				hasWildcard = true
				break
			}
		}
		if !hasWildcard {
			return false
		}
	}

	// For this simplified implementation, let's handle the specific patterns we need
	// Pattern: "manifests/*/deployment.yaml" should match "manifests/guestbook/deployment.yaml"
	// but also "manifests/postgres/deployment.yaml"

	if len(patternParts) == len(pathParts) {
		for i, patternPart := range patternParts {
			if patternPart == "*" {
				continue // Wildcard matches anything
			}

			// Use filepath.Match for individual segments
			matched, err := filepath.Match(patternPart, pathParts[i])
			if err != nil || !matched {
				return false
			}
		}
		return true
	}

	return false
}
