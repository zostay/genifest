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
	"github.com/zostay/genifest/internal/fileformat"
	"github.com/zostay/genifest/internal/keysel"
	"github.com/zostay/genifest/internal/output"
)

var (
	runAdditionalTags []string
)

var runCmd = &cobra.Command{
	Use:   "run [group] [directory]",
	Short: "Generate Kubernetes manifests from templates",
	Long: `Generate Kubernetes manifests from templates by applying changes from configuration files.
This command processes the genifest.yaml configuration and applies dynamic value 
substitution to your Kubernetes resources.

Arguments:
- No arguments: Uses the "all" group in the current directory
- One argument: Uses the specified group name in the current directory, OR if it's a path, uses the "all" group in that directory
- Two arguments: Uses the specified group name in the specified directory

The --tag option allows adding additional tag expressions to the selected group.`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Determine output mode from flags
		outputMode := parseOutputMode(cmd)

		// Create output writer
		writer := output.NewWriter(outputMode, os.Stdout)

		groupName, projectDir := parseRunArguments(args)
		err := GenerateManifestsWithOutput(groupName, projectDir, runAdditionalTags, writer)
		if err != nil {
			printErrorWithOutput(err, writer)
		}
	},
}

func init() {
	runCmd.Flags().StringSliceVar(&runAdditionalTags, "tag", []string{}, "additional tag expressions to include (supports wildcards, negations, and directory scoping)")
	runCmd.Flags().String("output", "auto", "Output mode: color, plain, markdown, or auto (auto detects TTY)")
	rootCmd.AddCommand(runCmd)
}

// parseRunArguments parses the command line arguments to determine the group name and project directory.
func parseRunArguments(args []string) (groupName, projectDir string) {
	switch len(args) {
	case 0:
		// No arguments: use "all" group in current directory
		return "all", ""
	case 1:
		// One argument: could be either group name or directory
		arg := args[0]

		// If it looks like a directory (contains path separators or exists as a directory), treat as directory
		if strings.Contains(arg, "/") || strings.Contains(arg, "\\") {
			return "all", arg
		}

		// Check if it exists as a directory
		if info, err := os.Stat(arg); err == nil && info.IsDir() {
			return "all", arg
		}

		// Otherwise treat as group name
		return arg, ""
	case 2:
		// Two arguments: group name and directory
		return args[0], args[1]
	default:
		// This should not happen due to cobra.MaximumNArgs(2)
		return "all", ""
	}
}

func GenerateManifests(groupName, projectDir string, additionalTags []string) error {
	// Use default output mode for backwards compatibility
	writer := output.NewWriter(output.DetectDefaultMode(), os.Stdout)
	return GenerateManifestsWithOutput(groupName, projectDir, additionalTags, writer)
}

