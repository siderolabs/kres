// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"fmt"
	"path/filepath"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// UnitTests runs unit-tests for Go packages.
type UnitTests struct { //nolint:govet
	dag.BaseNode

	RequiresInsecure bool `yaml:"requiresInsecure"`
	// ExtraArgs are extra arguments for `go test`.
	ExtraArgs   string `yaml:"extraArgs"`
	packagePath string

	meta *meta.Options
}

// NewUnitTests initializes UnitTests.
func NewUnitTests(meta *meta.Options, packagePath string) *UnitTests {
	meta.BuildArgs.Add("TESTPKGS")

	return &UnitTests{
		BaseNode:    dag.NewBaseNode(genName("unit-tests", packagePath)),
		meta:        meta,
		packagePath: packagePath,
	}
}

// CompileDockerfile implements dockerfile.Compiler.
func (tests *UnitTests) CompileDockerfile(output *dockerfile.Output) error {
	wrapAsInsecure := func(s *step.RunStep) *step.RunStep {
		if tests.RequiresInsecure {
			return s.SecurityInsecure()
		}

		return s
	}

	extraArgs := tests.ExtraArgs
	if extraArgs != "" {
		extraArgs += " "
	}

	workdir := step.WorkDir(filepath.Join("/src", tests.packagePath))
	testRun := tests.Name() + "-run"

	output.Stage(testRun).
		Description("runs unit-tests").
		From("base").
		Step(workdir).
		Step(step.Arg("TESTPKGS")).
		Step(wrapAsInsecure(
			step.Script(
				fmt.Sprintf(
					`go test -v -covermode=atomic -coverprofile=coverage.txt -coverpkg=${TESTPKGS} -count 1 %s${TESTPKGS}`,
					extraArgs),
			).
				MountCache(filepath.Join(tests.meta.CachePath, "go-build")).
				MountCache(filepath.Join(tests.meta.GoPath, "pkg")).
				MountCache("/tmp")))

	output.Stage(tests.Name()).
		From("scratch").
		Step(step.Copy(filepath.Join("/src", tests.packagePath, "coverage.txt"), fmt.Sprintf("/coverage-%s.txt", tests.Name())).From(testRun))

	//nolint:goconst
	output.Stage(tests.Name() + "-race").
		Description("runs unit-tests with race detector").
		From("base").
		Step(workdir).
		Step(step.Arg("TESTPKGS")).
		Step(wrapAsInsecure(
			step.Script(
				fmt.Sprintf(
					`go test -v -race -count 1 %s${TESTPKGS}`,
					extraArgs,
				),
			).
				MountCache(filepath.Join(tests.meta.CachePath, "go-build")).
				MountCache(filepath.Join(tests.meta.GoPath, "pkg")).
				MountCache("/tmp").
				Env("CGO_ENABLED", "1")))

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (tests *UnitTests) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("TESTPKGS", "./..."))

	scriptExtraArgs := ""

	if tests.RequiresInsecure {
		scriptExtraArgs += `  TARGET_ARGS="--allow security.insecure"`
	}

	output.Target(tests.Name()).
		Description("Performs unit tests").
		Script("@$(MAKE) local-$@ DEST=$(ARTIFACTS)" + scriptExtraArgs).
		Phony()

	output.Target(tests.Name() + "-race").
		Description("Performs unit tests with race detection enabled.").
		Script("@$(MAKE) target-$@" + scriptExtraArgs).
		Phony()

	return nil
}

// CompileDrone implements drone.Compiler.
func (tests *UnitTests) CompileDrone(output *drone.Output) error {
	output.Step(drone.MakeStep(tests.Name()).
		DependsOn(dag.GatherMatchingInputNames(tests, dag.Implements[drone.Compiler]())...),
	)

	output.Step(drone.MakeStep(fmt.Sprintf(tests.Name(), "-race")).
		DependsOn(dag.GatherMatchingInputNames(tests, dag.Implements[drone.Compiler]())...),
	)

	return nil
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (tests *UnitTests) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	output.AddStep(
		"default",
		ghworkflow.MakeStep(tests.Name()),
		ghworkflow.MakeStep(tests.Name()+"-race"),
	)

	return nil
}
