// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package codecov implements output to .codecov.yml.
package codecov

import (
	_ "embed"
	"fmt"
	"io"

	"github.com/siderolabs/kres/internal/output"
)

const (
	filename = ".codecov.yml"
)

//go:embed codecov.yml
var configTemplate string

// Output implements .codecov.yml generation.
type Output struct {
	output.FileAdapter

	enabled bool
	target  int
}

// NewOutput creates new codecov.yml output.
func NewOutput() *Output {
	output := &Output{
		target: 50,
	}

	output.FileWriter = output

	return output
}

// Compile implements output.Writer interface.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileCodeCov(o)
}

// Enable should be called to enable config generation.
func (o *Output) Enable() {
	o.enabled = true
}

// Target sets target coverage threshold (percent).
func (o *Output) Target(threshold int) {
	o.target = threshold
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

	if _, err := fmt.Fprintf(w, configTemplate, o.target); err != nil {
		return err
	}

	return nil
}

// Compiler is implemented by project blocks which support .codecov.yml generate.
type Compiler interface {
	CompileCodeCov(*Output) error
}
