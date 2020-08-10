// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package golangci implements output to .golangci.yml.
package golangci

import (
	"fmt"
	"io"

	"github.com/talos-systems/kres/internal/output"
)

const (
	filename = ".golangci.yml"
)

// Output implements .golangci.yml generation.
type Output struct {
	output.FileAdapter

	enabled       bool
	canonicalPath string
}

// NewOutput creates new Makefile output.
func NewOutput() *Output {
	output := &Output{
		canonicalPath: "github.com/example.com/example.proj",
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

	return compiler.CompileGolangci(o)
}

// Enable should be called to enable config generation.
func (o *Output) Enable() {
	o.enabled = true
}

// CanonicalPath sets canonical import path.
func (o *Output) CanonicalPath(path string) {
	o.canonicalPath = path
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

	if _, err := fmt.Fprintf(w, config, o.canonicalPath); err != nil {
		return err
	}

	return nil
}

// Compiler is implemented by project blocks which support Dockerfile generate.
type Compiler interface {
	CompileGolangci(*Output) error
}
