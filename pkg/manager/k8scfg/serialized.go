package k8scfg

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/zostay/genifest/pkg/client/k8s"
)

// SerializeResource turns a resource into JSON bytes ready for application. This
// works similar to the SerializeResource method of the k8s client, but does not
// need to talk to the cluster to work. This returns a SerializedResource, but
// the dynamic resource interface will not be set.
func SerializeResource(
	un *unstructured.Unstructured,
) (*k8s.SerializedResource, error) {
	data, err := json.Marshal(un)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal(): %w", err)
	}

	return k8s.NewSerializedResource(un, nil, data), nil
}
