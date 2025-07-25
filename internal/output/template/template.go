// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package template implements custom template output.
package template

import (
	"fmt"
	"io"
	"os"
	"slices"
	"text/template"

	"github.com/siderolabs/kres/internal/output"
	"github.com/siderolabs/kres/internal/project/common"
)

// FileTemplate defines a single file template to be generated by this output.
type FileTemplate struct {
	params          any
	preamblePrefix  string
	name            string
	template        string
	withLicenseText string
	withLicense     bool
	withPreamble    bool
	noOverwrite     bool
}

// PreamblePrefix sets preamble prefix.
func (t *FileTemplate) PreamblePrefix(value string) *FileTemplate {
	t.preamblePrefix = value

	return t
}

// WithLicense prepends the license text before the preamble.
func (t *FileTemplate) WithLicense() *FileTemplate {
	t.withLicense = true

	return t
}

// WithLicenseText sets the license text.
//
// If unset and WithLicense is requested, the default MPL license is used.
func (t *FileTemplate) WithLicenseText(value string) *FileTemplate {
	t.withLicenseText = value

	return t
}

// NoPreamble disables preamble geneneration.
func (t *FileTemplate) NoPreamble() *FileTemplate {
	t.withPreamble = false

	return t
}

// NoOverwrite generates the template only if it doesn't exist yet.
func (t *FileTemplate) NoOverwrite() *FileTemplate {
	t.noOverwrite = true

	return t
}

// Params sets template params.
func (t *FileTemplate) Params(value any) *FileTemplate {
	t.params = value

	return t
}

func (t *FileTemplate) write(w io.Writer) error {
	if t.noOverwrite {
		_, err := os.Stat(t.name)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		} else {
			return output.ErrSkip
		}
	}

	if t.withLicense {
		licenseText := t.withLicenseText
		if licenseText == "" {
			licenseText = common.MPLHeader
		}

		if _, err := w.Write([]byte(output.License(licenseText, t.preamblePrefix) + "\n")); err != nil {
			return err
		}
	}

	if t.withPreamble {
		if _, err := w.Write([]byte(output.Preamble(t.preamblePrefix))); err != nil {
			return err
		}
	}

	tmpl, err := template.New(t.name).Parse(t.template)
	if err != nil {
		return err
	}

	err = tmpl.Execute(w, t.params)
	if err != nil {
		return fmt.Errorf("failed to execute custom template '%s': %w", t.name, err)
	}

	return nil
}

// Output implements custom templates generation.
type Output struct {
	output.FileAdapter

	templates map[string]*FileTemplate
}

// NewOutput creates new Makefile output.
func NewOutput() *Output {
	output := &Output{
		templates: map[string]*FileTemplate{},
	}

	output.FileWriter = output

	return output
}

// Compile implements [output.TypedWriter] interface.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileTemplates(o)
}

// Define should be called to add a templated file output.
func (o *Output) Define(name, template string) *FileTemplate {
	o.templates[name] = &FileTemplate{
		name:         name,
		template:     template,
		withPreamble: true,
	}

	return o.templates[name]
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	files := make([]string, len(o.templates))
	index := -1

	for name := range o.templates {
		index++

		files[index] = name
	}

	slices.Sort(files)

	return files
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	if t, ok := o.templates[filename]; ok {
		if t.name == filename {
			return t.write(w)
		}
	}

	return fmt.Errorf("unexpected file name %s", filename)
}

// Compiler is implemented by project blocks which support template compile.
type Compiler interface {
	CompileTemplates(*Output) error
}
