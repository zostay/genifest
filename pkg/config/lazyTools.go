package config

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/pelletier/go-toml/v2"

	"github.com/zostay/genifest/pkg/client/aws/iam"

	"github.com/zostay/genifest/pkg/client/k8s"

	k8scfg "github.com/zostay/genifest/pkg/config/kubecfg"
	"github.com/zostay/genifest/pkg/log"
	cfgstr "github.com/zostay/genifest/pkg/strtools"
)

type LazyTools struct {
	cf *Config
	c  *Cluster

	kube *k8s.Client
	iam  *iam.Client
}

func (t *LazyTools) Kube() (*k8s.Client, error) {
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
		iam, err := iam.New()
		if err != nil {
			return nil, err
		}

		t.iam = iam
	}

	return t.iam, nil
}

func (t *LazyTools) ResMgr(ctx context.Context) (*k8scfg.Client, error) {
	funcMap := t.makeFuncMap(ctx)
	return k8scfg.New(t.cf.CloudHome, funcMap), nil
}

// MakeFuncMap builds a template function map that is used while templating
// resource and configuration files.
func (t *LazyTools) makeFuncMap(
	ctx context.Context,
) template.FuncMap {
	// Given an object, return the serialized TOML version of it.
	tomlize := func(o interface{}) (string, error) {
		bs, err := toml.Marshal(o)
		if err != nil {
			return "", err
		}

		return string(bs), nil
	}

	// A simple map lookup function in DynamoDB
	ddbLookup := func(table, field string, key map[string]interface{}) (string, error) {
		ddbKey := make(map[string]*dynamodb.AttributeValue, len(key))
		for k, v := range key {
			ddbKey[k] = &dynamodb.AttributeValue{S: aws.String(v.(string))}
		}
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(t.c.AWS.Region),
		})
		if err != nil {
			return "", err
		}
		ddbc := dynamodb.New(sess)
		in := dynamodb.GetItemInput{
			TableName: aws.String(table),
			Key:       ddbKey,
		}
		out, err := ddbc.GetItem(&in)
		if err != nil {
			return "", err
		}

		fps := strings.SplitN(field, ".", 2)
		fieldName, fieldType := fps[0], fps[1]

		if out.Item == nil {
			return "", fmt.Errorf("no counter named %s", key["Project"])
		}

		switch fieldType {
		case "S":
			return aws.StringValue(out.Item[fieldName].S), nil
		case "N":
			return aws.StringValue(out.Item[fieldName].N), nil
		default:
			return "", fmt.Errorf("unknown field type %q", fieldType)
		}
	}

	// Generate a map of key => secret pairs, where the secret is looked up.
	secretDict := func(sd ...string) (map[string]interface{}, error) {
		var k string
		dict := make(map[string]interface{})
		for i, v := range sd {
			if i%2 == 0 {
				k = v
			} else {
				var err error
				dict[k], err = cfgstr.GhostSecret(v)
				if err != nil {
					return nil, err
				}
			}
		}
		return dict, nil
	}

	// Lookup an EFS file systems description
	awsDescribeEfsFileSystemId := func(token string) (string, error) {
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(t.c.AWS.Region),
		})
		if err != nil {
			return "", err
		}
		efsc := efs.New(sess)
		in := efs.DescribeFileSystemsInput{
			CreationToken: aws.String(token),
		}
		out, err := efsc.DescribeFileSystems(&in)
		if err != nil {
			return "", err
		}
		return aws.StringValue(out.FileSystems[0].FileSystemId), nil
	}

	// Lookup EFS mount targets
	awsDescribeEfsMountTargets := func(id string) (*efs.DescribeMountTargetsOutput, error) {
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(t.c.AWS.Region),
		})
		if err != nil {
			return nil, err
		}
		efsc := efs.New(sess)
		in := efs.DescribeMountTargetsInput{
			FileSystemId: aws.String(id),
		}
		out, err := efsc.DescribeMountTargets(&in)
		if err != nil {
			return nil, err
		}
		return out, nil
	}

	// Lookup one of my SSH keys
	sshKey := func(name string) (string, error) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		bs, err := ioutil.ReadFile(filepath.Join(home, ".ssh", name))
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(bs)), nil
	}

	sshKnownHost := func(name string) (string, error) {
		ksCmd := exec.Command("ssh-keyscan", name)
		out, err := ksCmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}

	file := func(app, path string) (string, error) {
		p := filepath.Join(t.cf.CloudHome, "files", app+path)
		data, err := ioutil.ReadFile(p)
		log.LineBytes("EMBED", data)
		if err != nil {
			return "", err
		}
		return string(data), err
	}

	templateFile := func(name string, tmpl string) (string, error) {
		rmgr, err := t.ResMgr(ctx)
		if err != nil {
			return "", err
		}

		return rmgr.TemplateConfigFile(name, []byte(tmpl))
	}

	return template.FuncMap{
		"tomlize":                    tomlize,
		"secretDict":                 secretDict,
		"ddbLookup":                  ddbLookup,
		"awsDescribeEfsFileSystemId": awsDescribeEfsFileSystemId,
		"awsDescribeEfsMountTargets": awsDescribeEfsMountTargets,
		"sshKey":                     sshKey,
		"sshKnownHost":               sshKnownHost,
		"file":                       file,
		"applyTemplate":              templateFile,
		"zostaySecret":               cfgstr.GhostSecret,
		"kubeseal":                   cfgstr.KubeSeal,
	}
}
