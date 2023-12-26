package k8stools

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

// ConvertFromUnstructured is a helper that converts an unstructured kubernetes
// resource into the given structured form.
func ConvertFromUnstructured(
	un *unstructured.Unstructured,
	to interface{},
) error {
	return scheme.Scheme.Convert(un, to, nil)
}

// ConvertToUnstructured is a helper that converts a structured kubernetes
// resource into it's unstructured form.
func ConvertToUnstructured(
	from interface{},
	un *unstructured.Unstructured,
) error {
	return scheme.Scheme.Convert(from, un, nil)
}
