// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"slices"

	"github.com/siderolabs/gen/xslices"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
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

// Activate marks the linter as active.
func (lint *Lint) Activate() {
	if !slices.Contains(lint.meta.ExtraEnforcedContexts, "lint") {
		lint.meta.ExtraEnforcedContexts = append(lint.meta.ExtraEnforcedContexts, "lint")
	}
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (lint *Lint) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	output.AddStepInParallelJob(
		"lint",
		ghworkflow.GenericRunner,
		ghworkflow.Step("lint").SetMakeStep("lint"),
	)

	if lint.meta.SOPSEnabled {
		output.AddStepAfter(
			"lint",
			"setup-buildx",
			ghworkflow.SOPSSteps()...,
		)
	}

	return nil
}

// LinterHasFmt is implemented by linters that have a formatting step.
type LinterHasFmt interface {
	LinterHasFmt()
}

// CompileMakefile implements makefile.Compiler.
func (lint *Lint) CompileMakefile(output *makefile.Output) error {
	output.Target("lint").Description("Run all linters for the project.").
		Depends(dag.GatherMatchingInputNames(lint, dag.Not(dag.Implements[makefile.SkipAsMakefileDependency]()))...).
		Phony()

	output.Target("lint-fmt").Description("Run all linter formatters and fix up the source tree.").
		Depends(
			xslices.Map(
				dag.GatherMatchingInputNames(lint, dag.Implements[LinterHasFmt]()),
				func(name string) string {
					return name + "-fmt"
				},
			)...).
		Phony()

	return nil
}
