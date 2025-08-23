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
	"github.com/zostay/genifest/internal/keysel"
)

var (
	runIncludeTags, runExcludeTags []string
)

var runCmd = &cobra.Command{
	Use:   "run [directory]",
	Short: "Generate Kubernetes manifests from templates",
	Long: `Generate Kubernetes manifests from templates by applying changes from configuration files.
This command processes the genifest.yaml configuration and applies dynamic value 
substitution to your Kubernetes resources.

If a directory is specified, the command will operate from that directory instead 
of the current working directory.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		var projectDir string
		if len(args) > 0 {
			projectDir = args[0]
		}
		err := GenerateManifests(nil, []string{projectDir})
		if err != nil {
			printError(err)
		}
	},
}

func init() {
	runCmd.Flags().StringSliceVarP(&runIncludeTags, "include-tags", "i", []string{}, "include only changes with these tags (supports glob patterns)")
	runCmd.Flags().StringSliceVarP(&runExcludeTags, "exclude-tags", "x", []string{}, "exclude changes with these tags (supports glob patterns)")
	rootCmd.AddCommand(runCmd)
}

func GenerateManifests(_ *cobra.Command, args []string) error {
	// Use the run command's tag flags if they exist, otherwise use root flags
	var currentIncludeTags, currentExcludeTags []string
	if len(runIncludeTags) > 0 || len(runExcludeTags) > 0 {
		currentIncludeTags = runIncludeTags
		currentExcludeTags = runExcludeTags
	} else {
		currentIncludeTags = includeTags
		currentExcludeTags = excludeTags
	}

	// Load project configuration
	var projectDir string
	if len(args) > 0 {
		projectDir = args[0]
	}
	projectInfo, err := loadProjectConfiguration(projectDir)
	if err != nil {
		return err
	}

	workDir := projectInfo.WorkDir
	cfg := projectInfo.Config

	// Determine tags to process
	tagsToProcess := determineTagsWithFlags(cfg, currentIncludeTags, currentExcludeTags)

	// Count total changes and changes that will be processed
	totalChanges := len(cfg.Changes)
	changesToRun := countChangesToRun(cfg, tagsToProcess)

	// Display initial summary
	fmt.Printf("🔍 \033[1;34mConfiguration loaded:\033[0m\n")
	fmt.Printf("  \033[36m•\033[0m \033[1m%d\033[0m total change definition(s) found\n", totalChanges)
	if len(currentIncludeTags) > 0 || len(currentExcludeTags) > 0 {
		fmt.Printf("  \033[36m•\033[0m \033[1m%d\033[0m change definition(s) match tag filter\n", changesToRun)
		if len(tagsToProcess) > 0 {
			fmt.Printf("  \033[36m•\033[0m Tags to process: \033[35m%v\033[0m\n", tagsToProcess)
		}
	} else {
		fmt.Printf("  \033[36m•\033[0m \033[1m%d\033[0m change definition(s) will be processed (all changes)\n", changesToRun)
	}
	fmt.Printf("  \033[36m•\033[0m \033[1m%d\033[0m file(s) to examine\n", len(cfg.Files))
	fmt.Printf("\n")

	if changesToRun == 0 {
		fmt.Printf("✅ \033[33mNo changes to apply based on current tag filters\033[0m\n")
		return nil
	}

	// Create applier
	applier := changes.NewApplier(cfg)

	// Find and process all managed files
	processedChanges, err := processAllFilesWithCounting(applier, cfg, tagsToProcess, workDir)
	if err != nil {
		return fmt.Errorf("failed to process files: %w", err)
	}

	// Final summary
	fmt.Printf("\n✅ \033[1;32mSuccessfully completed processing:\033[0m\n")
	fmt.Printf("  \033[32m•\033[0m \033[36m%d\033[0m change application(s) processed\n", processedChanges.Applied)
	fmt.Printf("  \033[32m•\033[0m \033[36m%d\033[0m change application(s) resulted in actual modifications\n", processedChanges.Modified)
	fmt.Printf("  \033[32m•\033[0m \033[36m%d\033[0m file(s) were updated\n", processedChanges.FilesModified)
	return nil
}

// determineTagsWithFlags determines which tags should be processed based on specific include/exclude flags.
func determineTagsWithFlags(cfg *config.Config, includeTagFlags, excludeTagFlags []string) []string {
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
	if len(includeTagFlags) == 0 && len(excludeTagFlags) == 0 {
		return append(availableTags, "") // Include untagged changes
	}

	var result []string

	// Add untagged changes unless specifically excluded
	includeUntagged := true
	if len(excludeTagFlags) > 0 {
		for _, excludePattern := range excludeTagFlags {
			if matchesGlob(excludePattern, "") {
				includeUntagged = false
				break
			}
		}
	}
	if len(includeTagFlags) > 0 {
		includeUntagged = false
		for _, includePattern := range includeTagFlags {
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
		if len(includeTagFlags) > 0 {
			include = false
			for _, includePattern := range includeTagFlags {
				if matchesGlob(includePattern, tag) {
					include = true
					break
				}
			}
		}

		// Check exclude patterns
		if include && len(excludeTagFlags) > 0 {
			for _, excludePattern := range excludeTagFlags {
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

// setValueInDocument sets a value in a YAML document using a key selector.
func setValueInDocument(doc *yaml.Node, keySelector, value string) (bool, error) {
	// Parse the selector using keysel
	parser, err := keysel.NewParser()
	if err != nil {
		return false, fmt.Errorf("failed to create keysel parser: %w", err)
	}

	expression, err := parser.ParseSelector(keySelector)
	if err != nil {
		return false, fmt.Errorf("failed to parse selector %q: %w", keySelector, err)
	}

	// Try to handle as a simple path first for backwards compatibility
	components, err := expression.GetSimplePath()
	if err != nil {
		// Complex expression - use evaluation-based write approach
		return setValueInDocumentComplex(doc, expression, value)
	}

	current := doc

	// If we start with a document node, navigate to its content
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}

	// If empty selector (root), we can't set a value
	if len(components) == 0 {
		return false, fmt.Errorf("cannot set value at root")
	}

	// Navigate to the parent of the target location
	for i, component := range components[:len(components)-1] {
		var navigateErr error
		current, navigateErr = navigateToComponent(current, component)
		if navigateErr != nil {
			return false, fmt.Errorf("navigation failed at component %d: %w", i, navigateErr)
		}
	}

	// Handle the final component for setting the value
	finalComponent := components[len(components)-1]
	return setValueAtComponent(current, finalComponent, value)
}

// setValueInDocumentComplex handles complex expressions with array iteration and functions.
func setValueInDocumentComplex(doc *yaml.Node, expression *keysel.Expression, newValue string) (bool, error) {
	// Create an evaluator
	evaluator := keysel.NewEvaluator()
	// Find the target node using the complex expression
	targetNode, err := expression.Evaluate(doc, evaluator)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate complex expression: %w", err)
	}

	if targetNode == nil {
		return false, fmt.Errorf("complex expression did not find a target node")
	}

	// Modify the target node directly
	originalValue := targetNode.Value
	targetNode.Value = newValue
	targetNode.Kind = yaml.ScalarNode

	// Return whether the value actually changed
	return originalValue != newValue, nil
}

// navigateToComponent navigates to a component using keysel logic.
func navigateToComponent(node *yaml.Node, component *keysel.Component) (*yaml.Node, error) {
	switch {
	case component.Field != nil:
		return navigateToField(node, component.Field.Name)
	case component.Bracket != nil:
		return navigateToBracket(node, component.Bracket.Content)
	default:
		return nil, fmt.Errorf("unknown component type")
	}
}

// navigateToField navigates to a field in a mapping node.
func navigateToField(node *yaml.Node, fieldName string) (*yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("cannot access field %q from non-mapping node", fieldName)
	}

	for i := 0; i < len(node.Content); i += 2 {
		if i+1 < len(node.Content) && node.Content[i].Value == fieldName {
			return node.Content[i+1], nil
		}
	}

	return nil, fmt.Errorf("field %q not found", fieldName)
}

// navigateToBracket navigates using bracket notation (index or key).
func navigateToBracket(node *yaml.Node, content string) (*yaml.Node, error) {
	// Check if it's a slice operation (contains colon)
	if strings.Contains(content, ":") {
		return nil, fmt.Errorf("slice operations not supported for value setting")
	}

	// Try numeric index first
	if _, err := fmt.Sscanf(content, "%d", new(int)); err == nil {
		var index int
		_, _ = fmt.Sscanf(content, "%d", &index)

		if node.Kind == yaml.SequenceNode {
			if index < 0 {
				index = len(node.Content) + index
			}
			if index < 0 || index >= len(node.Content) {
				return nil, fmt.Errorf("array index %d out of bounds (length %d)", index, len(node.Content))
			}
			return node.Content[index], nil
		}
		return nil, fmt.Errorf("cannot index non-sequence node with numeric index %d", index)
	}

	// Handle string key indexing
	key := content
	// Remove quotes if present (they would have been removed by participle unquoting)

	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 < len(node.Content) && node.Content[i].Value == key {
				return node.Content[i+1], nil
			}
		}
		return nil, fmt.Errorf("key %q not found", key)
	}
	return nil, fmt.Errorf("cannot index non-mapping node with string key %q", key)
}

// setValueAtComponent sets a value at the final component location.
func setValueAtComponent(node *yaml.Node, component *keysel.Component, value string) (bool, error) {
	switch {
	case component.Field != nil:
		return setValueAtField(node, component.Field.Name, value)
	case component.Bracket != nil:
		return setValueAtBracket(node, component.Bracket.Content, value)
	default:
		return false, fmt.Errorf("unknown component type for value setting")
	}
}

// setValueAtField sets a value at a field in a mapping node.
func setValueAtField(node *yaml.Node, fieldName string, value string) (bool, error) {
	if node.Kind != yaml.MappingNode {
		return false, fmt.Errorf("cannot set field %q in non-mapping node", fieldName)
	}

	for i := 0; i < len(node.Content); i += 2 {
		if i+1 < len(node.Content) && node.Content[i].Value == fieldName {
			node.Content[i+1].Value = value
			node.Content[i+1].Kind = yaml.ScalarNode
			return true, nil
		}
	}

	return false, fmt.Errorf("field %q not found", fieldName)
}

// setValueAtBracket sets a value using bracket notation.
func setValueAtBracket(node *yaml.Node, content string, value string) (bool, error) {
	// Check if it's a slice operation (contains colon)
	if strings.Contains(content, ":") {
		return false, fmt.Errorf("slice operations not supported for value setting")
	}

	// Try numeric index first
	if _, err := fmt.Sscanf(content, "%d", new(int)); err == nil {
		var index int
		_, _ = fmt.Sscanf(content, "%d", &index)

		if node.Kind == yaml.SequenceNode {
			if index < 0 {
				index = len(node.Content) + index
			}
			if index < 0 || index >= len(node.Content) {
				return false, fmt.Errorf("array index %d out of bounds (length %d)", index, len(node.Content))
			}
			node.Content[index].Value = value
			node.Content[index].Kind = yaml.ScalarNode
			return true, nil
		}
		return false, fmt.Errorf("cannot index non-sequence node with numeric index %d", index)
	}

	// Handle string key indexing
	key := content
	// The key would have been unquoted by participle already

	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 < len(node.Content) && node.Content[i].Value == key {
				node.Content[i+1].Value = value
				node.Content[i+1].Kind = yaml.ScalarNode
				return true, nil
			}
		}
		return false, fmt.Errorf("key %q not found", key)
	}
	return false, fmt.Errorf("cannot index non-mapping node with string key %q", key)
}

// writeYAMLFile writes YAML documents to a file.
func writeYAMLFile(filePath string, documents []yaml.Node, mode os.FileMode) error {
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

	return os.WriteFile(filePath, []byte(output.String()), mode)
}

// ProcessingStats tracks statistics during the run.
type ProcessingStats struct {
	Applied       int // Total number of changes applied
	Modified      int // Number of changes that actually modified values
	FilesModified int // Number of files that were modified
}

// countChangesToRun counts how many changes will be processed based on tags.
func countChangesToRun(cfg *config.Config, tagsToProcess []string) int {
	if len(tagsToProcess) == 0 {
		return 0
	}

	tagSet := make(map[string]bool)
	for _, tag := range tagsToProcess {
		tagSet[tag] = true
	}

	count := 0
	for _, change := range cfg.Changes {
		changeTag := change.Tag
		if changeTag == "" {
			changeTag = "" // Normalize empty tag
		}
		if tagSet[changeTag] {
			count++
		}
	}
	return count
}

// processAllFilesWithCounting processes files and tracks statistics.
func processAllFilesWithCounting(applier *changes.Applier, cfg *config.Config, tagsToProcess []string, workDir string) (*ProcessingStats, error) {
	stats := &ProcessingStats{}

	// Collect all files from the configuration
	filesToProcess := make([]string, 0, len(cfg.Files))
	filesToProcess = append(filesToProcess, cfg.Files...)

	// Process each file
	for _, filePath := range filesToProcess {
		fileStats, err := processFileWithCounting(applier, filePath, tagsToProcess, workDir)
		if err != nil {
			return stats, fmt.Errorf("failed to process file %s: %w", filePath, err)
		}

		stats.Applied += fileStats.Applied
		stats.Modified += fileStats.Modified
		if fileStats.Modified > 0 {
			stats.FilesModified++
		}
	}

	return stats, nil
}

// processFileWithCounting processes a file and tracks statistics.
func processFileWithCounting(applier *changes.Applier, filePath string, tagsToProcess []string, workDir string) (*ProcessingStats, error) {
	stats := &ProcessingStats{}

	// Get absolute path from working directory
	fullPath := filepath.Join(workDir, filePath)

	// Check if file exists
	var fi os.FileInfo
	var err error
	if fi, err = os.Stat(fullPath); os.IsNotExist(err) {
		fmt.Printf("⚠️  \033[33mWarning:\033[0m file %s does not exist, skipping\n", filePath)
		return stats, nil
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return stats, fmt.Errorf("failed to read file: %w", err)
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
			return stats, fmt.Errorf("failed to parse YAML: %w", err)
		}
		documents = append(documents, doc)
	}

	if len(documents) == 0 {
		fmt.Printf("⚠️  \033[33mWarning:\033[0m no YAML documents found in %s, skipping\n", filePath)
		return stats, nil
	}

	// Track if any changes were made to the file
	fileModified := false

	// Process each document
	for docIndex := range documents {
		doc := &documents[docIndex]

		// Apply changes to this document
		docStats, err := applyChangesToDocumentWithCounting(applier, filePath, doc, tagsToProcess, docIndex)
		if err != nil {
			return stats, fmt.Errorf("failed to apply changes to document %d: %w", docIndex, err)
		}

		stats.Applied += docStats.Applied
		stats.Modified += docStats.Modified
		if docStats.Modified > 0 {
			fileModified = true
		}
	}

	// Write back if modified
	if fileModified {
		err := writeYAMLFile(fullPath, documents, fi.Mode())
		if err != nil {
			return stats, fmt.Errorf("failed to write modified file: %w", err)
		}
		fmt.Printf("📝 \033[1;32mUpdated file:\033[0m %s (\033[36m%d\033[0m changes)\n", filePath, stats.Modified)
	}

	return stats, nil
}

// applyChangesToDocumentWithCounting applies changes to a document and tracks statistics.
func applyChangesToDocumentWithCounting(applier *changes.Applier, filePath string, doc *yaml.Node, tagsToProcess []string, docIndex int) (*ProcessingStats, error) {
	stats := &ProcessingStats{}

	// Apply all changes that match the tags at once
	results, err := applier.ApplyChanges(filePath, tagsToProcess)
	if err != nil {
		return stats, fmt.Errorf("failed to apply changes: %w", err)
	}

	// Apply each change result to the document
	for _, result := range results {
		changed, oldValue, err := applyChangeToDocumentWithOldValue(doc, result, applier, filePath)
		if err != nil {
			return stats, fmt.Errorf("failed to apply change %s: %w", result.KeyPath, err)
		}
		if changed {
			stats.Applied++
			if oldValue != result.Value {
				stats.Modified++
				fmt.Printf("  ✏️  %s -> document[\033[35m%d\033[0m] -> \033[36m%s\033[0m: \033[31m%s\033[0m → \033[32m%s\033[0m\n", filePath, docIndex, result.KeyPath, oldValue, result.Value)
			} else {
				fmt.Printf("  ✓  %s -> document[\033[35m%d\033[0m] -> \033[36m%s\033[0m: \033[37m%s\033[0m (no change)\n", filePath, docIndex, result.KeyPath, result.Value)
			}
		}
	}

	return stats, nil
}

// applyChangeToDocumentWithOldValue applies a single change to a YAML document and returns the old value.
func applyChangeToDocumentWithOldValue(doc *yaml.Node, result changes.ChangeResult, applier *changes.Applier, filePath string) (bool, string, error) {
	// First, get the old value
	oldValue, err := getValueInDocument(doc, result.Change.KeySelector)
	if err != nil {
		oldValue = "<not found>" // Value doesn't exist yet
	}

	// Always evaluate the value in the context of this document
	// Create evaluation context with this document
	evalCtx := applier.GetEvalContext().WithFile(filePath).WithDocument(doc)

	// Evaluate the ValueFrom in the context of this document
	value, err := evalCtx.Evaluate(result.Change.ValueFrom)
	if err != nil {
		return false, oldValue, fmt.Errorf("failed to evaluate change value: %w", err)
	}

	// Apply the change using the key selector
	changed, err := setValueInDocument(doc, result.Change.KeySelector, value)
	return changed, oldValue, err
}

// getValueInDocument gets a value from a YAML document using a key selector.
func getValueInDocument(doc *yaml.Node, keySelector string) (string, error) {
	// Remove leading dot if present
	keySelector = strings.TrimPrefix(keySelector, ".")

	// Split the selector into parts
	parts := strings.Split(keySelector, ".")

	current := doc

	// If we start with a document node, navigate to its content
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}

	// Navigate to the target field
	for i, part := range parts {
		if part == "" {
			continue
		}

		// Handle array indexing like "ports[0]"
		if bracketStart := strings.Index(part, "["); bracketStart >= 0 {
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
					return "", fmt.Errorf("key %q not found at path part %d", fieldName, i)
				}
			} else {
				return "", fmt.Errorf("cannot navigate to %q from non-mapping node", fieldName)
			}

			// Then handle the array index
			if strings.HasPrefix(indexPart, "[") && strings.HasSuffix(indexPart, "]") {
				indexStr := strings.Trim(indexPart, "[]")
				var index int
				if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
					return "", fmt.Errorf("invalid array index %q", indexStr)
				}

				if current.Kind == yaml.SequenceNode {
					if index < 0 || index >= len(current.Content) {
						return "", fmt.Errorf("array index %d out of bounds", index)
					}
					current = current.Content[index]
				} else {
					return "", fmt.Errorf("cannot index non-array node")
				}
			}
			continue
		}

		// Handle plain array index like "[0]"
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			indexStr := strings.Trim(part, "[]")
			var index int
			if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
				return "", fmt.Errorf("invalid array index %q", indexStr)
			}

			if current.Kind == yaml.SequenceNode {
				if index < 0 || index >= len(current.Content) {
					return "", fmt.Errorf("array index %d out of bounds", index)
				}
				current = current.Content[index]
			} else {
				return "", fmt.Errorf("cannot index non-array node")
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
				return "", fmt.Errorf("key %q not found at path part %d", part, i)
			}
		} else {
			return "", fmt.Errorf("cannot navigate to %q from non-mapping node", part)
		}
	}

	// Return the value
	return current.Value, nil
}
