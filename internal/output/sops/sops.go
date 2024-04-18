// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package sops implements output to .sops.yaml.
package sops

import (
	"fmt"
	"io"

	"github.com/siderolabs/kres/internal/output"
)

// Output provides output to .sops.yaml.
type Output struct {
	output.FileAdapter

	config  string
	enabled bool
}

const (
	filename = ".sops.yaml"
)

// NewOutput initializes Output.
func NewOutput() *Output {
	output := &Output{}

	output.FileWriter = output

	return output
}

// Compile implements [output.TypedWriter] interface.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileSops(o)
}

// Enable should be called to enable config generation.
func (o *Output) Enable() {
	o.enabled = true
}

// Config sets sops configuration.
func (o *Output) Config(config string) {
	o.config = config
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
		return o.sops(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

func (o *Output) sops(w io.Writer) error {
	preamble := output.Preamble("# ")

	if _, err := w.Write([]byte(preamble)); err != nil {
		return fmt.Errorf("failed to write preamble: %w", err)
	}

	if _, err := w.Write([]byte(o.config)); err != nil {
		return err
	}

	return nil
}

// Compiler is implemented by project blocks which support sops config generation.
type Compiler interface {
	CompileSops(*Output) error
}
