package kubecfg

import (
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// TODO Look into minimizing or eliminating the need for templating here. We may
// be able to incorporate kustomize to do much of it and specialized annotations
// to do the rest.

// TemplateConfigFile takes the given template string and templates teh file as
// a configuration. It returns the output of the templating.
func (c *Client) TemplateConfigFile(name string, data []byte) (string, error) {
	tmpl := template.New(name)
	tmpl.Delims("{{{", "}}}")
	tmpl.Funcs(c.funcMap)
	tmpl.Funcs(sprig.TxtFuncMap())
	_, err := tmpl.Parse(string(data))
	if err != nil {
		return "", err
	}

	res := new(strings.Builder)
	err = tmpl.Execute(res, nil)
	if err != nil {
		return "", err
	}

	return res.String(), err
}
