package cmd

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed version.txt
var Version string

var (
	rootCmd = &cobra.Command{
		Use:   "genifest",
		Short: "Generate Kubernetes manifests from templates",
		Long: `genifest is a Kubernetes manifest generation tool that creates deployment 
manifests from templates for GitOps workflows. It processes configuration files 
to generate Kubernetes resources with dynamic value substitution.`,
		Args: cobra.NoArgs,
	}

	includeTags, excludeTags []string
)

func init() {
	// Legacy flags for backward compatibility - these are not used anymore
	// as the functionality has moved to the 'run' subcommand
	rootCmd.Flags().StringSliceVarP(&includeTags, "include-tags", "i", []string{}, "include only changes with these tags (supports glob patterns)")
	rootCmd.Flags().StringSliceVarP(&excludeTags, "exclude-tags", "x", []string{}, "exclude changes with these tags (supports glob patterns)")
	_ = rootCmd.Flags().MarkHidden("include-tags")
	_ = rootCmd.Flags().MarkHidden("exclude-tags")
}

// Execute runs the root command.
func Execute() {
	// It is tempting to handle this error, but don't do it. Cobra already does
	// all the reporting necessary. Any additional reporting is simply redundant
	// and repetitive.
	_ = rootCmd.Execute()
}
