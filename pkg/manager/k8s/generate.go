package k8s

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/zostay/genifest/pkg/client/k8s"
	"github.com/zostay/genifest/pkg/config"
	"github.com/zostay/genifest/pkg/config/kubecfg"
	"github.com/zostay/genifest/pkg/log"
	"github.com/zostay/genifest/pkg/manager/k8scfg"
)

// GenerateK8sResources locates all the configuration file templates, renders
// the templates to te deployment folder, and returns any errors that occurred
// while doing it. This sets up deployment via gitops through ArgoCD.
func GenerateK8sResources(
	ctx context.Context,
	cfg *config.Config,
	cluster *config.Cluster,
	match string,
	skipSecrets bool,
	disableApi bool,
) error {
	log.Line("TASK", "Generate deployment resource manifests from source templates.")

	configFiles, err := k8scfg.ConfigFiles(
		cfg.CloudHome,
		cluster.SourceDir,
		cluster.Limits.NotResourceFilesMatches(),
		match,
		false,
	)
	if err != nil {
		return fmt.Errorf("k8s.ConfigFiles: %w", err)
	}

	tools := cfg.Tools(cluster, disableApi)

	var serializeResource func(un *unstructured.Unstructured) (*k8s.SerializedResource, error)
	if disableApi {
		log.Line("SKIP", "Skipping API calls.")
		serializeResource = k8scfg.SerializeResource
	} else {
		kc, err := tools.Kube()
		if err != nil {
			return fmt.Errorf("tools.Kube(): %w", err)
		}

		serializeResource = kc.SerializeResource
	}

	allowedKind := cluster.Limits.KindsSet()
	blockedNs := cluster.Limits.NotNamespacesSet()
	errs := []error{}
	for _, pc := range configFiles {
		appName := filepath.Base(filepath.Dir(pc))
		appDir := filepath.Join(cluster.DeployDir, appName)

		fmt.Printf("Generate %s (app %s): %s ... ", cluster.Context, appName, pc)

		errsThisTime := 0
		resources, err := k8scfg.ProcessResourceFile(ctx, tools, pc, skipSecrets)
		if err != nil {
			errs = append(errs, fmt.Errorf("k8scfg.ProcessResourceFile(): %w", err))
			errsThisTime++
			resources = []kubecfg.Resource{}
		}

		skipped := 0
		for _, r := range resources {
			// check limits
			_, ok := allowedKind[r.Data.GetKind()]
			if len(allowedKind) > 0 && !ok {
				log.Linef("SKIP", "- Skip resource kind %q", r.Data.GetKind())
				skipped++
				continue
			}
			if _, blocked := blockedNs[r.Data.GetNamespace()]; blocked {
				log.Linef("SKIP", "- Skip resource namespaces %q", r.Data.GetNamespace())
				skipped++
				continue
			}

			sr, err := serializeResource(r.Data)
			if err != nil {
				errs = append(errs, fmt.Errorf("kube.SerializeResource(): %w", err))
				errsThisTime++
				continue
			}

			err = k8scfg.SaveResourceFile(ctx, tools, appDir, sr)
			if err != nil {
				errs = append(errs, fmt.Errorf("k8scfg.SaveResourceFile(): %w", err))
				errsThisTime++
				continue
			}
		}

		if skipped > 0 || len(resources) == 0 {
			if skipped == len(resources) {
				if errsThisTime > 0 {
					fmt.Println("skipped with ERRORS (see below).")
				} else {
					fmt.Println("skipped.")
				}
			} else if errsThisTime > 0 {
				fmt.Printf("done with ERRORS (see below), skipped %d of %d.\n",
					skipped, len(resources))
			} else {
				fmt.Printf("done, skipped %d of %d.\n", skipped, len(resources))
			}
		} else if errsThisTime > 0 {
			fmt.Println("ERRORS (see below).")
		} else {
			fmt.Println("done.")
		}
	}

	if len(errs) > 0 {
		ss := make([]string, len(errs))
		for i, err := range errs {
			ss[i] = err.Error()
		}
		return fmt.Errorf("error during apply:\n    - %s", strings.Join(ss, "\n    - "))
	}

	return nil
}
