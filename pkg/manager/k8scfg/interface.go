package k8scfg

import (
	"context"

	"github.com/zostay/genifest/pkg/client/aws/iam"
	"github.com/zostay/genifest/pkg/client/k8s"
	k8scfg "github.com/zostay/genifest/pkg/config/kubecfg"
)

type Tools interface {
	Kube() (*k8s.Client, error)

	ResMgr(context.Context) (*k8scfg.Client, error)

	IAM() (*iam.Client, error)
}
