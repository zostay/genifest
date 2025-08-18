package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/zostay/genifest/internal/changes"
	"github.com/zostay/genifest/internal/config"
)

var (
	rootCmd = &cobra.Command{
		Use:   "genifest",
		Short: "Generate Kubernetes manifests from templates",
		Long: `genifest is a Kubernetes manifest generation tool that creates deployment 
manifests from templates for GitOps workflows. It processes configuration files 
to generate Kubernetes resources with dynamic value substitution.`,
		Args: cobra.NoArgs,
		RunE: GenerateManifests,
	}

	includeTags, excludeTags []string
)

func init() {
	rootCmd.Flags().StringSliceVarP(&includeTags, "include-tags", "i", []string{}, "include only changes with these tags (supports glob patterns)")
	rootCmd.Flags().StringSliceVarP(&excludeTags, "exclude-tags", "x", []string{}, "exclude changes with these tags (supports glob patterns)")
}

// Execute runs the root command.
func Execute() {
	// It is tempting to handle this error, but don't do it. Cobra already does
	// all the reporting necessary. Any additional reporting is simply redundant
	// and repetitive.
	_ = rootCmd.Execute()
}

func GenerateManifests(_ *cobra.Command, _ []string) error {
	// Find project root with genifest.yaml
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	configPath := filepath.Join(workDir, "genifest.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("genifest.yaml not found in current directory. Please run from project root")
	}

	// Load configuration
	cfg, err := config.LoadFromDirectory(workDir)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Determine tags to process
	tagsToProcess := determineTags(cfg)

	// Create applier
	applier := changes.NewApplier(cfg)

	// Find and process all managed files
	err = processAllFiles(applier, cfg, tagsToProcess)
	if err != nil {
		return fmt.Errorf("failed to process files: %w", err)
	}

	fmt.Printf("Successfully applied changes to manifests\n")
	return nil
}

// determineTags determines which tags should be processed based on include/exclude flags.
func determineTags(cfg *config.Config) []string {
	// Collect all unique tags from changes
	allTags := make(map[string]bool)
	for _, change := range cfg.Changes {
		if change.Tag != "" {
			allTags[change.Tag] = true
		}
	}

	// Convert to slice
	availableTags := make([]string, 0, len(allTags))
	for tag := range allTags {
		availableTags = append(availableTags, tag)
	}

	// If no include/exclude tags specified, return all tags plus empty tag
	if len(includeTags) == 0 && len(excludeTags) == 0 {
		return append(availableTags, "") // Include untagged changes
	}

	var result []string

	// Add untagged changes unless specifically excluded
	includeUntagged := true
	if len(excludeTags) > 0 {
		for _, excludePattern := range excludeTags {
			if matchesGlob(excludePattern, "") {
				includeUntagged = false
				break
			}
		}
	}
	if len(includeTags) > 0 {
		includeUntagged = false
		for _, includePattern := range includeTags {
			if matchesGlob(includePattern, "") {
				includeUntagged = true
				break
			}
		}
	}
	if includeUntagged {
		result = append(result, "")
	}

	// Process each available tag
	for _, tag := range availableTags {
		include := true

		// Check include patterns
		if len(includeTags) > 0 {
			include = false
			for _, includePattern := range includeTags {
				if matchesGlob(includePattern, tag) {
					include = true
					break
				}
			}
		}

		// Check exclude patterns
		if include && len(excludeTags) > 0 {
			for _, excludePattern := range excludeTags {
				if matchesGlob(excludePattern, tag) {
					include = false
					break
				}
			}
		}

		if include {
			result = append(result, tag)
		}
	}

	return result
}

// matchesGlob checks if a pattern matches a string using basic glob matching.
func matchesGlob(pattern, str string) bool {
	// Handle exact match
	if pattern == str {
		return true
	}

	// Handle wildcard patterns
	if pattern == "*" {
		return true
	}

	// Use filepath.Match for basic glob support
	matched, err := filepath.Match(pattern, str)
	if err != nil {
		// If pattern is invalid, fall back to exact match
		return pattern == str
	}
	return matched
}

// processAllFiles finds and processes all files that should have changes applied.
func processAllFiles(applier *changes.Applier, cfg *config.Config, tagsToProcess []string) error {
	// Collect all files from the configuration
	filesToProcess := make([]string, 0, len(cfg.Files))

	// Add files explicitly listed in the config
	filesToProcess = append(filesToProcess, cfg.Files...)

	// Process each file
	for _, filePath := range filesToProcess {
		err := processFile(applier, filePath, tagsToProcess)
		if err != nil {
			return fmt.Errorf("failed to process file %s: %w", filePath, err)
		}
	}

	return nil
}

