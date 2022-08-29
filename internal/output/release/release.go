// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package release implements output for releases.
package release

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/siderolabs/kres/internal/output"
	"github.com/siderolabs/kres/internal/project/meta"
)

const (
	releaseScript   = "./hack/release.sh"
	releaseTemplate = "./hack/release.toml"
)

//go:embed release.sh
var releaseScriptStr string

//go:embed release.toml
var releaseTemplateStr string

// Output implements .gitignore generation.
type Output struct {
	output.FileAdapter

	meta *meta.Options
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

	return compiler.CompileRelease(o)
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	return []string{releaseScript, releaseTemplate}
}

// SetMeta grabs build options.
func (o *Output) SetMeta(meta *meta.Options) {
	o.meta = meta
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case releaseScript:
		return o.releaseScript(w)
	case releaseTemplate:
		return o.releaseTemplate(filename, w)
	default:
		panic("unexpected filename: " + filename)
	}
}

// Permissions implements output.PermissionsWriter interface.
func (o *Output) Permissions(filename string) os.FileMode {
	if filename == releaseScript {
		return 0o744
	}

	return 0
}

func (o *Output) releaseScript(w io.Writer) error {
	if _, err := w.Write([]byte("#!/bin/bash\n\n")); err != nil {
		return err
	}

	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s\n", releaseScriptStr); err != nil {
		return err
	}

	return nil
}

func (o *Output) releaseTemplate(filename string, w io.Writer) error {
	_, err := os.Stat(filename)

	if err == nil {
		return output.ErrSkip
	}

	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// no preamble as this file is meant to be edited

	tmpl, err := template.New("config").Parse(strings.TrimSpace(releaseTemplateStr) + "\n")
	if err != nil {
		return err
	}

	return tmpl.Execute(w, o.meta)
}

// Compiler is implemented by project blocks which support Dockerfile generate.
type Compiler interface {
	CompileRelease(*Output) error
}
