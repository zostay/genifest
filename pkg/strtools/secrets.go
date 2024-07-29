package strtools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zostay/ghost/pkg/config"
	"github.com/zostay/ghost/pkg/keeper"
	"github.com/zostay/ghost/pkg/secrets"
)

// SecretDict will generate a map of key => secret pairs, where the secret is looked up.
func SecretDict(sd ...string) (map[string]any, error) {
	var k string
	dict := make(map[string]any)
	for i, v := range sd {
		if i%2 == 0 {
			k = v
		} else {
			var err error
			dict[k], err = GhostSecret(v)
			if err != nil {
				return nil, err
			}
		}
	}
	return dict, nil
}

func GhostSecret(name string) (string, error) {
	c := config.Instance()
	err := c.Load("")
	if err != nil {
		return "", err
	}

	ctx := keeper.WithBuilder(context.Background(), c)
	k, err := keeper.Build(ctx, c.MasterKeeper)
	if err != nil {
		return "", err
	}

	ss, err := k.GetSecretsByName(ctx, name)
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
	cmd := exec.Command(
		"kubeseal", "--raw",
		"--namespace", ns,
		"--name", name,
		"--from-file", "/dev/stdin",
	)

	cmd.Stdin = strings.NewReader(secret)

	sealed := new(strings.Builder)
	cmd.Stdout = sealed

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return sealed.String(), nil
}
