package k8s

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

	"github.com/zostay/genifest/pkg/log"
)

// SerializedResource is a resource that has been fully prepared for
// application, but allows some introspection for detecting changes and other
// such details.
type SerializedResource struct {
	un *unstructured.Unstructured
	dr dynamic.ResourceInterface

	ns   string
	name string
	gvk  schema.GroupVersionKind

	data []byte
}

// NewSerializedResource creates a new SerializedResource from an unstructured
// resource and a dynamic resource interface and the serialized bytes.
func NewSerializedResource(
	un *unstructured.Unstructured,
	dr dynamic.ResourceInterface,
	data []byte,
) *SerializedResource {
	gvk := un.GroupVersionKind()

	ns := un.GetNamespace()
	if ns == "" {
		ns = "default"
	}

	name := un.GetName()

	return &SerializedResource{un, dr, ns, name, gvk, data}
}

// HasDynamicResource returns true if the serialized resource has a dynamic
// resource interface set.
func (s *SerializedResource) HasDynamicResource() bool {
	return s.dr != nil
}

// ResourceID returns a resource identifier consisting of
// "ns/group/version/kind/name" which provides a convenient naming scheme for
// comparing one revision of a resource to another.
func (s *SerializedResource) ResourceID() string {
	return strings.Join([]string{s.ns, s.gvk.Group, s.gvk.Version, s.gvk.Kind, s.name}, "/")
}

// Bytes returns the serialized resource as bytes.
func (s *SerializedResource) Bytes() []byte {
	return s.data
}

// Apply applies a SerializedResource to the cluster.
func (s *SerializedResource) Apply(
	ctx context.Context,
	force bool,
) error {
	log.Linef("PATCH", "Patching from unstructured %q / %q", s.ns, s.name)
	log.LineBytes("PATCH-DATA", s.data)

	_, err := s.dr.Patch(
		ctx,
		s.name,
		types.ApplyPatchType,
		s.data,
		metav1.PatchOptions{
			Force:        &force,
			FieldManager: FieldManagerGenifest,
		},
	)

	if err != nil {
		return fmt.Errorf("dr.Patch(%q, %q, %q): %w", s.ns, s.name, s.gvk.Kind, err)
	}

	return nil
}
