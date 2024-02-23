// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package license implements output to LICENSE.
package license

import (
	_ "embed"
	"fmt"
	"io"
	"path/filepath"
	"text/template"

	"github.com/siderolabs/kres/internal/output"
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

type dirConfig struct {
	templateParams  any
	licenseTemplate string
	licenseHeader   string
}

// Output implements LICENSE generation.
type Output struct {
	output.FileAdapter

	dirConfigs map[string]*dirConfig
}

// NewOutput creates new Makefile output.
func NewOutput() *Output {
	output := &Output{
		dirConfigs: map[string]*dirConfig{},
	}

	output.FileWriter = output

	return output
}

// Compile implements [output.TypedWriter]  interface.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileLicense(o)
}

// SetLicenseHeader configures license header.
func (o *Output) SetLicenseHeader(dir, header string) {
	o.getDirConfig(dir).licenseHeader = header
}

// Enable should be called to enable config generation.
func (o *Output) Enable(dir, licenseID string, params any) error {
	config := o.getDirConfig(dir)

	var ok bool

	config.licenseTemplate, ok = licenseTemplates[licenseID]
	if !ok {
		return fmt.Errorf("unsupported license %q: missing LICENSE template", licenseID)
	}

	config.templateParams = params

	return nil
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	filenames := make([]string, 0, len(o.dirConfigs)*2)

	for dir, config := range o.dirConfigs {
		if config.licenseTemplate == "" {
			continue
		}

		filenames = append(filenames, filepath.Join(dir, filename), filepath.Join(dir, Header))
	}

	return filenames
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(path string, w io.Writer) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	switch base {
	case filename:
		return o.license(dir, w)
	case Header:
		return o.boilerplate(dir, w)
	default:
		panic("unexpected filename: " + filename)
	}
}

func (o *Output) license(dir string, w io.Writer) error {
	config := o.getDirConfig(dir)

	tmpl, err := template.New("license").Parse(config.licenseTemplate)
	if err != nil {
		return err
	}

	err = tmpl.Execute(w, config.templateParams)
	if err != nil {
		return fmt.Errorf("failed to execute license template: %w", err)
	}

	return nil
}

func (o *Output) boilerplate(dir string, w io.Writer) error {
	config := o.getDirConfig(dir)

	if config.licenseHeader == "" {
		return nil
	}

	tmpl, err := template.New("licenseHeader").Parse(config.licenseHeader)
	if err != nil {
		return err
	}

	err = tmpl.Execute(w, config.templateParams)
	if err != nil {
		return fmt.Errorf("failed to execute license header template: %w", err)
	}

	return nil
}

func (o *Output) getDirConfig(dir string) *dirConfig {
	if dir == "" {
		dir = "."
	}

	config, ok := o.dirConfigs[dir]
	if !ok {
		config = &dirConfig{}
		o.dirConfigs[dir] = config
	}

	return config
}

// Compiler is implemented by project blocks which support LICENSE generation.
type Compiler interface {
	CompileLicense(*Output) error
}
