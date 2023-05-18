// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package service

import (
	"fmt"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/codecov"
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// CodeCov provides build step which uploads coverage info to codecov.io.
type CodeCov struct {
	dag.BaseNode

	meta *meta.Options

	InputPaths      []string `yaml:"inputPaths"`
	TargetThreshold int      `yaml:"targetThreshold"`
	Enabled         bool     `yaml:"enabled"`
}

// NewCodeCov initializes CodeCov.
func NewCodeCov(meta *meta.Options) *CodeCov {
	return &CodeCov{
		BaseNode: dag.NewBaseNode("coverage"),

		meta: meta,

		Enabled:         true,
		TargetThreshold: 50,
	}
}

// CompileDrone implements drone.Compiler.
func (coverage *CodeCov) CompileDrone(output *drone.Output) error {
	if !coverage.Enabled {
		return nil
	}

	output.Step(drone.MakeStep("coverage").
		DependsOn(dag.GatherMatchingInputNames(coverage, dag.Implements[drone.Compiler]())...).
		EnvironmentFromSecret("CODECOV_TOKEN", "CODECOV_TOKEN"),
	)

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (coverage *CodeCov) CompileMakefile(output *makefile.Output) error {
	if !coverage.Enabled {
		return nil
	}

	target := output.Target("coverage").Description("Upload coverage data to codecov.io.")

	for _, inputPath := range coverage.InputPaths {
		target.Script(fmt.Sprintf(`bash -c "bash <(curl -s https://codecov.io/bash) -f $(ARTIFACTS)/%s -X fix"`, inputPath)).
			Phony()
	}

	return nil
}

// CompileCodeCov implements codecov.Compiler.
func (coverage *CodeCov) CompileCodeCov(output *codecov.Output) error {
	if !coverage.Enabled {
		return nil
	}

	output.Enable()
	output.Target(coverage.TargetThreshold)

	return nil
}

// SkipAsMakefileDependency implements makefile.SkipAsMakefileDependency.
func (coverage *CodeCov) SkipAsMakefileDependency() {
}
