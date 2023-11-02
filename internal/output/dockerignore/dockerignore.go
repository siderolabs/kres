// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package dockerignore implements output to .dockerignore.
package dockerignore

import (
	"fmt"
	"io"

	"github.com/siderolabs/kres/internal/output"
)

const (
	filename = ".dockerignore"
)

// Output implements .dockerignore generation.
type Output struct {
	output.FileAdapter

	allowedLocalPaths []string
}

// NewOutput creates new dockerignore output.
func NewOutput() *Output {
	output := &Output{}

	output.FileWriter = output

	return output
}

// Compile implements [output.TypedWriter] interface.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileDockerignore(o)
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	return []string{filename}
}

// AllowLocalPath adds path to the list of paths to be copied into the context.
func (o *Output) AllowLocalPath(paths ...string) *Output {
	o.allowedLocalPaths = append(o.allowedLocalPaths, paths...)

	return o
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case filename:
		return o.dockerignore(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

func (o *Output) dockerignore(w io.Writer) error {
	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "*"); err != nil {
		return err
	}

	for _, path := range o.allowedLocalPaths {
		if _, err := fmt.Fprintf(w, "!%s\n", path); err != nil {
			return err
		}
	}

	return nil
}

// Compiler is implemented by project blocks which support Dockerfile generate.
type Compiler interface {
	CompileDockerignore(*Output) error
}
