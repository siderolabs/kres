// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package lefthook implements output to lefthook.yml.
package lefthook

import (
	"fmt"
	"io"

	"go.yaml.in/yaml/v4"

	"github.com/siderolabs/kres/internal/output"
)

const configFile = "lefthook.yml"

// Output implements lefthook.yml generation.
type Output struct {
	output.FileAdapter

	hooks map[string]*Hook

	enabled bool
}

// NewOutput creates new lefthook.yml output.
func NewOutput() *Output {
	o := &Output{
		hooks: map[string]*Hook{},
	}

	o.FileWriter = o

	return o
}

// Compile implements [output.TypedWriter] interface.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileLefthook(o)
}

// Enable should be called to enable config generation.
func (o *Output) Enable() {
	o.enabled = true
}

// Hook returns the configuration for the named git hook (e.g. "pre-commit",
// "commit-msg"), creating it on first access. New hooks are blank — set the
// execution model via WithParallel/WithPiped, and populate either Commands
// or Jobs depending on which lefthook style you want.
func (o *Output) Hook(name string) *Hook {
	if h, ok := o.hooks[name]; ok {
		return h
	}

	h := &Hook{}
	o.hooks[name] = h

	return h
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	if !o.enabled {
		return nil
	}

	return []string{configFile}
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case configFile:
		return o.config(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

func (o *Output) config(w io.Writer) error {
	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	encoder := yaml.NewEncoder(w)
	defer encoder.Close() // nolint:errcheck

	encoder.SetIndent(2)

	if err := encoder.Encode(o.hooks); err != nil {
		return fmt.Errorf("failed to encode lefthook config: %w", err)
	}

	return nil
}

// Compiler is implemented by project blocks which support lefthook.yml generation.
type Compiler interface {
	CompileLefthook(*Output) error
}
