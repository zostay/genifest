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
	Run: func(_ *cobra.Command, args []string) {
		var projectDir string
		if len(args) > 0 {
			projectDir = args[0]
		}
		err := listTags(projectDir)
		if err != nil {
			printError(err)
		}
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
		fmt.Println("🏷️  \033[33mNo tags found in configuration\033[0m")
		return nil
	}

	fmt.Printf("🏷️  \033[1;34mFound %d tag(s) in configuration:\033[0m\n", len(tags))
	for _, tag := range tags {
		fmt.Printf("  \033[36m•\033[0m \033[1m%s\033[0m\n", tag)
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
		fmt.Println("\n💡 \033[33mNote:\033[0m Configuration also contains untagged changes")
	}

	return nil
}
