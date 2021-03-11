// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package custom provides blocks defined in the config manually.
package custom

import (
	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/output/drone"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/project/meta"
)

// Step is defined in the config manually.
type Step struct {
	dag.BaseNode

	meta *meta.Options

	Makefile struct { //nolint: govet
		Enabled bool     `yaml:"enabled"`
		Phony   bool     `yaml:"phony"`
		Depends []string `yaml:"depends"`
		Script  []string `yaml:"script"`

		Variables []struct {
			Name         string `yaml:"name"`
			DefaultValue string `yaml:"defaultValue"`
		} `yaml:"variables"`
	} `yaml:"makefile"`

	Drone struct {
		Enabled    bool `yaml:"enabled"`
		Privileged bool `yaml:"privileged"`
	} `yaml:"drone"`
}

// NewStep initializes Step.
func NewStep(meta *meta.Options, name string) *Step {
	return &Step{
		BaseNode: dag.NewBaseNode(name),
		meta:     meta,
	}
}

// CompileDockerfile implements dockerfile.Compiler.
func (step *Step) CompileDockerfile(output *dockerfile.Output) error {
	return nil
}

// CompileDrone implements drone.Compiler.
func (step *Step) CompileDrone(output *drone.Output) error {
	if !step.Drone.Enabled {
		return nil
	}

	droneMatches := func(node dag.Node) bool {
		if !dag.Implements((*drone.Compiler)(nil))(node) {
			return false
		}

		if nodeStep, ok := node.(*Step); ok {
			return nodeStep.Drone.Enabled
		}

		return true
	}

	droneStep := drone.MakeStep(step.Name()).
		DependsOn(dag.GatherMatchingInputNames(step, droneMatches)...)

	if step.Drone.Privileged {
		droneStep.Privileged()
	}

	output.Step(droneStep)

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (step *Step) CompileMakefile(output *makefile.Output) error {
	if !step.Makefile.Enabled {
		return nil
	}

	for _, variable := range step.Makefile.Variables {
		output.VariableGroup(makefile.VariableGroupExtra).
			Variable(makefile.OverridableVariable(variable.Name, variable.DefaultValue))
	}

	target := output.Target(step.Name()).
		Depends(step.Makefile.Depends...).
		Script(step.Makefile.Script...)

	if step.Makefile.Phony {
		target.Phony()
	}

	return nil
}
