// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"fmt"
	"path/filepath"

	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/output/dockerfile/step"
	"github.com/talos-systems/kres/internal/output/drone"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/project/meta"
)

// Build produces binaries for Go programs.
type Build struct {
	dag.BaseNode

	meta       *meta.Options
	sourcePath string
}

// NewBuild initializes Build.
func NewBuild(meta *meta.Options, name, sourcePath string) *Build {
	return &Build{
		BaseNode:   dag.NewBaseNode(name),
		meta:       meta,
		sourcePath: sourcePath,
	}
}

// CompileDockerfile implements dockerfile.Compiler.
func (build *Build) CompileDockerfile(output *dockerfile.Output) error {
	stage := output.Stage(fmt.Sprintf("%s-build", build.Name())).
		Description(fmt.Sprintf("builds %s", build.Name())).
		From("base").
		Step(step.WorkDir(filepath.Join("/src", build.sourcePath)))

	ldflags := "-s -w"

	if build.meta.VersionPackage != "" {
		stage.
			Step(step.Arg(fmt.Sprintf("VERSION_PKG=\"%s\"", build.meta.VersionPackage))).
			Step(step.Arg("SHA")).
			Step(step.Arg("TAG"))

		ldflags += fmt.Sprintf(" -X ${VERSION_PKG}.Name=%s", build.Name())
		ldflags += " -X ${VERSION_PKG}.SHA=${SHA} -X ${VERSION_PKG}.Tag=${TAG}"
	}

	stage.Step(step.Script(fmt.Sprintf(`go build -ldflags "%s" -o /%s`, ldflags, build.Name())).
		MountCache(filepath.Join(build.meta.CachePath, "go-build")))

	output.Stage(build.Name()).
		From("scratch").
		Step(step.Copy("/"+build.Name(), "/"+build.Name()).From(fmt.Sprintf("%s-build", build.Name())))

	return nil
}

// CompileDrone implements drone.Compiler.
func (build *Build) CompileDrone(output *drone.Output) error {
	output.Step(drone.MakeStep(build.Name()).DependsOn(dag.GatherMatchingInputNames(build, dag.Implements((*drone.Compiler)(nil)))...))

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (build *Build) CompileMakefile(output *makefile.Output) error {
	output.Target(fmt.Sprintf("$(ARTIFACTS)/%s", build.Name())).
		Script(fmt.Sprintf("@$(MAKE) local-%s DEST=$(ARTIFACTS)", build.Name())).
		Phony()

	output.Target(build.Name()).
		Description(fmt.Sprintf("Builds executable for %s.", build.Name())).
		Depends(fmt.Sprintf("$(ARTIFACTS)/%s", build.Name())).
		Phony()

	return nil
}
