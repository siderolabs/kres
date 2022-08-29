// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package makefile implements output to Makefiles.
package makefile

import (
	"io"
	"sort"

	"github.com/siderolabs/kres/internal/output"
)

const (
	makefile = "Makefile"
)

// Output implements Makefile generation.
type Output struct {
	output.FileAdapter

	variableGroups     map[string]*VariableGroup
	variableGroupOrder []string

	targets []*Target
}

// NewOutput creates new Makefile output.
func NewOutput() *Output {
	output := &Output{}

	output.FileAdapter.FileWriter = output

	return output
}

// VariableGroup creates new group of variables.
func (o *Output) VariableGroup(description string) *VariableGroup {
	if o.variableGroups == nil {
		o.variableGroups = make(map[string]*VariableGroup)
	}

	if _, ok := o.variableGroups[description]; !ok {
		o.variableGroups[description] = &VariableGroup{
			description: description,
		}

		o.variableGroupOrder = append(o.variableGroupOrder, description)
	}

	return o.variableGroups[description]
}

// Target creates new Makefile target.
func (o *Output) Target(name string) *Target {
	target := &Target{name: name}

	o.targets = append(o.targets, target)

	return target
}

// Compile implements output.Writer interface.
func (o *Output) Compile(node interface{}) error {
	compiler, implements := node.(Compiler)

	if !implements {
		return nil
	}

	return compiler.CompileMakefile(o)
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	return []string{makefile}
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case makefile:
		return o.makefile(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

func (o *Output) makefile(w io.Writer) error {
	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	for _, varGroupName := range o.variableGroupOrder {
		if err := o.variableGroups[varGroupName].Generate(w); err != nil {
			return err
		}
	}

	sort.SliceStable(o.targets, func(i, j int) bool {
		return o.targets[i].name == "all"
	})

	for _, target := range o.targets {
		if err := target.Generate(w); err != nil {
			return err
		}
	}

	return nil
}

// Compiler is implemented by project blocks which support Dockerfile generate.
type Compiler interface {
	CompileMakefile(*Output) error
}

// SkipAsMakefileDependency signals that this node should never be exposed as Makefile dependency.
type SkipAsMakefileDependency interface {
	SkipAsMakefileDependency()
}
