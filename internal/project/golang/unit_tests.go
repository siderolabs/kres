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
	ExtraArgs string `yaml:"extraArgs"`
	Docker    struct {
		Steps []struct {
			Copy *struct {
				From     string `yaml:"from"`
				Platform string `yaml:"platform"`
				Src      string `yaml:"src"`
				Dst      string `yaml:"dst"`
			} `yaml:"copy"`
			Arg string `yaml:"arg"`
		} `yaml:"steps"`
	} `yaml:"docker"`
	RunFIPS bool `yaml:"runFIPS"`
	Verbose bool `yaml:"verbose"`
	Count   int  `yaml:"count"`

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

	applyDockerCopySteps := func(stage *dockerfile.Stage) {
		for _, dockerStep := range tests.Docker.Steps {
			if dockerStep.Copy != nil {
				copyStep := step.Copy(dockerStep.Copy.Src, dockerStep.Copy.Dst)
				if dockerStep.Copy.From != "" {
					copyStep.From(dockerStep.Copy.From)
				}

				if dockerStep.Copy.Platform != "" {
					copyStep.Platform(dockerStep.Copy.Platform)
				}

				stage.Step(copyStep)
			}
		}
	}

	extraArgs := tests.ExtraArgs
	if extraArgs != "" {
		extraArgs += " "
	}

	workdir := step.WorkDir(filepath.Join("/src", tests.packagePath))
	testRun := tests.Name() + "-run"

	testRunStage := output.Stage(testRun).
		Description("runs unit-tests").
		From("base")

	applyDockerCopySteps(testRunStage)

	verboseArg := ""
	if tests.Verbose {
		verboseArg = "-v "
	}

	countArg := ""
	if tests.Count > 0 {
		countArg = fmt.Sprintf("-count %d ", tests.Count)
	}

	testRunStage.Step(workdir).
		Step(step.Arg("TESTPKGS")).
		Step(wrapAsInsecure(
			step.Script(
				fmt.Sprintf(`go test %s-covermode=atomic -coverprofile=coverage.txt -coverpkg=${TESTPKGS} %s%s${TESTPKGS}`, verboseArg, countArg, extraArgs)).
				MountCache(filepath.Join(tests.meta.CachePath, "go-build"), tests.meta.GitHubRepository).
				MountCache(filepath.Join(tests.meta.GoPath, "pkg"), tests.meta.GitHubRepository).
				MountCache("/tmp", tests.meta.GitHubRepository)))

	output.Stage(tests.Name()).
		From("scratch").
		Step(step.Copy(filepath.Join("/src", tests.packagePath, "coverage.txt"), fmt.Sprintf("/coverage-%s.txt", tests.Name())).From(testRun))

	testRunRaceStage := output.Stage(tests.Name() + "-race").
		Description("runs unit-tests with race detector").
		From("base")

	applyDockerCopySteps(testRunRaceStage)

	testRunRaceStage.Step(workdir).
		Step(step.Arg("TESTPKGS")).
		Step(wrapAsInsecure(
			step.Script(
				fmt.Sprintf(`go test %s-race %s%s${TESTPKGS}`, verboseArg, countArg, extraArgs),
			).
				MountCache(filepath.Join(tests.meta.CachePath, "go-build"), tests.meta.GitHubRepository).
				MountCache(filepath.Join(tests.meta.GoPath, "pkg"), tests.meta.GitHubRepository).
				MountCache("/tmp", tests.meta.GitHubRepository).
				Env("CGO_ENABLED", "1")))

	if tests.RunFIPS {
		testRunFIPSStage := output.Stage(tests.Name() + "-fips").
			Description("runs unit-tests with strict FIPS-140 mode").
			From("base")

		applyDockerCopySteps(testRunFIPSStage)

		testRunFIPSStage.Step(workdir).
			Step(step.Arg("TESTPKGS")).
			Step(wrapAsInsecure(
				step.Script(
					fmt.Sprintf(`go test %s%s%s${TESTPKGS}`, verboseArg, countArg, extraArgs),
				).
					MountCache(filepath.Join(tests.meta.CachePath, "go-build"), tests.meta.GitHubRepository).
					MountCache(filepath.Join(tests.meta.GoPath, "pkg"), tests.meta.GitHubRepository).
					MountCache("/tmp", tests.meta.GitHubRepository).
					Env("GOFIPS140", "latest").
					Env("GODEBUG", "fips140=only,tlsmlkem=0")))
	}

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

	if tests.RunFIPS {
		output.Target(tests.Name() + "-fips").
			Description("Performs unit tests with strict FIPS-140 mode.").
			Script("@$(MAKE) target-$@" + scriptExtraArgs).
			Phony()
	}

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

	if tests.RunFIPS {
		output.Step(drone.MakeStep(fmt.Sprintf(tests.Name(), "-fips")).
			DependsOn(dag.GatherMatchingInputNames(tests, dag.Implements[drone.Compiler]())...),
		)
	}

	return nil
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (tests *UnitTests) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	output.AddStep(
		"default",
		ghworkflow.Step(tests.Name()).SetMakeStep(tests.Name()),
		ghworkflow.Step(tests.Name()+"-race").SetMakeStep(tests.Name()+"-race"),
	)

	if tests.RunFIPS {
		output.AddStep(
			"default",
			ghworkflow.Step(tests.Name()+"-fips").SetMakeStep(tests.Name()+"-fips"),
		)
	}

	return nil
}
