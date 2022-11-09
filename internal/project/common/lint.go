// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Lint provides common lint target.
type Lint struct { //nolint:govet
	dag.BaseNode

	meta *meta.Options
}

// NewLint initializes Lint.
func NewLint(meta *meta.Options) *Lint {
	return &Lint{
		BaseNode: dag.NewBaseNode("lint"),

		meta: meta,
	}
}

// CompileDrone implements drone.Compiler.
func (lint *Lint) CompileDrone(output *drone.Output) error {
	output.Step(drone.MakeStep("lint").
		DependsOn(dag.GatherMatchingInputNames(lint, dag.Implements[drone.Compiler]())...),
	)

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (lint *Lint) CompileMakefile(output *makefile.Output) error {
	output.Target("lint").Description("Run all linters for the project.").
		Depends(dag.GatherMatchingInputNames(lint, dag.Not(dag.Implements[makefile.SkipAsMakefileDependency]()))...).
		Phony()

	return nil
}
