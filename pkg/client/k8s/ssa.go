package k8s

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

	"github.com/zostay/genifest/pkg/log"
)

const (
	FieldManagerGenifest = "genifest" // k8s SSA field manager string used by this app
)

var decun = yaml.NewDecodingSerializer(
	unstructured.UnstructuredJSONScheme,
)

// PrepareResource parses a configuration string into an applicable parsed
// configuration.
func (c *Client) PrepareResource(
	config string,
) (*unstructured.Unstructured, dynamic.ResourceInterface, error) {
	obj := new(unstructured.Unstructured)
	_, gvk, err := decun.Decode([]byte(config), nil, obj)
	if err != nil {
		return nil, nil, fmt.Errorf("decun.Decode(): %w", err)
	}

	mapping, err := c.mapper.RESTMapping(
		gvk.GroupKind(),
		gvk.Version,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("c.mapper.RESTMappig(): %w", err)
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = c.dyn.Resource(mapping.Resource).Namespace(
			obj.GetNamespace(),
		)
	} else {
		dr = c.dyn.Resource(mapping.Resource)
	}

	return obj, dr, nil
}

// SerializeResource turns a resource into JSON bytes ready for application via
// ApplySerializedResource and returns those bytes along with the namespace into
// which the resource should be applied.
func (c *Client) SerializeResource(
	un *unstructured.Unstructured,
) (*SerializedResource, error) {
	gvk := un.GroupVersionKind()

	mapping, err := c.mapper.RESTMapping(
		gvk.GroupKind(),
		gvk.Version,
	)
	if err != nil {
		return nil, fmt.Errorf("c.mapper.RESTMapping(): %w", err)
	}

	ns := un.GetNamespace()
	if ns == "" {
		ns = "default"
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = c.dyn.Resource(mapping.Resource).Namespace(ns)
	} else {
		dr = c.dyn.Resource(mapping.Resource)
	}

	data, err := json.Marshal(un)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal(): %w", err)
	}

	return NewSerializedResource(un, dr, data), nil
}

// ApplyResource serializes a resource and then applies.
func (c *Client) ApplyResource(
	ctx context.Context,
	un *unstructured.Unstructured,
	force bool,
) error {
	sr, err := c.SerializeResource(un)
	if err != nil {
		return err
	}

	err = sr.Apply(ctx, force)
	if err != nil {
		return err
	}

	return nil
}

// ApplyResourceConfig performs server-side apply for the given resource
// configuration.
func (c *Client) ApplyResourceConfig(
	ctx context.Context,
	context,
	config string,
	force bool,
) error {
	obj, dr, err := c.PrepareResource(config)
	if err != nil {
		return fmt.Errorf("c.PrepareResource(): %w", err)
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("json.Marshal(): %w", err)
	}

	ns := obj.GetNamespace()
	if ns == "" {
		ns = "default"
	}

	log.Linef("PATCH", "Patching from config %q / %q", ns, obj.GetName())
	log.LineBytes("PATCH-DATA", data)

	_, err = dr.Patch(
		ctx,
		obj.GetName(),
		types.ApplyPatchType,
		data,
		metav1.PatchOptions{
			Force:        &force,
			FieldManager: FieldManagerGenifest,
		},
	)

	if err != nil {
		return fmt.Errorf("dr.Patch(%q, %q, %q): %w", ns, obj.GetName(), obj.GetKind(), err)
	}

	return nil
}

// DeleteResource deletes a single resource given by name.
func (c *Client) DeleteResource(
	ctx context.Context,
	un *unstructured.Unstructured,
) error {
	gvk := un.GroupVersionKind()

	mapping, err := c.mapper.RESTMapping(
		gvk.GroupKind(),
		gvk.Version,
	)
	if err != nil {
		return fmt.Errorf("c.mapper.RESTMapping(): %w", err)
	}

	ns := un.GetNamespace()
	if ns == "" {
		ns = "default"
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = c.dyn.Resource(mapping.Resource).Namespace(ns)
	} else {
		dr = c.dyn.Resource(mapping.Resource)
	}

	log.Linef("DELETE", "Delete from unstructured %q / %q", ns, un.GetName())

	err = dr.Delete(
		ctx,
		un.GetName(),
		metav1.DeleteOptions{},
	)

	if err != nil {
		return fmt.Errorf("dr.Delete(%q, %q, %q): %w", ns, un.GetName(), un.GetKind(), err)
	}

	return nil
}

// DeleteResourceConfig performs deletion of the resource described by the given
// configuration file contents.
func (c *Client) DeleteResourceConfig(
	ctx context.Context,
	config string,
) error {
	obj, dr, err := c.PrepareResource(config)
	if err != nil {
		return err
	}

	log.Linef("DELETE", "Delete from config %q / %q", obj.GetNamespace(), obj.GetName())

	return dr.Delete(
		ctx,
		obj.GetName(),
		metav1.DeleteOptions{},
	)
}
