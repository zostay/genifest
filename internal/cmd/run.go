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
	fmt.Printf("🔍 Configuration loaded:\n")
	fmt.Printf("  • %d total change definition(s) found\n", totalChanges)
	if len(currentIncludeTags) > 0 || len(currentExcludeTags) > 0 {
		fmt.Printf("  • %d change definition(s) match tag filter\n", changesToRun)
		if len(tagsToProcess) > 0 {
			fmt.Printf("  • Tags to process: %v\n", tagsToProcess)
		}
	} else {
		fmt.Printf("  • %d change definition(s) will be processed (all changes)\n", changesToRun)
	}
	fmt.Printf("  • %d file(s) to examine\n", len(cfg.Files))
	fmt.Printf("\n")

	if changesToRun == 0 {
		fmt.Printf("✅ No changes to apply based on current tag filters\n")
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
	fmt.Printf("\n✅ Successfully completed processing:\n")
	fmt.Printf("  • %d change application(s) processed\n", processedChanges.Applied)
	fmt.Printf("  • %d change application(s) resulted in actual modifications\n", processedChanges.Modified)
	fmt.Printf("  • %d file(s) were updated\n", processedChanges.FilesModified)
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
	if bracketStart := strings.Index(finalKey, "["); bracketStart >= 0 {
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
		fmt.Printf("⚠️  Warning: file %s does not exist, skipping\n", filePath)
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
		fmt.Printf("⚠️  Warning: no YAML documents found in %s, skipping\n", filePath)
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
		fmt.Printf("📝 Updated file: %s (%d changes)\n", filePath, stats.Modified)
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
				fmt.Printf("  ✏️  %s -> document[%d] -> %s: %s → %s\n", filePath, docIndex, result.KeyPath, oldValue, result.Value)
			} else {
				fmt.Printf("  ✓  %s -> document[%d] -> %s: %s (no change)\n", filePath, docIndex, result.KeyPath, result.Value)
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

	// For document references, we need to evaluate the value in the context of this document
	if result.Change.KeySelector != "" {
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

	// For other types of changes, use the pre-evaluated value
	changed, err := setValueInDocument(doc, result.KeyPath, result.Value)
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
