// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package markdownlint implements output to .markdownlint.json.
package markdownlint

import (
	"encoding/json"
	"io"

	"github.com/talos-systems/kres/internal/output"
)

const (
	filename = ".markdownlint.json"
)

// Output implements .markdownlint.json generation.
type Output struct {
	output.FileAdapter

	enabled bool
	rules   map[string]bool
}

// NewOutput creates new Makefile output.
func NewOutput() *Output {
	output := &Output{
		rules: map[string]bool{
			"default": true,
			"MD013":   false,
			"MD033":   false,
		},
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

	return compiler.CompileMarkdownLint(o)
}

// Enable should be called to enable config generation.
func (o *Output) Enable() {
	o.enabled = true
}

// Rules sets linting rules.
func (o *Output) Rules(rules map[string]bool) {
	o.rules = rules
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

	enc := json.NewEncoder(w)
	enc.SetIndent("  ", "  ")

	return enc.Encode(o.rules)
}

// Compiler is implemented by project blocks which support Dockerfile generate.
type Compiler interface {
	CompileMarkdownLint(*Output) error
}
