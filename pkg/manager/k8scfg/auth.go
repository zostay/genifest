package k8scfg

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/zostay/genifest/pkg/client/k8s"
	k8scfg "github.com/zostay/genifest/pkg/config/kubecfg"
	"github.com/zostay/genifest/pkg/k8stools"
	"github.com/zostay/genifest/pkg/log"
)

const (
	AnnotationRotationEnabled   = "qubling.cloud/key-rotation"
	AnnotationIAMUser           = "iam.amazonaws.com/user"
	AnnotationManagedSecretName = "qubling.cloud/managed-secret-name"

	AnnotationValueRotationEnabled = "perform"
)

var AccessKeyLifetime = 30 * 24 * time.Hour

func insertEnvSecret(
	container *corev1.Container,
	envKey,
	secretRef,
	secretKey string,
) {
	envVar := corev1.EnvVar{
		Name: envKey,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretRef,
				},
				Key: secretKey,
			},
		},
	}

	container.Env = append(container.Env, envVar)
}

func insertAccessKeyEnvSecrets(
	containers []corev1.Container,
	secretRef string,
) {
	// insert the access key environment variables
	for i, cont := range containers {
		if cont.Env == nil {
			cont.Env = make([]corev1.EnvVar, 0, 2)
		}

		insertEnvSecret(&containers[i],
			"AWS_ACCESS_KEY_ID", secretRef, k8s.AwsAccessKeyId)
		insertEnvSecret(&containers[i],
			"AWS_SECRET_ACCESS_KEY", secretRef, k8s.SecretAccessKey)
	}
}

func rewriteAuth(
	ctx context.Context,
	tools Tools,
	rin k8scfg.Resource,
	resource interface{},
	podSpec *corev1.PodSpec,
	rewriteOpt *RewriteOptions,
) ([]k8scfg.ProcessedResource, error) {
	un := rin.Data

	// see if the user annotation is set or quit
	user := un.GetAnnotations()[AnnotationIAMUser]
	if user == "" {
		return []k8scfg.ProcessedResource{rin.ProcessedResource()}, nil
	}

	// see if rotation is enabled
	enablement, ok := un.GetAnnotations()[AnnotationRotationEnabled]
	if ok && enablement != AnnotationValueRotationEnabled {
		return []k8scfg.ProcessedResource{rin.ProcessedResource()}, nil
	}

	// see if the secret name annotation is set or use the user name as the
	// secret name
	ns := un.GetNamespace()
	name := un.GetAnnotations()[AnnotationManagedSecretName]
	if name == "" {
		name = user
	}

	replaceSecret := false
	res := make([]k8scfg.ProcessedResource, 0, 2)
	if !rewriteOpt.SkipSecrets {

		iamc, err := tools.IAM()
		if err != nil {
			return nil, fmt.Errorf("tools.IAM(): %w", err)
		}

		kube, err := tools.Kube()
		if err != nil {
			return nil, fmt.Errorf("tools.Kube(): %w", err)
		}

		// get the current access key
		userKey, keyDate, err := iamc.BestAccessKeyForUser(user)
		if err != nil {
			return nil, fmt.Errorf("iamc.BestAccessKeyForUser(): %w", err)
		}

		// check to see if the secret needs rotation and replacement
		if userKey == "" {
			log.Line("ACCESSKEY", "No API key found.")
			replaceSecret = true
		} else if time.Since(keyDate) > AccessKeyLifetime {
			log.Linef("ACCESSKEY", "API key is too old (%v).", keyDate)
			replaceSecret = true
		} else {
			ak, err := kube.CurrentAccessKeyFromSecrets(ctx, ns, name)
			if err != nil {
				return nil, fmt.Errorf("kube.CurrentAccessKeyFromSecrets(): %w", err)
			}

			if ak != userKey {
				log.Line("ACCESSKEY", "Current API secret differs from AWS secret.")
				replaceSecret = true
			}
		}

		// rotate the secret if we determined it needs rotation
		if replaceSecret {
			log.Linef("ACCESSKEY", "Rotating access key for user %q", user)

			ak, sk, err := iamc.RotateAccessKeyForUser(user)
			if err != nil {
				return nil, fmt.Errorf("iamc.RotateAccessKeyForUser(): %w", err)
			}

			aksr, err := k8s.MakeAccessKeySecretResource(ns, name, ak, sk)
			if err != nil {
				return nil, fmt.Errorf("k8s.MakeAccessKeySecretResource(): %w", err)
			}

			// only update the secret on rotation
			prSecret := k8scfg.ProcessedResource{
				Data: aksr,
			}
			res = append(res, prSecret)
		}
	}

	// insert the access key environment variables
	insertAccessKeyEnvSecrets(podSpec.Containers, name)
	insertAccessKeyEnvSecrets(podSpec.InitContainers, name)

	pr := k8scfg.ProcessedResource{
		Data:            resource,
		ResourceOptions: rin.ResourceOptions,
	}
	pr.NeedsRestart = replaceSecret || pr.NeedsRestart

	res = append(res, pr)

	return res, nil
}