// processFile reads a YAML file, applies changes, and writes it back.
func processFile(applier *changes.Applier, filePath string, tagsToProcess []string) error {
	// Get absolute path from working directory
	workDir, _ := os.Getwd()
	fullPath := filepath.Join(workDir, filePath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		fmt.Printf("Warning: file %s does not exist, skipping\n", filePath)
		return nil
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse YAML documents
	var documents []yaml.Node
	decoder := yaml.NewDecoder(strings.NewReader(string(content)))

	for {
		var doc yaml.Node
		err := decoder.Decode(&doc)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
		documents = append(documents, doc)
	}

	if len(documents) == 0 {
		fmt.Printf("Warning: no YAML documents found in %s, skipping\n", filePath)
		return nil
	}

	// Track if any changes were made
	modified := false

	// Process each document
	for docIndex := range documents {
		doc := &documents[docIndex]

		// Apply changes to this document
		changed, err := applyChangesToDocument(applier, filePath, doc, tagsToProcess)
		if err != nil {
			return fmt.Errorf("failed to apply changes to document %d: %w", docIndex, err)
		}

		if changed {
			modified = true
		}
	}

	// Write back if modified
	if modified {
		err := writeYAMLFile(fullPath, documents)
		if err != nil {
			return fmt.Errorf("failed to write modified file: %w", err)
		}
		fmt.Printf("Applied changes to %s\n", filePath)
	}

	return nil
}

// applyChangesToDocument applies changes to a single YAML document.
func applyChangesToDocument(applier *changes.Applier, filePath string, doc *yaml.Node, tagsToProcess []string) (bool, error) {
	modified := false

	// Apply all changes that match the tags at once
	results, err := applier.ApplyChanges(filePath, tagsToProcess)
	if err != nil {
		return false, fmt.Errorf("failed to apply changes: %w", err)
	}

	// Apply each change result to the document
	for _, result := range results {
		changed, err := applyChangeToDocument(doc, result, applier, filePath)
		if err != nil {
			return false, fmt.Errorf("failed to apply change %s: %w", result.KeyPath, err)
		}
		if changed {
			modified = true
			fmt.Printf("  Applied change: %s = %s\n", result.KeyPath, result.Value)
		}
	}

	return modified, nil
}

// applyChangeToDocument applies a single change to a YAML document.
func applyChangeToDocument(doc *yaml.Node, result changes.ChangeResult, applier *changes.Applier, filePath string) (bool, error) {
	// For document references, we need to evaluate the value in the context of this document
	if result.Change.KeySelector != "" {
		// Create evaluation context with this document
		evalCtx := applier.GetEvalContext().WithFile(filePath).WithDocument(doc)

		// Evaluate the ValueFrom in the context of this document
		value, err := evalCtx.Evaluate(result.Change.ValueFrom)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate change value: %w", err)
		}

		// Apply the change using the key selector
		return setValueInDocument(doc, result.Change.KeySelector, value)
	}

	// For other types of changes, use the pre-evaluated value
	return setValueInDocument(doc, result.KeyPath, result.Value)
}

