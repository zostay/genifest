package config

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/zostay/genifest/pkg/client/aws/iam"
	"github.com/zostay/genifest/pkg/tmpltools"

	"github.com/zostay/genifest/pkg/client/k8s"

	k8scfg "github.com/zostay/genifest/pkg/config/kubecfg"
	k8smgr "github.com/zostay/genifest/pkg/manager/k8scfg"
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

func (t *LazyTools) ResMgr(ctx context.Context, skipSecrets bool) (*k8scfg.Client, error) {
	rmgr := k8scfg.New(t.cf.CloudHome)
	rmgr.SetFuncMap(t.makeFuncMap(ctx, rmgr, skipSecrets))
	return rmgr, nil
}

// MakeFuncMap builds a template function map that is used while templating
// resource and configuration files.
func (t *LazyTools) makeFuncMap(
	ctx context.Context,
	rmgr *k8scfg.Client,
	skipSecrets bool,
) template.FuncMap {
	aws := tmpltools.AWS{
		Region: t.c.AWS.Region,
	}

	ghost := tmpltools.Ghost{
		Context:    ctx,
		Config:     t.c.Ghost.ConfigFile,
		KeeperName: t.c.Ghost.Keeper,
	}

	filesRoot := t.cf.CloudHome
	if filesDir := t.c.FilesDir; filesDir != "" {
		if strings.HasPrefix(filesDir, "/") {
			filesRoot = filesDir
		} else {
			filesRoot = filepath.Join(filesRoot, filesDir)
		}
	} else {
		filesRoot = filepath.Join(filesRoot, "files")
	}

	file := func(app, path string) (string, error) {
		return tmpltools.File(filesRoot, app, path)
	}

	applyTemplate := func(name, data string) (string, error) {
		return rmgr.TemplateConfigFile(name, []byte(data))
	}

	fm := template.FuncMap{
		"tomlize":                    tmpltools.Tomlize,
		"secretDict":                 ghost.SecretDict,
		"ddbLookup":                  aws.DDBLookup,
		"awsDescribeEfsFileSystemId": aws.DescribeEfsFileSystemId,
		"awsDescribeEfsMountTargets": aws.DescribeEfsMountTargets,
		"sshKey":                     tmpltools.SSHKey,
		"sshKnownHost":               tmpltools.SSHKnownHost,
		"file":                       file,
		"applyTemplate":              applyTemplate,
		"zostaySecret":               ghost.Secret,
		"kubeseal":                   tmpltools.KubeSeal,
	}

	if skipSecrets {
		secretsDie := func(_ ...interface{}) (string, error) {
			return "", k8smgr.ErrSecret
		}
		fm["kubeseal"] = secretsDie
		fm["sshKey"] = secretsDie
		fm["zostaySecret"] = secretsDie
	}

	return fm
}
