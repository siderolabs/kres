// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package gitignore implements output to .gitignore.
package gitignore

import (
	"fmt"
	"io"

	"github.com/siderolabs/kres/internal/output"
)

const (
	filename = ".gitignore"
)

// Output implements .gitignore generation.
type Output struct {
	output.FileAdapter

	ignoredPaths []string
}

// NewOutput creates new Makefile output.
func NewOutput() *Output {
	output := &Output{}

	output.FileAdapter.FileWriter = output

	return output
}

// Compile implements [output.TypedWriter]  interface.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileGitignore(o)
}

// IgnorePath adds paths to the list of ignored by git.
func (o *Output) IgnorePath(paths ...string) {
	o.ignoredPaths = append(o.ignoredPaths, paths...)
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	return []string{filename}
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case filename:
		return o.gitignore(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

func (o *Output) gitignore(w io.Writer) error {
	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	for _, path := range o.ignoredPaths {
		if _, err := fmt.Fprintf(w, "%s\n", path); err != nil {
			return err
		}
	}

	return nil
}

// Compiler is implemented by project blocks which support Dockerfile generate.
type Compiler interface {
	CompileGitignore(*Output) error
}
