package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zostay/genifest/pkg/config"
	"github.com/zostay/genifest/pkg/log"
)

//go:embed version.txt
var Version string

var (
	logStderr   bool
	configFile  string
	clusterName string

	c *config.Config

	rootCmd = &cobra.Command{
		Use:   "genifest",
		Short: "Prepare the configuration of the kubenetes cluster from templates",
	}

	printVersionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version of the genifest tool",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("genifest v%s\n", Version)
		},
	}
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVar(&logStderr, "log-to-stderr", false, "send logs to stdout only, skip logging to file")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "name of the configuration file to use")
	rootCmd.PersistentFlags().StringVarP(&clusterName, "cluster-name", "c", "", "only work with the cluster with this name")

	rootCmd.AddCommand(generateManifestsCmd, printVersionCmd)
}

func initConfig() {
	var err error

	c, err = config.InitConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL Unable to load configuration %q: %v\n", configFile, err)
		os.Exit(1)
	}

	err = log.Setup(c.CloudHome, "", logStderr, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to rotate and open log file: %v\n", err)
		os.Exit(1)
	}

	log.Line("START", strings.Repeat("#", 78))
	log.Linef("START", "# Running %s", os.Args[0])

	found := false
	if clusterName != "" {
		log.LineAndSayf("CLUSTER", "Only working on cluster %q", clusterName)

		// Only keep parts of the config dictated by the named cluster
		// configuration
		for k := range c.Clusters {
			if k == clusterName {
				found = true
			} else {
				delete(c.Clusters, k)
			}
		}
	} else {
		for k, cl := range c.Clusters {
			if cl.Disabled {
				delete(c.Clusters, k)
			} else {
				found = true
			}
		}
	}

	if !found {
		log.LineAndSayf("FATAL", "No cluster configuration named %q\n", clusterName)
		os.Exit(1)
	}

	if c.CloudHome == "" {
		var err error
		c.CloudHome, err = os.Getwd()
		if err != nil {
			log.LineAndSayf("FATAL", "Please set GENIFEST_HOME in your environment.\n")
			os.Exit(1)
		}
	}

	if c.CloudHome == "" {
		log.LineAndSayf("FATAL", "Please set GENIFEST_HOME in your environment.\n")
		os.Exit(1)
	}

	validCloudHome, err := regexp.MatchString(`^[a-zA-Z0-9_./-]+$`, c.CloudHome)
	if err != nil {
		log.LineAndSayf("FATAL", "GENIFEST_HOME contains illegal characters.\n")
		os.Exit(1)
	}

	if !validCloudHome {
		fmt.Fprintf(os.Stderr, "Please set GENIFEST_HOME to a valid value.\n")
		os.Exit(1)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		cobra.CheckErr(err)
	}
}
