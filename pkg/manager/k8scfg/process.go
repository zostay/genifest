package k8scfg

import (
	"context"
	"errors"
	"fmt"

	k8scfg "github.com/zostay/genifest/pkg/config/kubecfg"
	"github.com/zostay/genifest/pkg/log"
)

var ErrSecret = errors.New("SKIP SECRET")

var Rewriters = []RewriteRoutine{
	RewriteDeploymentAuth,
	RewriteCronJobAuth,
}

// ProcessResourceFile reads the contents of the named resource file and
// breaks it into individual resources. These are each templated and rewritten
// and then the result is returned as a slice of Resource objects, which contain
// the parsed resource and any other options.
//
// Returns an error if any of this fails.
func ProcessResourceFile(
	ctx context.Context,
	tools Tools,
	config string,
	skipSecrets bool,
) ([]k8scfg.Resource, error) {
	c, err := tools.ResMgr(ctx)
	if err != nil {
		return nil, fmt.Errorf("tools.ResMgr(): %w", err)
	}

	if skipSecrets {
		secretsDie := func(_ ...interface{}) (string, error) {
			return "", ErrSecret
		}
		c.SetFunc("kubeseal", secretsDie)
		c.SetFunc("sshKey", secretsDie)
		c.SetFunc("zostaySecret", secretsDie)
	}

	cfs, err := c.ReadResourceFile(config)
	if err != nil {
		return nil, fmt.Errorf("c.ReadResourceFile(): %w", err)
	}

	ress := make([]k8scfg.Resource, 0, len(cfs))
	for _, cf := range cfs {
		res, err := c.TemplateConfigFile(config, cf.Config)
		if err != nil {
			if skipSecrets && errors.Is(err, ErrSecret) {
				// just ignore this and keep going
				log.Linef("SKIP", "Skip templating a resource in %q because it contains a secret.", config)
				continue
			}
			return nil, fmt.Errorf("c.TemplateConfigFile(): %w", err)
		}

		rewriteOpt := RewriteOptions{
			SkipSecrets: skipSecrets,
		}
		routs, err := RewriteConfigFile(
			ctx, tools, res, cf.ResourceOptions, Rewriters, &rewriteOpt)
		if err != nil {
			return nil, fmt.Errorf("c.RewriteConfigFile(): %w", err)
		}

		ress = append(ress, routs...)
	}

	return ress, nil
}
