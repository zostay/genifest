package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/zostay/genifest/pkg/manager/k8s"

	"github.com/zostay/genifest/pkg/log"
)

var (
	// generateManifestsCmd is the command configuration for generate-manifests.
	generateManifestsCmd = &cobra.Command{
		Use:   "run",
		Short: "Generate deployment manifests from template source for gitops",
		Args:  cobra.MaximumNArgs(1),
		Run:   RunGenerateManifests,
	}

	skipSecrets bool
	disableApi  bool
)

func init() {
	generateManifestsCmd.Flags().BoolVar(&skipSecrets, "skip-secrets", true, "skip generating deploy manifests containing secrets")
	generateManifestsCmd.Flags().BoolVar(&disableApi, "disable-api", false, "prevent kubernetes API calls")
}

// RunGenerateManifests performs argument parsing and startup, generates
// deployment manifests from source templates, and reports any errors that
// occur.
func RunGenerateManifests(cmd *cobra.Command, args []string) {
	match := ""
	if len(args) > 0 {
		match = args[0]
	}

	ctx := context.Background()

	sayMatch := match
	if sayMatch == "" {
		sayMatch = "all"
	}
	sayMatch = "matching " + sayMatch
	log.LineAndSayf(
		"TASK",
		"Generate manifests from source configurations %s",
		sayMatch)

	var err error
	for _, cluster := range c.Clusters {
		err = k8s.GenerateK8sResources(ctx, c, &cluster, match, skipSecrets, disableApi)
		if err != nil {
			err = fmt.Errorf("GenerateManifests: %w", err)
			break
		}
	}

	if err != nil {
		log.LineAndSayf("FATAL", "%v", err)
		os.Exit(1)
	}
}
