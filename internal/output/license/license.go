// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package license implements output to LICENSE.
package license

import (
	_ "embed"
	"fmt"
	"io"
	"text/template"

	"github.com/talos-systems/kres/internal/output"
)

const (
	filename = "LICENSE"
	// Header file path.
	Header = ".license-header.go.txt"
)

//go:embed MPL-2.0.txt
var mpl2 string

//go:embed BSL-1.1.txt
var bsl11 string

var licenseTemplates = map[string]string{
	"MPL-2.0": mpl2,
	"BSL-1.1": bsl11,
}

// Output implements LICENSE generation.
type Output struct {
	output.FileAdapter

	templateParams  interface{}
	licenseTemplate string
	licenseHeader   string
}

// NewOutput creates new Makefile output.
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

	return compiler.CompileLicense(o)
}

// SetLicenseHeader configures license header.
func (o *Output) SetLicenseHeader(header string) {
	o.licenseHeader = header
}

// Enable should be called to enable config generation.
func (o *Output) Enable(licenseID string, params interface{}) error {
	var ok bool

	o.licenseTemplate, ok = licenseTemplates[licenseID]
	if !ok {
		return fmt.Errorf("unsupported license %q: missing LICENSE template", licenseID)
	}

	o.templateParams = params

	return nil
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	if o.licenseTemplate == "" {
		return nil
	}

	return []string{filename, Header}
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(path string, w io.Writer) error {
	switch path {
	case filename:
		return o.license(w)
	case Header:
		return o.boilerplate(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

func (o *Output) license(w io.Writer) error {
	tmpl, err := template.New("license").Parse(o.licenseTemplate)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, o.templateParams)
}

func (o *Output) boilerplate(w io.Writer) error {
	if o.licenseHeader == "" {
		return nil
	}

	tmpl, err := template.New("licenseHeader").Parse(o.licenseHeader)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, o.templateParams)
}

// Compiler is implemented by project blocks which support LICENSE generation.
type Compiler interface {
	CompileLicense(*Output) error
}
