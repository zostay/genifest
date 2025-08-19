package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var tagsCmd = &cobra.Command{
	Use:   "tags [directory]",
	Short: "List all tags in the configuration",
	Long: `List all tags found in the loaded configuration files.
This command will scan all changes in the configuration and display 
the unique tags that can be used with the --include-tags and --exclude-tags 
options in the run command.

If a directory is specified, the command will operate from that directory instead 
of the current working directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		var projectDir string
		if len(args) > 0 {
			projectDir = args[0]
		}
		return listTags(projectDir)
	},
}

func init() {
	rootCmd.AddCommand(tagsCmd)
}

func listTags(projectDir string) error {
	// Load project configuration
	projectInfo, err := loadProjectConfiguration(projectDir)
	if err != nil {
		return err
	}

	cfg := projectInfo.Config

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