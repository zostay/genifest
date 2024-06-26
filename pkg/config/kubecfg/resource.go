package kubecfg

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"text/template"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/zostay/genifest/pkg/log"
)

// Client grants access to locate, read, and process k8s resource/manifest
// files.
type Client struct {
	cloudHome string
	funcMap   template.FuncMap
}

// ResourceOptions encapsulates operational options associated with a resource
// that modify how these libraries work with it.
type ResourceOptions struct {
	Validate     bool // Set by the ZZZ --validate option.
	NeedsRestart bool // Can be set by rewriters to trigger a rollout restart on a deployment
}

// RawResource encapsulates a configuration file with its ResourceOptions.
type RawResource struct {
	Config []byte // The content of the configuration to apply.

	ResourceOptions
}

// ProcessedResource represents any parsed typed or unstructured kubernetes
// resource that can be applied via SSA patch after being converted to
// unstructured with its ResourceOptions.
type ProcessedResource struct {
	Data interface{}

	ResourceOptions
}

// Resource represents a parsed and unstructured kubernetes resource that can be
// applied via SSA patch with its ResourceOptions.
type Resource struct {
	Data *unstructured.Unstructured // The templated, parsed, and rewritten resource.

	ResourceOptions
}

// ProcessedResource converts the Resource to a ProcessedResource without making
// any changes.
func (r *Resource) ProcessedResource() ProcessedResource {
	return ProcessedResource{
		Data:            r.Data,
		ResourceOptions: r.ResourceOptions,
	}
}

var (
	emptyLine = regexp.MustCompile(`^\s*(?:#.*)?$`)
)

// New returns a client for reading kubernetes configuration files and
// processing them.
func New(cloudHome string, funcMap template.FuncMap) *Client {
	return &Client{cloudHome, funcMap}
}

// SetFunc modifies the function map associated with the Client to replace or
// add another function to it.
func (c *Client) SetFunc(
	name string,
	f any,
) {
	c.funcMap[name] = f
}

// ReadResourceFile reads a resource file and breaks it into parts by the triple
// hyphen separator. The parts are raw bytes. They may be in need of templating
// or otherwise incomplete. Empty parts (those consisting entirely of blank
// lines or comments) will be removed.
func (c *Client) ReadResourceFile(rfile string) ([]RawResource, error) {
	configPath := rfile
	if !filepath.IsAbs(rfile) {
		configPath = filepath.Join(c.cloudHome, rfile)
	}

	res, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	sres := bytes.Split(res, []byte("\n---"))
	fres := make([]RawResource, 0, len(sres))
	for _, s := range sres {
		scanner := bufio.NewScanner(bytes.NewReader(s))
		hasContent := false
		for scanner.Scan() {
			t := scanner.Text()
			if !emptyLine.MatchString(t) {
				hasContent = true
				break
			}
		}

		// skip empty sections
		if !hasContent {
			log.Linef("SKIP SECTION", string(s))
			continue
		}

		f := bytes.TrimSpace(s)
		if len(f) == 0 {
			continue
		}

		fo := RawResource{f, ResourceOptions{true, false}}
		fres = append(fres, fo)
	}

	return fres, nil
}

// ParseResource parses a single resource. This MUST already be broken up from
// other files using a triple hyphen separator.
func ParseResource(data []byte) (*unstructured.Unstructured, error) {
	r := bytes.NewReader(data)
	dec := yaml.NewYAMLOrJSONDecoder(r, 4096)

	var uns unstructured.Unstructured
	err := dec.Decode(&uns)

	return &uns, err
}

// WriteResourceFile writes out a resource to a configuration file.
func (c *Client) WriteResourceFile(
	wfile string,
	bs []byte,
) error {
	configPath := wfile
	if !filepath.IsAbs(wfile) {
		configPath = filepath.Join(c.cloudHome, wfile)
	}

	configDir := filepath.Dir(configPath)

	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		return fmt.Errorf("os.MkdirAll(%q): %w", configDir, err)
	}

	err = os.WriteFile(configPath, bs, 0644) //nolint:gosec // 0644 is fine
	if err != nil {
		return fmt.Errorf("os.WriteFile(%q): %w", configPath, err)
	}

	return nil
}