func GenerateManifestsWithOutput(groupName, projectDir string, additionalTags []string, writer output.Writer) error {
	// Load project configuration
	projectInfo, err := loadProjectConfiguration(projectDir)
	if err != nil {
		return err
	}

	workDir := projectInfo.WorkDir
	cfg := projectInfo.Config

	// Ensure we have groups configuration, use defaults if not specified
	if cfg.Groups == nil {
		cfg.Groups = config.GetDefaultGroups()
	}

	// Get the tag expressions for the specified group
	groupExpressions, exists := cfg.Groups[groupName]
	if !exists {
		return fmt.Errorf("group '%s' is not defined in configuration", groupName)
	}

	// Add any additional tag expressions from --tag flags
	if len(additionalTags) > 0 {
		groupExpressions = append(groupExpressions, additionalTags...)
	}

	// Determine which tags should be processed based on the group expressions
	tagsToProcess := determineTagsFromGroup(cfg, groupExpressions, workDir)

	// Count total changes and changes that will be processed
	totalChanges := len(cfg.Changes)
	changesToRun := countChangesToRun(cfg, tagsToProcess)

	// Display initial summary
	writer.Header("Configuration loaded:")
	writer.Bullet("total change definition(s) found", totalChanges)
	writer.Bullet(fmt.Sprintf("change definition(s) match group '%s'", groupName), changesToRun)
	if len(tagsToProcess) > 0 {
		writer.Printf("  • Tags to process: %v\n", tagsToProcess)
	}

	// Get resolved files count for display
	resolvedFiles, err := cfg.Files.ResolveFiles(workDir)
	if err != nil {
		resolvedFiles = cfg.Files.Include // Fallback to include list
	}
	writer.Bullet("file(s) to examine", len(resolvedFiles))
	writer.Println()

	if changesToRun == 0 {
		writer.Success(fmt.Sprintf("No changes to apply for group '%s'", groupName))
		return nil
	}

	// Create applier
	applier := changes.NewApplier(cfg)

	// Find and process all managed files
	processedChanges, err := processAllFilesWithCountingAndOutput(applier, cfg, tagsToProcess, workDir, writer)
	if err != nil {
		return fmt.Errorf("failed to process files: %w", err)
	}

	// Final summary
	writer.Println()
	writer.Success("Successfully completed processing:")
	writer.Bullet("change application(s) processed", processedChanges.Applied)
	writer.Bullet("change application(s) resulted in actual modifications", processedChanges.Modified)
	writer.Bullet("file(s) were updated", processedChanges.FilesModified)
	return nil
}

