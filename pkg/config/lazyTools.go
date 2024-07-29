package config

import (
	"context"
	"fmt"
	"text/template"

	"github.com/zostay/genifest/pkg/client/aws/iam"

	"github.com/zostay/genifest/pkg/client/k8s"

	k8scfg "github.com/zostay/genifest/pkg/config/kubecfg"
	cfgstr "github.com/zostay/genifest/pkg/strtools"
)

type LazyTools struct {
	cf *Config
	c  *Cluster

	kube *k8s.Client
	iam  *iam.Client

	noApi bool
}

func (t *LazyTools) Kube() (*k8s.Client, error) {
	if t.noApi {
		return nil, fmt.Errorf("no k8s API access")
	}

	if t.kube == nil {
		kube, err := k8s.New(t.c.Context)
		if err != nil {
			return nil, err
		}

		t.kube = kube
	}

	return t.kube, nil
}

func (t *LazyTools) IAM() (*iam.Client, error) {
	if t.iam == nil {
		client, err := iam.New()
		if err != nil {
			return nil, err
		}

		t.iam = client
	}

	return t.iam, nil
}

func (t *LazyTools) ResMgr(ctx context.Context) (*k8scfg.Client, error) {
	rmgr := k8scfg.New(t.cf.CloudHome)
	rmgr.SetFuncMap(t.makeFuncMap(ctx, rmgr))
	return rmgr, nil
}

// MakeFuncMap builds a template function map that is used while templating
// resource and configuration files.
func (t *LazyTools) makeFuncMap(
	ctx context.Context,
	rmgr *k8scfg.Client,
) template.FuncMap {

	aws := cfgstr.AWS{
		Region: t.c.AWS.Region,
	}

	ghost := cfgstr.Ghost{
		Context:    ctx,
		Config:     t.c.Ghost.ConfigFile,
		KeeperName: t.c.Ghost.Keeper,
	}

	file := func(app, path string) (string, error) {
		return cfgstr.File(t.cf.CloudHome, app, path)
	}

	return template.FuncMap{
		"tomlize":                    cfgstr.Tomlize,
		"secretDict":                 ghost.SecretDict,
		"ddbLookup":                  aws.DDBLookup,
		"awsDescribeEfsFileSystemId": aws.DescribeEfsFileSystemId,
		"awsDescribeEfsMountTargets": aws.DescribeEfsMountTargets,
		"sshKey":                     cfgstr.SSHKey,
		"sshKnownHost":               cfgstr.SSHKnownHost,
		"file":                       file,
		"applyTemplate":              rmgr.TemplateConfigFile,
		"zostaySecret":               ghost.Secret,
		"kubeseal":                   cfgstr.KubeSeal,
	}
}
