// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package service

import (
	"fmt"
	"strings"

	"github.com/siderolabs/gen/xslices"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/codecov"
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

type dependentJobs struct {
	name  string
	flags string
}

// CodeCov provides build step which uploads coverage info to codecov.io.
type CodeCov struct {
	dag.BaseNode

	meta *meta.Options

	discoveredPaths map[dependentJobs][]string
	InputPaths      []string `yaml:"inputPaths"`
	TargetThreshold int      `yaml:"targetThreshold"`
	Enabled         bool     `yaml:"enabled"`
}

// NewCodeCov initializes CodeCov.
func NewCodeCov(meta *meta.Options) *CodeCov {
	return &CodeCov{
		BaseNode: dag.NewBaseNode("coverage"),

		meta:            meta,
		discoveredPaths: make(map[dependentJobs][]string),

		Enabled:         true,
		TargetThreshold: 50,
	}
}

// AddDiscoveredInputs sets automatically discovered codecov.txt files.
func (coverage *CodeCov) AddDiscoveredInputs(jobName string, flags string, inputs ...string) {
	coverage.discoveredPaths[dependentJobs{name: jobName, flags: flags}] = append(coverage.discoveredPaths[dependentJobs{name: jobName, flags: flags}], inputs...)
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

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (coverage *CodeCov) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	if !coverage.Enabled {
		return nil
	}

	for job, paths := range coverage.discoveredPaths {
		output.AddStepInParallelJob(
			job.name,
			ghworkflow.GenericRunner,
			ghworkflow.Step("coverage").
				SetUses(fmt.Sprintf("codecov/codecov-action@%s", config.CodeCovActionVersion)).
				SetWith("files",
					strings.Join(
						xslices.Map(paths,
							func(p string) string {
								return fmt.Sprintf("%s/%s", coverage.meta.ArtifactsPath, p)
							},
						), ","),
				).
				SetWith("flags", job.flags).
				SetWith("token", "${{ secrets.CODECOV_TOKEN }}").
				SetTimeoutMinutes(3),
		)
	}

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (coverage *CodeCov) CompileMakefile(_ *makefile.Output) error {
	return nil
}

// CompileCodeCov implements codecov.Compiler.
func (coverage *CodeCov) CompileCodeCov(output *codecov.Output) error {
	if !coverage.Enabled || coverage.meta.ContainerImageFrontend != config.ContainerImageFrontendDockerfile {
		return nil
	}

	output.Enable()
	output.Target(coverage.TargetThreshold)

	return nil
}

// SkipAsMakefileDependency implements makefile.SkipAsMakefileDependency.
func (coverage *CodeCov) SkipAsMakefileDependency() {
}
