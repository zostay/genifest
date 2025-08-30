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
)

func init() {
	// No global flags needed anymore
}

// Execute runs the root command.
func Execute() {
	// It is tempting to handle this error, but don't do it. Cobra already does
	// all the reporting necessary. Any additional reporting is simply redundant
	// and repetitive.
	_ = rootCmd.Execute()
}
