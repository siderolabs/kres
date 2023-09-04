// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package golangci implements output to .golangci.yml.
package golangci

import (
	_ "embed"
	"fmt"
	"io"
	"path/filepath"
	"slices"

	"github.com/siderolabs/gen/xslices"

	"github.com/siderolabs/kres/internal/output"
)

const (
	filename = ".golangci.yml"
)

//go:embed golangci.yml
var configTemplate string

// Output implements .golangci.yml generation.
type Output struct {
	output.FileAdapter

	files   []file
	enabled bool
}

type file struct {
	path          string
	canonicalPath string
}

// NewOutput creates new Makefile output.
func NewOutput() *Output {
	output := &Output{}

	output.FileAdapter.FileWriter = output

	return output
}

// Compile implements [output.TypedWriter] interface.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileGolangci(o)
}

// Enable should be called to enable config generation.
func (o *Output) Enable() {
	o.enabled = true
}

// NewFile sets canonical import path and project path.
func (o *Output) NewFile(canonicalPath, path string) {
	o.files = append(o.files, file{
		path:          filepath.Join(path, filename),
		canonicalPath: canonicalPath,
	})
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	if !o.enabled {
		return nil
	}

	return xslices.Map(o.files, func(f file) string { return f.path })
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	if index := slices.IndexFunc(o.files, func(f file) bool { return f.path == filename }); index >= 0 {
		return o.config(o.files[index], w)
	}

	panic("unexpected filename: " + filename)
}

func (o *Output) config(f file, w io.Writer) error {
	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, configTemplate, f.canonicalPath); err != nil {
		return err
	}

	return nil
}

// Compiler is implemented by project blocks which support Dockerfile generate.
type Compiler interface {
	CompileGolangci(*Output) error
}