// RewriteDeploymentAuth is a RewriteRoutine
// which looks for the iam.amazonaws.com/user annotation in deployments. When
// found, it finds that user, checks on the status of the managed secret for
// tracking the access key information for the user, and refreshes that status
// if needed.
//
// The managed secret either has the name qubling.cloud/managed-secret-name (if
// present as an annotation on the deployment) or the name of the user is used
// as the secret name.
//
// If the associated secret does not exist, an access key is generated, the
// secret is deployed, and the deployment is marked for restart.
//
// If the associated secret has an access key that differs from the most recent
// access key for the IAM user, the access key is rotated, the secret is
// updated, and the deployment is marked for restart.
//
// If the key associated with the user is older than AccessKeyLifetime, then the
// access key for the IAM user is rotated, the secret is updated, and the
// deployment is marked for restart.
//
// In all cases where the iam.amazon.com/user annotation is set, the
// environment for each container in the deployment's pod template is updated to
// include an AWS_ACCESS_KEY_ID and an AWS_SECRET_ACCESS_KEY that refer to those
// values in the managed secret.
func RewriteDeploymentAuth(
	ctx context.Context,
	tools Tools,
	rin k8scfg.Resource,
	opt *RewriteOptions,
) ([]k8scfg.ProcessedResource, error) {
	un := rin.Data

	// we only care about deployments
	if un.GetKind() != "Deployment" {
		return []k8scfg.ProcessedResource{rin.ProcessedResource()}, nil
	}

	var deployment appsv1.Deployment
	err := k8stools.ConvertFromUnstructured(un, &deployment)
	if err != nil {
		return nil, fmt.Errorf("k8stools.ConvertFromUnstructures(): %w", err)
	}
	podSpec := &deployment.Spec.Template.Spec

	return rewriteAuth(ctx, tools, rin, &deployment, podSpec, opt)
}

// RewriteCronJobAuth is a RewriteRoutine
// which looks for the iam.amazonaws.com/user annotation in cronjobs. When
// found, it finds that user, checks on the status of the managed secret for
// tracking the access key information for the user, and refreshes that status
// if needed.
//
// The managed secret either has the name qubling.cloud/managed-secret-name (if
// present as an annotation on the cronjob) or the name of the user is used
// as the secret name.
//
// If the associated secret does not exist, an access key is generated, the
// secret is deployed, and the cronjob is marked for restart.
//
// If the associated secret has an access key that differs from the most recent
// access key for the IAM user, the access key is rotated, the secret is
// updated, and the cronjob is marked for restart.
//
// If the key associated with the user is older than AccessKeyLifetime, then the
// access key for the IAM user is rotated, the secret is updated, and the
// cronjob is marked for restart.
//
// In all cases where the iam.amazon.com/user annotation is set, the
// environment for each container in the cronjob's pod template is updated to
// include an AWS_ACCESS_KEY_ID and an AWS_SECRET_ACCESS_KEY that refer to those
// values in the managed secret.
func RewriteCronJobAuth(
	ctx context.Context,
	tools Tools,
	rin k8scfg.Resource,
	opt *RewriteOptions,
) ([]k8scfg.ProcessedResource, error) {
	un := rin.Data

	// we only care about cronjobs
	if un.GetKind() != "CronJob" {
		return []k8scfg.ProcessedResource{rin.ProcessedResource()}, nil
	}

	var cronjob batchv1.CronJob
	err := k8stools.ConvertFromUnstructured(un, &cronjob)
	if err != nil {
		return nil, fmt.Errorf("k8stools.ConvertFromUnstructures(): %w", err)
	}
	podSpec := &cronjob.Spec.JobTemplate.Spec.Template.Spec

	return rewriteAuth(ctx, tools, rin, &cronjob, podSpec, opt)
}
