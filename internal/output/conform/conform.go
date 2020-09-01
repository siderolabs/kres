// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package conform implements output to .conform.yml.
package conform

import (
	"encoding/json"
	"io"
	"text/template"

	"github.com/talos-systems/kres/internal/output"
)

const (
	filename = ".conform.yaml"
)

// Output implements .conform.yaml generation.
type Output struct {
	output.FileAdapter

	enabled bool

	scopes []string
	types  []string
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
		Types  string
		Scopes string
	}{
		Types:  string(types),
		Scopes: string(scopes),
	}

	if err = tmpl.Execute(w, vars); err != nil {
		return err
	}

	return nil
}

// Compiler is implemented by project blocks which support .conform.yaml generate.
type Compiler interface {
	CompileConform(*Output) error
}
