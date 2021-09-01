// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"fmt"
	"path/filepath"
	"sort"

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

	Outputs map[string]CompileConfig `yaml:"outputs"`

	meta       *meta.Options
	sourcePath string
	entrypoint string
	artifacts  []string
}

// CompileConfig defines Go cross compile architecture settings.
type CompileConfig map[string]string

func (c CompileConfig) set(script *step.RunStep) {
	keys := make([]string, 0, len(c))

	for key := range c {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		script.Env(key, c[key])
	}
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
	addBuildSteps := func(name string, opts CompileConfig) {
		stage := output.Stage(fmt.Sprintf("%s-build", name)).
			Description(fmt.Sprintf("builds %s", name)).
			From("base").
			Step(step.Copy("/", "/").From("generate")).
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

		script := step.Script(fmt.Sprintf(`go build -ldflags "%s" -o /%s`, ldflags, name)).
			MountCache(filepath.Join(build.meta.CachePath, "go-build")).
			MountCache(filepath.Join(build.meta.GoPath, "pkg"))

		if opts != nil {
			opts.set(script)
		}

		stage.Step(script)

		output.Stage(name).
			From("scratch").
			Step(step.Copy("/"+name, "/"+name).From(fmt.Sprintf("%s-build", name)))
	}

	if len(build.Outputs) == 0 {
		build.Outputs = map[string]CompileConfig{
			fmt.Sprintf("%s-linux-amd64", build.Name()): nil,
		}
	}

	for _, name := range build.getArtifacts() {
		addBuildSteps(name, build.Outputs[name])
	}

	build.entrypoint = fmt.Sprintf("%s-linux-${TARGETARCH}", build.Name())
	output.Stage(build.Name()).
		From(build.entrypoint)

	return nil
}

// CompileDrone implements drone.Compiler.
func (build *Build) CompileDrone(output *drone.Output) error {
	output.Step(drone.MakeStep(build.Name()).DependsOn(dag.GatherMatchingInputNames(build, dag.Implements((*drone.Compiler)(nil)))...))

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (build *Build) CompileMakefile(output *makefile.Output) error {
	for _, artifact := range build.getArtifacts() {
		output.Target(fmt.Sprintf("$(ARTIFACTS)/%s", artifact)).
			Script(fmt.Sprintf("@$(MAKE) local-%s DEST=$(ARTIFACTS)", artifact)).
			Phony()

		output.Target(artifact).
			Description(fmt.Sprintf("Builds executable for %s.", artifact)).
			Depends(fmt.Sprintf("$(ARTIFACTS)/%s", artifact)).
			Phony()
	}

	output.Target(build.Name()).
		Description(fmt.Sprintf("Builds executables for %s.", build.Name())).
		Depends(build.artifacts...).
		Phony()

	return nil
}

// Entrypoint implements dockerfile.CmdCompiler.
func (build *Build) Entrypoint() string {
	return build.entrypoint
}

func (build *Build) getArtifacts() []string {
	if build.artifacts != nil {
		return build.artifacts
	}

	if len(build.Outputs) == 0 {
		build.artifacts = []string{fmt.Sprintf("%s-linux-amd64", build.Name())}
	} else {
		build.artifacts = []string{}

		for name := range build.Outputs {
			build.artifacts = append(build.artifacts, name)
		}

		sort.Strings(build.artifacts)
	}

	return build.artifacts
}
