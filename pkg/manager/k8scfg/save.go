package k8scfg

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/zostay/genifest/pkg/client/k8s"
)

// SaveResourceFile turns a serialized resource into a resource file mounted in
// the given save directory.
func SaveResourceFile(
	ctx context.Context,
	tools Tools,
	saveDir string,
	sr *k8s.SerializedResource,
) error {
	c, err := tools.ResMgr(ctx)
	if err != nil {
		return fmt.Errorf("tools.ResMgr(): %w", err)
	}

	wfile := filepath.Join(saveDir, sr.ResourceID()) + ".yaml"

	err = c.WriteResourceFile(wfile, sr.Bytes())
	if err != nil {
		return fmt.Errorf("c.WriteResourceFile(%q): %w", wfile, err)
	}

	return nil
}
