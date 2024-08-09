package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"

	cfgstr "github.com/zostay/genifest/pkg/strtools"
)

// Config defines configuration for the cluster.
type Config struct {
	// CloudHome is the absolute path to the root of the configuration.
	CloudHome string `mapstructure:"cloud_home"`

	// Clusters defines the orchestration configuration for each cluster managed
	// by this configuration.
	Clusters map[string]Cluster
}

// Cluster configures orchestration of a single cluster.
type Cluster struct {
	// Context names the configuration used for accessing the cluster.
	Context string

	// KubeDir tells genifest where to find the kubernetes resource files
	// which are applied to the cluster.
	KubeDir string `mapstructure:"kube_dir"`

	// SourceDir tells genifest where to find the source files to use for
	// generating the deployment resources.
	SourceDir string `mapstructure:"source_dir"`

	// DeployDir tells genifest where to send the generated resource files
	// for deployment via gitops.
	DeployDir string `mapstructure:"deploy_dir"`

	// Host names the hosting service on which the cluster is based.
	Host string

	// AWS provides AWS specific configuration when Host is set to AWS.
	AWS AWS `mapstructure:"aws"`

	// AutoDNS defines parameters on how DNS entries are automatically generated
	// and configured.
	AutoDNS AutoDNS `mapstructure:"auto_dns"`

	// Disabled is set to prevent the cluster from being configured unless it is
	// specifically named when running genifest.
	Disabled bool

	// Limits is a set of allowlists that define which resources genifest
	// will attempt to manage.
	Limits Limits

	// Ghost is the ghost configuration to use.
	Ghost Ghost
}

// Limits defines the allowlists and blocklists that identify resources the
// tooling will attempt to manage.
type Limits struct {
	// Kinds specifies which kinds of resources the tooling will attempt to
	// manage when set. If not set, no consideration of kinds is made.
	Kinds    []string
	kindsSet map[string]struct{}

	// NotNamespaces specifies a blocklist of namespaces that the tooling will
	// not attempt to manage.
	NotNamespaces    []string `mapstructure:"not_namespaces"`
	notNamespacesSet map[string]struct{}

	// NotResourceFiles specifies glob patterns to identify resource files the
	// tool will not attempt to manage.
	NotResourceFiles        []string `mapstructure:"not_resources"`
	notResourceFilesMatches []string
}

// Ghost defines the ghost configuration to use.
type Ghost struct {
	// ConfigFile is the path to the ghost configuration file to use (default is the
	// user's default ghost configuration file).
	ConfigFile string `mapstructure:"config_file"`

	// Keeper is the name of the keeper to use (default is the master keeper
	// defined in the ghost configuration file)..
	Keeper string
}

// AWS defines AWS specific configuration.
type AWS struct {
	// Region is the AWS region in which the cluster is hosted and should be used
	// by default for all AWS resources.
	Region string

	// LoadBalancer is either set to "classic" or "network" to determine how to
	// locate and configure load balancers attached to a cluster.
	LoadBalancer string `mapstructure:"load_balancer"`
}

// AutoDNS provides configuration for the automatic DNS tooling.
type AutoDNS struct {
	// Config is the name of the Terraform configuration used to generate
	// automatic DNS.
	Config string

	// File is the name of the Terrform file that is generated and managed by
	// the automatic DNS tooling.
	File string
}

func InitConfig(cfgFile string) (*Config, error) {
	var config Config

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("clusters")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("/etc")
		viper.AddConfigPath("/app")
		viper.AddConfigPath(".")
	}

	viper.SetEnvPrefix("genifest")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		return &config, fmt.Errorf("Error reading in clusters.yaml: %w", err)
	}

	// separate file for secret config in production
	viper.SetConfigFile("/etc/clusters-secrets.yaml")
	if err := viper.MergeInConfig(); err != nil {
		const errPre = "Error merging in clusters-secrets.yaml"

		// Make sure there's a warning recorded
		fmt.Fprintf(os.Stderr, "WARN %s: %v\n", errPre, err)
	}

	err := viper.Unmarshal(&config)
	if err != nil {
		return &config, err
	}

	return &config, nil
}

func (c *Config) Tools(cluster *Cluster, noApi bool) *LazyTools {
	return &LazyTools{cf: c, c: cluster, noApi: noApi}
}

func makeSet(list []string) map[string]struct{} {
	m := make(map[string]struct{}, len(list))
	for _, k := range list {
		m[k] = struct{}{}
	}
	return m
}

func (l *Limits) KindsSet() map[string]struct{} {
	if l.kindsSet == nil {
		l.kindsSet = makeSet(l.Kinds)
	}
	return l.kindsSet
}

func (l *Limits) DropKind(dropped string) {
	if l.kindsSet != nil {
		l.kindsSet = nil
	}

	newKinds := make([]string, len(l.Kinds))
	for _, k := range l.Kinds {
		if k == dropped {
			continue
		}
		newKinds = append(newKinds, k)
	}

	l.Kinds = newKinds
}

func (l *Limits) NotNamespacesSet() map[string]struct{} {
	if l.notNamespacesSet == nil {
		l.notNamespacesSet = makeSet(l.NotNamespaces)
	}
	return l.notNamespacesSet
}

func (l *Limits) NotResourceFilesMatches() []string {
	if l.notResourceFilesMatches == nil {
		l.notResourceFilesMatches = make([]string, len(l.NotResourceFiles))
		for i, v := range l.NotResourceFiles {
			l.notResourceFilesMatches[i] = cfgstr.MakeMatch(v)
		}
	}
	return l.notResourceFilesMatches
}
