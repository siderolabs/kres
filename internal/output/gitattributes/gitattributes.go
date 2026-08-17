// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package gitattributes implements output to .gitattributes.
package gitattributes

import (
	"fmt"
	"io"
	"slices"

	"github.com/siderolabs/kres/internal/output"
)

const (
	configFile = ".gitattributes"
)

// Output implements .gitattributes generation.
type Output struct {
	output.FileAdapter

	additionalContent []string
	generatedPaths    []string
}

// NewOutput creates new .gitattributes output.
func NewOutput() *Output {
	output := &Output{}

	output.FileWriter = output

	return output
}

// Compile implements [output.TypedWriter] interface.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileGitattributes(o)
}

// AddContent adds additional lines to be included in the generated .gitattributes file,
// e.g. "* text=auto eol=lf".
func (o *Output) AddContent(lines ...string) {
	o.additionalContent = append(o.additionalContent, lines...)
}

// MarkGenerated marks paths as generated so that GitHub treats them as generated files.
//
// See https://docs.github.com/en/repositories/working-with-files/managing-files/customizing-how-changed-files-appear-on-github.
func (o *Output) MarkGenerated(paths ...string) {
	o.generatedPaths = append(o.generatedPaths, paths...)
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	return []string{configFile}
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case configFile:
		return o.gitattributes(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

func (o *Output) gitattributes(w io.Writer) error {
	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	for _, line := range o.additionalContent {
		if _, err := fmt.Fprintf(w, "%s\n", line); err != nil {
			return err
		}
	}

	paths := slices.Clone(o.generatedPaths)
	slices.Sort(paths)
	paths = slices.Compact(paths)

	for _, path := range paths {
		if _, err := fmt.Fprintf(w, "%s linguist-generated=true\n", path); err != nil {
			return err
		}
	}

	return nil
}

// Compiler is implemented by project blocks which support .gitattributes generate.
type Compiler interface {
	CompileGitattributes(*Output) error
}
