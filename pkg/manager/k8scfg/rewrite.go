package k8scfg

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	k8scfg "github.com/zostay/genifest/pkg/config/kubecfg"
	"github.com/zostay/genifest/pkg/k8stools"
)

type RewriteOptions struct {
	SkipSecrets bool
}

type RewriteRoutine func(context.Context, Tools, k8scfg.Resource, *RewriteOptions) ([]k8scfg.ProcessedResource, error)

// RewriteConfigFile applies rewrite routines to the configuration file. The
// configuration file is parsed into the generic unstructured.Unstructured
// format. It is passed to each handler in turn to be processed. The processor
// will then return at least one object (but possibly more if the object needs
// to generate additional objects in the process), which are then passed on to
// the next rewrite routines until all rewrite routines have been used to
// process the objects. This means later routines may run against more than one
// object per original singular objects.
//
// If any rewrite routine returns an error, the process is immediately halted
// and only an error is returned.
//
// If all rewrite routines succeed, the results are serialized back into YAML
// for further processing.
func RewriteConfigFile(
	ctx context.Context,
	tools Tools,
	data string,
	resourceOpt k8scfg.ResourceOptions,
	rewriters []RewriteRoutine,
	rewriteOpt *RewriteOptions,
) ([]k8scfg.Resource, error) {
	un, err := k8scfg.ParseResource([]byte(data))
	if err != nil {
		return nil, fmt.Errorf("ParseResource(): %w", err)
	}

	// We need to process the resources individually, not as a list.
	//
	// TODO Does this need to be able to handle nested lists?
	uns := make([]*unstructured.Unstructured, 0, 1)
	if un.IsList() {
		err := un.EachListItem(func(r runtime.Object) error {
			run, ok := r.(*unstructured.Unstructured)
			if !ok {
				return fmt.Errorf("unable to convert runtime.Object to expected *unstructured.Unstructured")
			}
			uns = append(uns, run)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("un.EachListItem(): %w", err)
		}
	} else {
		uns = append(uns, un)
	}

	keepOrConvert := func(from interface{}, to *unstructured.Unstructured) error {
		un, ok := from.(*unstructured.Unstructured)
		if ok {
			to.Object = un.DeepCopy().Object
		} else {
			err := k8stools.ConvertToUnstructured(from, to)
			if err != nil {
				return fmt.Errorf("k8stools.ConvertToUnstructured(): %w", err)
			}
		}
		return nil
	}

	// Prep the initial list prior to any rewriters
	thisun := make([]k8scfg.ProcessedResource, len(uns))
	for i, item := range uns {
		thisun[i] = k8scfg.ProcessedResource{
			Data:            item,
			ResourceOptions: resourceOpt,
		}
	}

	// Perform rewriting on each resource
	thatun := make([]k8scfg.ProcessedResource, 0, len(thisun))
	var rewriteOptCopy RewriteOptions
	for _, rewriter := range rewriters {
		for _, prin := range thisun {
			var un unstructured.Unstructured
			err := keepOrConvert(prin.Data, &un)
			if err != nil {
				return nil, fmt.Errorf("keepOrConvert(): %w", err)
			}

			if un.GetName() == "" {
				return nil, fmt.Errorf("sanity check failed: failed to convert to unstructured")
			}

			rin := k8scfg.Resource{
				Data:            &un,
				ResourceOptions: prin.ResourceOptions,
			}
			rewriteOptCopy = *rewriteOpt
			prouts, err := rewriter(ctx, tools, rin, &rewriteOptCopy)
			if err != nil {
				return nil, fmt.Errorf("rewriter(): %w", err)
			}

			thatun = append(thatun, prouts...)
		}

		if len(thisun) > len(thatun) {
			return nil, fmt.Errorf("sanity check failed: rewriter shrunk the number of records")
		}

		thisun, thatun = thatun, thisun[:0]
	}

	finalun := make([]k8scfg.Resource, len(thisun))
	for i, rout := range thisun {
		var un unstructured.Unstructured
		err := keepOrConvert(rout.Data, &un)
		if err != nil {
			return nil, fmt.Errorf("keepOrConvert() (2): %w", err)
		}

		finalun[i] = k8scfg.Resource{
			Data:            &un,
			ResourceOptions: rout.ResourceOptions,
		}
	}

	return finalun, nil
}
