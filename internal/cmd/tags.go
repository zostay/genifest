package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/zostay/genifest/internal/config"
)

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List all tags in the configuration",
	Long: `List all tags found in the loaded configuration files.
This command will scan all changes in the configuration and display 
the unique tags that can be used with the --include-tags and --exclude-tags 
options in the run command.`,
	Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		return listTags()
	},
}

func init() {
	rootCmd.AddCommand(tagsCmd)
}

func listTags() error {
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

	// Collect all unique tags from changes
	tagSet := make(map[string]bool)
	for _, change := range cfg.Changes {
		if change.Tag != "" {
			tagSet[change.Tag] = true
		}
	}

	// Convert to sorted slice
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	// Display results
	if len(tags) == 0 {
		fmt.Println("No tags found in configuration")
		return nil
	}

	fmt.Printf("Found %d tag(s) in configuration:\n", len(tags))
	for _, tag := range tags {
		fmt.Printf("  %s\n", tag)
	}

	// Also show if there are untagged changes
	hasUntaggedChanges := false
	for _, change := range cfg.Changes {
		if change.Tag == "" {
			hasUntaggedChanges = true
			break
		}
	}

	if hasUntaggedChanges {
		fmt.Println("\nNote: Configuration also contains untagged changes")
	}

	return nil
}