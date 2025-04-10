package tmpltools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zostay/genifest/pkg/log"
	"github.com/zostay/ghost/pkg/config"
	"github.com/zostay/ghost/pkg/keeper"
	"github.com/zostay/ghost/pkg/secrets"
)

type Ghost struct {
	Config     string
	KeeperName string
	Context    context.Context

	k secrets.Keeper
}

func (g *Ghost) initialize() error {
	if g.k != nil {
		return nil
	}

	c := config.Instance()
	err := c.Load(g.Config)
	if err != nil {
		return err
	}

	keeperName := g.KeeperName
	if keeperName == "" {
		keeperName = c.MasterKeeper
	}

	ctx := keeper.WithBuilder(context.Background(), c)
	k, err := keeper.Build(ctx, keeperName)
	if err != nil {
		return err
	}

	g.k = k
	return nil
}

// SecretDict will generate a map of key => secret pairs, where the secret is looked up.
func (g *Ghost) SecretDict(sd ...string) (map[string]any, error) {
	var k string
	dict := make(map[string]any)
	for i, v := range sd {
		if i%2 == 0 {
			k = v
		} else {
			var err error
			dict[k], err = g.Secret(v)
			if err != nil {
				return nil, err
			}
		}
	}
	return dict, nil
}

func (g *Ghost) Secret(name string) (string, error) {
	if err := g.initialize(); err != nil {
		return "", err
	}

	ctx := g.Context
	if ctx == nil {
		ctx = context.Background()
	}

	ss, err := g.k.GetSecretsByName(ctx, name)
	if err != nil {
		return "", err
	}

	var s secrets.Secret
	if len(ss) == 1 {
		s = ss[0]
	} else {
		return "", fmt.Errorf("wrong number of secrets %d found for secret named %q", len(ss), name)
	}

	return s.Password(), nil
}

// KubeSeal runs the kubeseal command to output a raw sealed secret.
func KubeSeal(ns, name, secret string) (string, error) {
	// TODO This doesn't select context, but should.
	cmd := exec.Command(
		"kubeseal", "--raw",
		"--namespace", ns,
		"--name", name,
		"--from-file", "/dev/stdin",
	)

	cmd.Stdin = strings.NewReader(secret)

	sealed := new(strings.Builder)
	cmd.Stdout = sealed

	errors := new(strings.Builder)
	cmd.Stderr = errors

	err := cmd.Run()
	if err != nil {
		log.LineAndSay("STDERR", errors.String())
		return "", err
	}

	return sealed.String(), nil
}
