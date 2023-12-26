package k8s

import (
	"context"
	"time"

	apimetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	appsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
	metav1 "k8s.io/client-go/applyconfigurations/meta/v1"

	"github.com/zostay/genifest/pkg/log"
)

const (
	FieldManagerRestart = "qubling.cloud/restart"

	AnnotationRestartTrigger = "qubling.cloud/restart-trigger"
)

// TODO This manual restart is a lame way to do this. I do this because I need
// the tooling to automatically perform restarts when a secret is updated.
// However, a better way that would work for this and other cases where a
// deployment or cronjob needs to be refreshed properly would be to instead
// treat ConfigMap and Secret as immutable. Every change to a ConfigMap or
// Secret results in a new object being created with a new name. Any object
// referring to these gets updated to refer to the new object. When that latter
// update the restart happens automatically. Then, I need another tool that goes
// back through later and prunes managed, but unreferenced ConfigMap and Secret
// objects.
//
// This will require some thought and planning to make sure I can track the
// ConfigMaps and Secrets that are managed and how to track which objects refer
// to them that need an update somewhere.

// Restart performs different tasks depending on the kind of record.
//
// * Pod - no operation
// * ReplicaSet - delete pods
// * Deployment - modify the pod template to trigger a redeploy
// * Daemonset - modify the pod template to trigger a redeploy
// * Statefulset - modify the pod template to trigger a redeploy
// * CronJob - modify the job template to trigger restart, also delete jobs
// * Job - delete pods
//
// As of this writing, only Deployment handling is implemented.
func (c *Client) Restart(
	ctx context.Context,
	un *unstructured.Unstructured,
	force bool,
) error {
	switch un.GetKind() {
	case "Deployment":
		deployment, err := c.kube.AppsV1().Deployments(un.GetNamespace()).
			Get(ctx, un.GetName(), apimetav1.GetOptions{})
		if err != nil {
			return err
		}

		d, err := appsv1.ExtractDeployment(deployment, FieldManagerRestart)
		if err != nil {
			return err
		}
		if d.Spec == nil {
			d.Spec = appsv1.DeploymentSpec()
		}
		if d.Spec.Template == nil {
			d.Spec.Template = corev1.PodTemplateSpec()
		}
		if d.Spec.Template.ObjectMetaApplyConfiguration == nil {
			d.Spec.Template.ObjectMetaApplyConfiguration = metav1.ObjectMeta()
		}
		if d.Spec.Template.Annotations == nil {
			d.Spec.Template.Annotations = make(map[string]string, 1)
		}
		d.Spec.Template.Annotations[AnnotationRestartTrigger] = time.Now().String()

		_, err = c.kube.AppsV1().
			Deployments(un.GetNamespace()).
			Apply(ctx, d, apimetav1.ApplyOptions{
				Force:        true,
				FieldManager: FieldManagerRestart,
			})
		if err != nil {
			return err
		}
	default:
		log.Linef("NYI", "The restart operation is not implemented for kind %q", un.GetKind())
	}

	return nil
}
