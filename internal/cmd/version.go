package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print version information for genifest",
	Args:  cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("Genifest", Version, strings.Join([]string{runtime.GOOS, runtime.GOARCH}, "/"))
		fmt.Println("\nCopyright 2025 Qubling LLC.")
		fmt.Println("This program is free software, licensed under an MIT License.")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}