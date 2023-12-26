package k8s

import (
	"context"
	"fmt"

	bitnamiv1alpha1 "github.com/bitnami-labs/sealed-secrets/pkg/apis/sealedsecrets/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appcorev1 "k8s.io/client-go/applyconfigurations/core/v1"

	cfgstr "github.com/zostay/genifest/pkg/strtools"
)

const (
	AwsAccessKeyId  = "aws_access_key_id"
	SecretAccessKey = "secret_access_key"

	AnnotationManagedSecret = "qubling.cloud/managed-secret"
)

// GetSecret will the secret data for the identified secret.
func (c *Client) GetSecret(
	ctx context.Context,
	ns,
	name string,
) (map[string]string, error) {
	s, err := c.kube.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return map[string]string{}, fmt.Errorf("c.kube.CoreV1().Secrets(%q).Get(%q): %w", ns, name, err)
	}

	if s == nil {
		return map[string]string{}, nil
	}

	res := make(map[string]string, len(s.Data))
	for k, v := range s.Data {
		res[k] = string(v)
	}

	return res, nil
}

// CurrentAccessKeyFromSecrets will find and return the access key id stored in
// the identified secret or will return an error.
func (c *Client) CurrentAccessKeyFromSecrets(ctx context.Context, ns, name string) (string, error) {
	s, err := c.kube.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return "", fmt.Errorf("c.kube.CoreV1().Secrets(%q).Get(%q): %w", ns, name, err)
	}

	if s == nil {
		return "", nil
	}

	akbs, ok := s.Data["aws_access_key_id"]
	if !ok {
		return "", nil
	}

	return string(akbs), nil
}

// UpdateAccessKey is syntactic sugar around UpdateSecret() for use with access
// key information.
func (c *Client) UpdateAccessKey(ctx context.Context, ns, name, accessKey, secretKey string) error {
	return c.UpdateSecret(
		ctx,
		ns, name,
		map[string]string{
			AwsAccessKeyId:  accessKey,
			SecretAccessKey: secretKey,
		},
	)
}

// UpdateSecret turns the given data map into the secret data in a secret object
// and applies the change immediately.
func (c *Client) UpdateSecret(
	ctx context.Context,
	ns,
	name string,
	data map[string]string,
) error {
	sac := MakeSecretApplyConfigResource(ns, name, data)

	_, err := c.kube.CoreV1().Secrets(ns).Apply(ctx, sac, metav1.ApplyOptions{})
	return err
}

// MakeSecretApplyConfigResource constructs and returns a secret object ready to
// be applied via the simple Apply() method.
func MakeSecretApplyConfigResource(
	ns,
	name string,
	data map[string]string,
) *appcorev1.SecretApplyConfiguration {
	return appcorev1.Secret(name, ns).
		WithAnnotations(map[string]string{
			AnnotationManagedSecret: "true",
		}).
		WithStringData(data)
}

// MakeSealedSecretResource constructs a bitnami sealed secret object for the
// given namespace, name, and data to encrypt. It will perform the encryption of
// that data.
func MakeSealedSecretResource(
	ns,
	name string,
	data map[string]string,
) (*bitnamiv1alpha1.SealedSecret, error) {
	encryptedData := make(map[string]string, len(data))
	for k, v := range data {
		var err error
		encryptedData[k], err = cfgstr.KubeSeal(ns, name, v)
		if err != nil {
			return nil, err
		}
	}

	ss := bitnamiv1alpha1.SealedSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Annotations: map[string]string{
				AnnotationManagedSecret: "true",
			},
		},
		Spec: bitnamiv1alpha1.SealedSecretSpec{
			EncryptedData: encryptedData,
		},
	}

	return &ss, nil
}

// MakeAccessKeySecretResource is syntactic sugar around MakeSecretResource()
// for use with access key information.
func MakeAccessKeySecretResource(
	ns,
	name,
	accessKey,
	secretKey string,
) (*bitnamiv1alpha1.SealedSecret, error) {
	return MakeSealedSecretResource(ns, name, map[string]string{
		AwsAccessKeyId:  accessKey,
		SecretAccessKey: secretKey,
	})
}