// determineTagsFromGroup determines which tags should be processed based on group expressions.
func determineTagsFromGroup(cfg *config.Config, groupExpressions config.TagExpressions, workDir string) []string {
	// Collect all unique tags from changes
	allTags := make(map[string]bool)
	for _, change := range cfg.Changes {
		if change.Tag != "" {
			allTags[change.Tag] = true
		}
	}

	// Add empty tag for untagged changes
	allTags[""] = true

	var result []string
	for tag := range allTags {
		// Determine the directory for this change (for directory-scoped expressions)
		// For now, use workDir as the base directory
		// TODO: In actual implementation, we might need to track the directory per change
		changeDir := workDir

		if groupExpressions.MatchesTag(tag, changeDir) {
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

	// Check if any component uses bracket notation - if so, use complex path
	// to ensure proper quoted string handling
	for _, component := range components {
		if component.Bracket != nil {
			return setValueInDocumentComplex(doc, expression, value)
		}
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
	targetNode.Tag = "" // Clear the tag so YAML can infer the correct type

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

	// Check if this is a quoted string first
	isQuotedString := (strings.HasPrefix(content, "\"") && strings.HasSuffix(content, "\"")) ||
		(strings.HasPrefix(content, "'") && strings.HasSuffix(content, "'"))

	// Only try numeric parsing if it's not a quoted string
	if !isQuotedString {
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
	}

	// Handle string key indexing (remove quotes if present)
	key := content
	if isQuotedString {
		key = key[1 : len(key)-1]
	}

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
			node.Content[i+1].Tag = "" // Clear the tag so YAML can infer the correct type
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

	// Check if this is a quoted string first
	isQuotedString := (strings.HasPrefix(content, "\"") && strings.HasSuffix(content, "\"")) ||
		(strings.HasPrefix(content, "'") && strings.HasSuffix(content, "'"))

	// Only try numeric parsing if it's not a quoted string
	if !isQuotedString {
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
				node.Content[index].Tag = "" // Clear the tag so YAML can infer the correct type
				return true, nil
			}
			return false, fmt.Errorf("cannot index non-sequence node with numeric index %d", index)
		}
	}

	// Handle string key indexing (remove quotes if present)
	key := content
	if isQuotedString {
		key = key[1 : len(key)-1]
	}

	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 < len(node.Content) && node.Content[i].Value == key {
				node.Content[i+1].Value = value
				node.Content[i+1].Kind = yaml.ScalarNode
				node.Content[i+1].Tag = "" // Clear the tag so YAML can infer the correct type
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

// processAllFilesWithCountingAndOutput processes files and tracks statistics with output writer support.
func processAllFilesWithCountingAndOutput(applier *changes.Applier, cfg *config.Config, tagsToProcess []string, workDir string, writer output.Writer) (*ProcessingStats, error) {
	stats := &ProcessingStats{}

	// Resolve files with wildcard expansion
	filesToProcess, err := cfg.Files.ResolveFiles(workDir)
	if err != nil {
		return stats, fmt.Errorf("failed to resolve files: %w", err)
	}

	// Process each file
	for _, filePath := range filesToProcess {
		fileStats, err := processFileWithCountingAndOutput(applier, filePath, tagsToProcess, workDir, writer)
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

// processFileWithCountingAndOutput processes a file and tracks statistics with output writer support.
func processFileWithCountingAndOutput(applier *changes.Applier, filePath string, tagsToProcess []string, workDir string, writer output.Writer) (*ProcessingStats, error) {
	stats := &ProcessingStats{}

	// Get absolute path from working directory
	fullPath := filepath.Join(workDir, filePath)

	// Check if file exists
	var fi os.FileInfo
	var err error
	if fi, err = os.Stat(fullPath); os.IsNotExist(err) {
		writer.Warning(fmt.Sprintf("file %s does not exist, skipping", filePath))
		return stats, nil
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return stats, fmt.Errorf("failed to read file: %w", err)
	}

	// Detect file format
	format := fileformat.DetectFormat(filePath)
	if format == fileformat.FormatUnknown {
		writer.Warning(fmt.Sprintf("unknown file format for %s, skipping", filePath))
		return stats, nil
	}

	// Get parser for the format
	parser, err := fileformat.GetParser(format)
	if err != nil {
		return stats, fmt.Errorf("failed to get parser for format %s: %w", format, err)
	}

	// Parse documents
	documents, err := parser.Parse(content)
	if err != nil {
		return stats, fmt.Errorf("failed to parse %s file: %w", format, err)
	}

	if len(documents) == 0 {
		writer.Warning(fmt.Sprintf("no documents found in %s, skipping", filePath))
		return stats, nil
	}

	// Track if any changes were made to the file
	fileModified := false

	// Process each document
	for docIndex, doc := range documents {
		// Apply changes to this document using generic format
		docStats, err := applyChangesToGenericDocumentWithCountingAndOutput(applier, filePath, doc, tagsToProcess, docIndex, writer)
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
		serializedContent, err := parser.Serialize(documents)
		if err != nil {
			return stats, fmt.Errorf("failed to serialize modified file: %w", err)
		}

		err = os.WriteFile(fullPath, serializedContent, fi.Mode())
		if err != nil {
			return stats, fmt.Errorf("failed to write modified file: %w", err)
		}
		writer.UpdatedFile(filePath, stats.Modified)
	}

	return stats, nil
}

// applyChangesToDocumentWithCountingAndOutput applies changes to a document and tracks statistics with output writer support.
func applyChangesToDocumentWithCountingAndOutput(applier *changes.Applier, filePath string, doc *yaml.Node, tagsToProcess []string, docIndex int, writer output.Writer) (*ProcessingStats, error) {
	stats := &ProcessingStats{}

	// Apply all changes that match the tags at once
	results, err := applier.ApplyChanges(filePath, tagsToProcess)
	if err != nil {
		return stats, fmt.Errorf("failed to apply changes: %w", err)
	}

	// Apply each change result to the document
	for _, result := range results {
		// Check if document selector matches this document
		matches := matchesDocumentSelector(doc, result.Change.DocumentSelector)
		if !matches {
			continue
		}

		changed, oldValue, err := applyChangeToDocumentWithOldValue(doc, &result, applier, filePath)
		if err != nil {
			return stats, fmt.Errorf("failed to apply change %s: %w", result.KeyPath, err)
		}
		if changed {
			stats.Applied++
			if oldValue != result.Value {
				stats.Modified++
				writer.Change(filePath, result.KeyPath, oldValue, result.Value, docIndex)
			} else {
				writer.NoChange(filePath, result.KeyPath, result.Value, docIndex)
			}
		}
	}

	return stats, nil
}

// applyChangeToDocumentWithOldValue applies a single change to a YAML document and returns the old value.
func applyChangeToDocumentWithOldValue(doc *yaml.Node, result *changes.ChangeResult, applier *changes.Applier, filePath string) (bool, string, error) {
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

	// Update result with evaluated value for display purposes
	result.Value = value

	// Apply the change using the key selector
	changed, err := setValueInDocument(doc, result.Change.KeySelector, value)
	return changed, oldValue, err
}

// matchesDocumentSelector checks if a document matches the given document selector.
func matchesDocumentSelector(doc *yaml.Node, selector config.DocumentSelector) bool {
	// If no selector provided, match all documents
	if len(selector) == 0 {
		return true
	}

	// Start from the root content if this is a document node
	current := doc
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}

	// Check each selector key-value pair
	for selectorKey, expectedValue := range selector {
		actualValue, err := getDocumentValue(current, selectorKey)
		if err != nil || actualValue != expectedValue {
			return false
		}
	}

	return true
}

// getDocumentValue gets a value from a YAML document using a simple key path.
func getDocumentValue(doc *yaml.Node, keyPath string) (string, error) {
	// Split the key path by dots
	parts := strings.Split(keyPath, ".")
	current := doc

	// Navigate through the path
	for _, part := range parts {
		if part == "" {
			continue
		}

		if current.Kind != yaml.MappingNode {
			return "", fmt.Errorf("cannot navigate to %q from non-mapping node", part)
		}

		found := false
		for i := 0; i < len(current.Content); i += 2 {
			if i+1 < len(current.Content) && current.Content[i].Value == part {
				current = current.Content[i+1]
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("key %q not found", part)
		}
	}

	return current.Value, nil
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

// applyChangesToGenericDocumentWithCountingAndOutput applies changes to a generic document and tracks statistics with output writer support.
func applyChangesToGenericDocumentWithCountingAndOutput(applier *changes.Applier, filePath string, doc *fileformat.Node, tagsToProcess []string, docIndex int, writer output.Writer) (*ProcessingStats, error) {
	stats := &ProcessingStats{}

	// For now, convert the generic document to YAML for compatibility with the existing applier
	// TODO: In the future, update the entire changes system to work with generic AST
	yamlParser := &fileformat.YAMLParser{}
	yamlNodes, err := yamlParser.Serialize([]*fileformat.Node{doc})
	if err != nil {
		return stats, fmt.Errorf("failed to convert document to YAML for processing: %w", err)
	}

	// Parse back to yaml.Node
	var yamlDoc yaml.Node
	err = yaml.Unmarshal(yamlNodes, &yamlDoc)
	if err != nil {
		return stats, fmt.Errorf("failed to parse converted YAML: %w", err)
	}

	// Apply changes using existing YAML-based applier
	yamlStats, err := applyChangesToDocumentWithCountingAndOutput(applier, filePath, &yamlDoc, tagsToProcess, docIndex, writer)
	if err != nil {
		return stats, err
	}

	// Convert the modified YAML back to generic format and update the original document
	if yamlStats.Modified > 0 {
		yamlBytes, err := yaml.Marshal(&yamlDoc)
		if err != nil {
			return stats, fmt.Errorf("failed to marshal modified YAML: %w", err)
		}

		modifiedNodes, err := yamlParser.Parse(yamlBytes)
		if err != nil {
			return stats, fmt.Errorf("failed to parse modified YAML: %w", err)
		}

		if len(modifiedNodes) > 0 {
			// Copy the modified node's structure back to the original
			*doc = *modifiedNodes[0]
		}
	}

	return yamlStats, nil
}
