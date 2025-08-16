package cmd

import "github.com/spf13/cobra"

var (
	rootCmd = &cobra.Command{
		Use:  "genifest",
		Args: cobra.NoArgs,
		Run:  GenerateManifests,
	}

	includeTags, excludeTags []string
)

func init() {
	rootCmd.Flags().StringSliceVarP(&includeTags, "include-tags", "i", []string{}, "include tags")
	rootCmd.Flags().StringSliceVarP(&excludeTags, "exclude-tags", "x", []string{}, "exclude tags")
}

func GenerateManifests(_ *cobra.Command, _ []string) {
	panic("not implemented")
}