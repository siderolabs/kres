// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"path/filepath"

	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/output/dockerfile/step"
	"github.com/talos-systems/kres/internal/output/drone"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/project/meta"
)

// UnitTests runs unit-tests for Go packages.
type UnitTests struct {
	dag.BaseNode

	meta *meta.Options
}

// NewUnitTests initializes UnitTests.
func NewUnitTests(meta *meta.Options) *UnitTests {
	meta.BuildArgs = append(meta.BuildArgs, "TESTPKGS")

	return &UnitTests{
		BaseNode: dag.NewBaseNode("unit-tests"),
		meta:     meta,
	}
}

// CompileDockerfile implements dockerfile.Compiler.
func (tests *UnitTests) CompileDockerfile(output *dockerfile.Output) error {
	output.Stage("unit-tests-run").
		Description("runs unit-tests").
		From("base").
		Step(step.Arg("TESTPKGS")).
		Step(step.Script(`go test -v -covermode=atomic -coverprofile=coverage.txt -count 1 ${TESTPKGS}`).
			MountCache(filepath.Join(tests.meta.CachePath, "go-build")).
			MountCache("/tmp"))

	output.Stage("unit-tests").
		From("scratch").
		Step(step.Copy("/src/coverage.txt", "/coverage.txt").From("unit-tests-run"))

	output.Stage("unit-tests-race").
		Description("runs unit-tests with race detector").
		From("base").
		Step(step.Arg("TESTPKGS")).
		Step(step.Script(`go test -v -race -count 1 ${TESTPKGS}`).
			MountCache(filepath.Join(tests.meta.CachePath, "go-build")).
			MountCache("/tmp").
			Env("CGO_ENABLED", "1"))

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (tests *UnitTests) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("TESTPKGS", "./..."))

	output.Target("unit-tests").
		Description("Performs unit tests").
		Script("@$(MAKE) local-$@ DEST=$(ARTIFACTS)").
		Phony()

	output.Target("unit-tests-race").
		Description("Performs unit tests with race detection enabled.").
		Script("@$(MAKE) target-$@").
		Phony()

	return nil
}

// CompileDrone implements drone.Compiler.
func (tests *UnitTests) CompileDrone(output *drone.Output) error {
	output.Step(drone.MakeStep("unit-tests").
		DependsOn(dag.GatherMatchingInputNames(tests, dag.Implements((*drone.Compiler)(nil)))...),
	)

	output.Step(drone.MakeStep("unit-tests-race").
		DependsOn(dag.GatherMatchingInputNames(tests, dag.Implements((*drone.Compiler)(nil)))...),
	)

	return nil
}
