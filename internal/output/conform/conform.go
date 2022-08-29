// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package conform implements output to .conform.yaml.
package conform

import (
	_ "embed"
	"encoding/json"
	"io"
	"text/template"

	"github.com/siderolabs/kres/internal/output"
)

const (
	filename = ".conform.yaml"
)

//go:embed conform.yaml
var configTemplate string

// Output implements .conform.yaml generation.
type Output struct {
	output.FileAdapter

	githubOrg         string
	licenseHeader     string
	scopes            []string
	types             []string
	licenseCheck      bool
	gpgSignatureCheck bool

	enabled bool
}

// NewOutput creates new conform.yaml output.
func NewOutput() *Output {
	output := &Output{}

	output.FileAdapter.FileWriter = output

	return output
}

// Compile implements output.Writer interface.
func (o *Output) Compile(node interface{}) error {
	compiler, implements := node.(Compiler)

	if !implements {
		return nil
	}

	return compiler.CompileConform(o)
}

// Enable should be called to enable config generation.
func (o *Output) Enable() {
	o.enabled = true
}

// SetScopes sets the conventional commit scopes.
func (o *Output) SetScopes(scopes []string) {
	o.scopes = scopes
}

// SetTypes sets the conventional commit types.
func (o *Output) SetTypes(types []string) {
	o.types = types
}

// SetLicenseCheck enables license check.
func (o *Output) SetLicenseCheck(enable bool) {
	o.licenseCheck = enable
}

// SetLicenseHeader configures license header.
func (o *Output) SetLicenseHeader(header string) {
	o.licenseHeader = header
}

// SetGPGSignatureCheck enables GPG signature check.
func (o *Output) SetGPGSignatureCheck(enable bool) {
	o.gpgSignatureCheck = enable
}

// SetGitHubOrganization scopes GPG identity check to the GitHub organization.
func (o *Output) SetGitHubOrganization(org string) {
	o.githubOrg = org
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	if !o.enabled {
		return nil
	}

	return []string{filename}
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case filename:
		return o.config(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

func (o *Output) config(w io.Writer) error {
	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	tmpl, err := template.New("config").Parse(configTemplate)
	if err != nil {
		return err
	}

	types, err := json.Marshal(o.types)
	if err != nil {
		return err
	}

	scopes, err := json.Marshal(o.scopes)
	if err != nil {
		return err
	}

	vars := struct {
		Types                   string
		Scopes                  string
		Organization            string
		LicenseHeader           string
		EnableLicenseCheck      bool
		EnableGPGSignatureCheck bool
	}{
		Types:                   string(types),
		Scopes:                  string(scopes),
		Organization:            o.githubOrg,
		EnableLicenseCheck:      o.licenseCheck,
		LicenseHeader:           o.licenseHeader,
		EnableGPGSignatureCheck: o.gpgSignatureCheck,
	}

	return tmpl.Execute(w, vars)
}

// Compiler is implemented by project blocks which support .conform.yaml generate.
type Compiler interface {
	CompileConform(*Output) error
}
