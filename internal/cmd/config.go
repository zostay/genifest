package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config [directory]",
	Short: "Display the merged configuration",
	Long: `Display the merged configuration in YAML format after loading all configuration files.
This command loads the genifest.yaml configuration and shows the complete merged 
configuration including all metadata, files, changes, and functions that genifest 
knows about for the project.

This is useful for debugging configuration issues and understanding how genifest 
interprets your project structure.

If a directory is specified, the command will operate from that directory instead 
of the current working directory.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		var projectDir string
		if len(args) > 0 {
			projectDir = args[0]
		}
		err := displayConfiguration(projectDir)
		if err != nil {
			printError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func displayConfiguration(projectDir string) error {
	// Load project configuration using shared utility
	projectInfo, err := loadProjectConfiguration(projectDir)
	if err != nil {
		return err
	}

	cfg := projectInfo.Config
	workDir := projectInfo.WorkDir

	// Display header information
	fmt.Printf("# Merged configuration for: %s\n", workDir)
	fmt.Printf("# This shows the complete configuration as understood by genifest\n")
	fmt.Printf("# after loading all configuration files and applying metadata-driven discovery\n\n")

	// Convert the configuration to YAML and output it
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)

	err = encoder.Encode(cfg)
	if err != nil {
		return fmt.Errorf("failed to encode configuration as YAML: %w", err)
	}

	err = encoder.Close()
	if err != nil {
		return fmt.Errorf("failed to close YAML encoder: %w", err)
	}

	return nil
}
