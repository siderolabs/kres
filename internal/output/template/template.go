// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package template implements custom template output.
package template

import (
	"fmt"
	"io"
	"sort"
	"text/template"

	"github.com/talos-systems/kres/internal/output"
)

// FileTemplate defines a single file template to be generated by this output.
type FileTemplate struct {
	params         interface{}
	preamblePrefix string
	name           string
	template       string
	withPreamble   bool
	withLicense    bool
}

// PreamblePrefix sets preamble prefix.
func (t *FileTemplate) PreamblePrefix(value string) *FileTemplate {
	t.preamblePrefix = value

	return t
}

// WithLicense prepends MPL license before the preable.
func (t *FileTemplate) WithLicense() *FileTemplate {
	t.withLicense = true

	return t
}

// NoPreamble disables preamble geneneration.
func (t *FileTemplate) NoPreamble() *FileTemplate {
	t.withPreamble = false

	return t
}

// Params sets template params.
func (t *FileTemplate) Params(value interface{}) *FileTemplate {
	t.params = value

	return t
}

func (t *FileTemplate) write(w io.Writer) error {
	if t.withLicense {
		if _, err := w.Write([]byte(output.License(t.preamblePrefix))); err != nil {
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

	return tmpl.Execute(w, t.params)
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

	output.FileAdapter.FileWriter = output

	return output
}

// Compile implements output.Writer interface.
func (o *Output) Compile(node interface{}) error {
	compiler, implements := node.(Compiler)

	if !implements {
		return nil
	}

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

	sort.Strings(files)

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