// setValueInDocument sets a value in a YAML document using a key selector.
func setValueInDocument(doc *yaml.Node, keySelector, value string) (bool, error) {
	// Remove leading dot if present
	keySelector = strings.TrimPrefix(keySelector, ".")

	// Split the selector into parts
	parts := strings.Split(keySelector, ".")

	current := doc

	// If we start with a document node, navigate to its content
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}

	// Navigate to the parent of the target field
	for i, part := range parts[:len(parts)-1] {
		if part == "" {
			continue
		}

		// Handle array indexing like "ports[0]"
		if strings.Contains(part, "[") {
			bracketStart := strings.Index(part, "[")
			fieldName := part[:bracketStart]
			indexPart := part[bracketStart:]

			// First navigate to the field
			if current.Kind == yaml.MappingNode {
				found := false
				for j := 0; j < len(current.Content); j += 2 {
					if j+1 < len(current.Content) && current.Content[j].Value == fieldName {
						current = current.Content[j+1]
						found = true
						break
					}
				}
				if !found {
					return false, fmt.Errorf("key %q not found at path part %d", fieldName, i)
				}
			} else {
				return false, fmt.Errorf("cannot navigate to %q from non-mapping node", fieldName)
			}

			// Then handle the array index
			if strings.HasPrefix(indexPart, "[") && strings.HasSuffix(indexPart, "]") {
				indexStr := strings.Trim(indexPart, "[]")
				var index int
				if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
					return false, fmt.Errorf("invalid array index %q", indexStr)
				}

				if current.Kind == yaml.SequenceNode {
					if index < 0 || index >= len(current.Content) {
						return false, fmt.Errorf("array index %d out of bounds", index)
					}
					current = current.Content[index]
				} else {
					return false, fmt.Errorf("cannot index non-array node")
				}
			}
			continue
		}

		// Handle plain array index like "[0]"
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			indexStr := strings.Trim(part, "[]")
			var index int
			if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
				return false, fmt.Errorf("invalid array index %q", indexStr)
			}

			if current.Kind == yaml.SequenceNode {
				if index < 0 || index >= len(current.Content) {
					return false, fmt.Errorf("array index %d out of bounds", index)
				}
				current = current.Content[index]
			} else {
				return false, fmt.Errorf("cannot index non-array node")
			}
			continue
		}

		// Navigate through mapping nodes
		if current.Kind == yaml.MappingNode {
			found := false
			for j := 0; j < len(current.Content); j += 2 {
				if j+1 < len(current.Content) && current.Content[j].Value == part {
					current = current.Content[j+1]
					found = true
					break
				}
			}
			if !found {
				return false, fmt.Errorf("key %q not found at path part %d", part, i)
			}
		} else {
			return false, fmt.Errorf("cannot navigate to %q from non-mapping node", part)
		}
	}

	// Set the final value
	finalKey := parts[len(parts)-1]

	// Handle array indexing in final key
	if strings.Contains(finalKey, "[") {
		bracketStart := strings.Index(finalKey, "[")
		fieldName := finalKey[:bracketStart]
		indexPart := finalKey[bracketStart:]

		// Navigate to the field
		if current.Kind == yaml.MappingNode {
			found := false
			for j := 0; j < len(current.Content); j += 2 {
				if j+1 < len(current.Content) && current.Content[j].Value == fieldName {
					current = current.Content[j+1]
					found = true
					break
				}
			}
			if !found {
				return false, fmt.Errorf("key %q not found", fieldName)
			}
		} else {
			return false, fmt.Errorf("cannot navigate to %q from non-mapping node", fieldName)
		}

		// Handle the array index
		if strings.HasPrefix(indexPart, "[") && strings.HasSuffix(indexPart, "]") {
			indexStr := strings.Trim(indexPart, "[]")
			var index int
			if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
				return false, fmt.Errorf("invalid array index %q", indexStr)
			}

			if current.Kind == yaml.SequenceNode {
				if index < 0 || index >= len(current.Content) {
					return false, fmt.Errorf("array index %d out of bounds", index)
				}
				// Update the array element
				current.Content[index].Value = value
				current.Content[index].Kind = yaml.ScalarNode
				return true, nil
			} else {
				return false, fmt.Errorf("cannot index non-array node")
			}
		}
	}

	// Handle plain array index
	if strings.HasPrefix(finalKey, "[") && strings.HasSuffix(finalKey, "]") {
		indexStr := strings.Trim(finalKey, "[]")
		var index int
		if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
			return false, fmt.Errorf("invalid array index %q", indexStr)
		}

		if current.Kind == yaml.SequenceNode {
			if index < 0 || index >= len(current.Content) {
				return false, fmt.Errorf("array index %d out of bounds", index)
			}
			current.Content[index].Value = value
			current.Content[index].Kind = yaml.ScalarNode
			return true, nil
		} else {
			return false, fmt.Errorf("cannot index non-array node")
		}
	}

	// Set field in mapping node
	if current.Kind == yaml.MappingNode {
		for j := 0; j < len(current.Content); j += 2 {
			if j+1 < len(current.Content) && current.Content[j].Value == finalKey {
				current.Content[j+1].Value = value
				current.Content[j+1].Kind = yaml.ScalarNode
				return true, nil
			}
		}
		return false, fmt.Errorf("key %q not found", finalKey)
	}

	return false, fmt.Errorf("cannot set value in non-mapping node")
}

// writeYAMLFile writes YAML documents to a file.
func writeYAMLFile(filePath string, documents []yaml.Node) error {
	var output strings.Builder

	for i, doc := range documents {
		if i > 0 {
			output.WriteString("---\n")
		}

		encoder := yaml.NewEncoder(&output)
		encoder.SetIndent(2)
		err := encoder.Encode(&doc)
		if err != nil {
			return fmt.Errorf("failed to encode document %d: %w", i, err)
		}
		err = encoder.Close()
		if err != nil {
			return fmt.Errorf("failed to close encoder for document %d: %w", i, err)
		}
	}

	return os.WriteFile(filePath, []byte(output.String()), 0644)
}
