package tmpltools

import (
	"os"
	"path/filepath"

	"github.com/zostay/genifest/pkg/log"
)

func File(cloudHome, app, path string) (string, error) {
	p := filepath.Join(cloudHome, "files", app+path)
	data, err := os.ReadFile(p)
	log.LineBytes("EMBED", data)
	if err != nil {
		return "", err
	}
	return string(data), err
}
