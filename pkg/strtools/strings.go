package cfgstr

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zostay/ghost/pkg/secrets"

	"github.com/zostay/ghost/pkg/config"
	"github.com/zostay/ghost/pkg/keeper"

	_ "github.com/zostay/ghost/pkg/secrets/cache"
	_ "github.com/zostay/ghost/pkg/secrets/http"
	_ "github.com/zostay/ghost/pkg/secrets/human"
	_ "github.com/zostay/ghost/pkg/secrets/keepass"
	_ "github.com/zostay/ghost/pkg/secrets/lastpass"
	_ "github.com/zostay/ghost/pkg/secrets/policy"
)

func IndentSpaces(n int, s string) string {
	var out strings.Builder
	first := true
	for _, line := range strings.SplitAfter(s, "\n") {
		if !first {
			fmt.Fprint(&out, strings.Repeat(" ", n))
		}
		fmt.Fprint(&out, line)
		first = false
	}
	return out.String()
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

func MakeMatch(match string) string {
	if match == "" {
		match = "**/*"
	}

	if len(strings.Split(match, "/")) == 1 {
		match = "**/" + match
	}

	if filepath.Ext(match) == "" {
		match += ".yaml"
	}

	return match
}
