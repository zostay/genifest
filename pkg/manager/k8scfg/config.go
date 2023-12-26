package k8scfg

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/zostay/genifest/pkg/log"
	cfgstr "github.com/zostay/genifest/pkg/strtools"
)

const TrashDir = "TRASH"

var PhasePrefixes = []string{"storageclass", "namespace", "addon"} // phases that need to run first in this order

// ConfigFiles returns the names of all the Kubernetes configuration files that
// match the given glob pattern.
func ConfigFiles(
	cloudHome,
	kubeDir string,
	excludeMatches []string,
	match string,
	remove bool,
) ([]string, error) {
	var kubeRoot string
	if filepath.IsAbs(kubeDir) {
		kubeRoot = kubeDir
	} else {
		kubeRoot = filepath.Join(cloudHome, kubeDir)
	}

	if remove {
		// when removing, we only want the trash
		kubeRoot = filepath.Join(kubeRoot, TrashDir)
	}

	match = cfgstr.MakeMatch(match)

	configFiles := make([]string, 0)
	err := filepath.WalkDir(kubeRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error reading %q: %w", path, err)
		}

		if d.IsDir() {
			// skip the trash directory unless removing
			if !remove && filepath.Base(path) == TrashDir {
				return fs.SkipDir
			}

			return nil
		}

		rel, err := filepath.Rel(kubeRoot, path)
		if err != nil {
			return err
		}

		// skip not_resource_files matches
		for _, m := range excludeMatches {
			matched, err := doublestar.Match(m, rel)
			if err != nil {
				return err
			}

			if matched {
				log.Linef("SKIP", "Skipping resource file %q", rel)
				return nil
			}
		}

		matched, err := doublestar.Match(match, rel)
		if err != nil {
			return err
		}

		if !matched {
			return nil
		}

		configFiles = append(configFiles, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	cfps := make([]struct {
		cf string
		p  int
	}, len(configFiles))
	for i, cf := range configFiles {
		phase := len(PhasePrefixes)
		bn := filepath.Base(cf)
		for p, pre := range PhasePrefixes {
			if strings.HasPrefix(bn, pre) {
				phase = p
				break
			}
		}
		cfps[i] = struct {
			cf string
			p  int
		}{cf, phase}
	}

	sort.Slice(cfps, func(a, b int) bool {
		if cfps[a].p != cfps[b].p {
			return cfps[a].p < cfps[b].p
		} else {
			return cfps[a].cf < cfps[b].cf
		}
	})

	cfs := make([]string, len(cfps))
	for i, cfp := range cfps {
		cfs[i] = cfp.cf
	}

	return cfs, nil
}
