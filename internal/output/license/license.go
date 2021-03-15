// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package license implements output to LICENSE.
package license

import (
	"io"

	"github.com/talos-systems/kres/internal/output"
)

const (
	filename = "LICENSE"
)

// Output implements LICENSE generation.
type Output struct {
	output.FileAdapter

	enabled bool
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

// Enable should be called to enable config generation.
func (o *Output) Enable() {
	o.enabled = true
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
		return o.license(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

func (o *Output) license(w io.Writer) error {
	if _, err := w.Write([]byte(mpl20)); err != nil {
		return err
	}

	return nil
}

// Compiler is implemented by project blocks which support LICENSE generation.
type Compiler interface {
	CompileLicense(*Output) error
}
