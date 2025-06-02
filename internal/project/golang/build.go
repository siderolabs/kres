// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/siderolabs/gen/maps"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Build produces binaries for Go programs.
type Build struct {
	dag.BaseNode

	Outputs    map[string]CompileConfig `yaml:"outputs"`
	BuildFlags []string                 `yaml:"buildFlags"`

	meta       *meta.Options
	sourcePath string
	entrypoint string
	command    string
	artifacts  []artifact
	configs    []CompileConfig
}

// CompileConfig defines Go cross compile architecture settings.
type CompileConfig map[string]string

type artifact struct {
	config CompileConfig
	name   string
}

func (c CompileConfig) set(script *step.RunStep) {
	keys := maps.Keys(c)

	slices.Sort(keys)

	for _, key := range keys {
		script.Env(key, c[key])
	}
}

// NewBuild initializes Build.
func NewBuild(meta *meta.Options, name, sourcePath, buildCommand string) *Build {
	return &Build{
		BaseNode:   dag.NewBaseNode(name),
		meta:       meta,
		sourcePath: sourcePath,
		command:    buildCommand,
	}
}

// CompileDockerfile implements dockerfile.Compiler.
func (build *Build) CompileDockerfile(output *dockerfile.Output) error {
	addBuildSteps := func(name string, opts CompileConfig) {
		stage := output.Stage(name + "-build").
			Description("builds " + name).
			From("base").
			Step(step.Copy("/", "/").From("generate"))

		if build.meta.VersionPackagePath != "" {
			stage.Step(step.Copy("/", "/").From("embed-generate"))
		}

		stage.
			Step(step.WorkDir(filepath.Join("/src", build.sourcePath))).
			Step(step.Arg("GO_BUILDFLAGS")).
			Step(step.Arg("GO_LDFLAGS"))

		ldflags := "${GO_LDFLAGS}"

		if build.meta.VersionPackagePath != "" {
			stage.
				Step(step.Arg(fmt.Sprintf("VERSION_PKG=\"%s\"", build.meta.VersionPackagePath))).
				Step(step.Arg("SHA")).
				Step(step.Arg("TAG"))

			ldflags += " -X ${VERSION_PKG}.Name=" + build.Name()
			ldflags += " -X ${VERSION_PKG}.SHA=${SHA} -X ${VERSION_PKG}.Tag=${TAG}"
		}

		buildFlags := " ${GO_BUILDFLAGS}"

		if build.BuildFlags != nil {
			buildFlags = " " + strings.Join(build.BuildFlags, " ")
		}

		script := step.Script(fmt.Sprintf(`%s%s -ldflags "%s" -o /%s`, build.command, buildFlags, ldflags, name)).
			MountCache(filepath.Join(build.meta.CachePath, "go-build"), build.meta.GitHubRepository).
			MountCache(filepath.Join(build.meta.GoPath, "pkg"), build.meta.GitHubRepository)

		if opts != nil {
			opts.set(script)
		}

		stage.Step(script)

		output.Stage(name).
			From("scratch").
			Step(step.Copy("/"+name, "/"+name).From(name + "-build"))
	}

	for _, artifact := range build.getArtifacts() {
		addBuildSteps(artifact.name, artifact.config)
	}

	// combine all binaries built for each arch into '-all' stage
	all := output.Stage(build.Name() + "-all").
		From("scratch")

	for _, artifact := range build.getArtifacts() {
		all.Step(step.Copy("/", "/").From(artifact.name))
	}

	build.entrypoint = build.Name() + "-linux-${TARGETARCH}"
	output.Stage(build.Name()).
		From(build.entrypoint)

	return nil
}

// CompileDrone implements drone.Compiler.
func (build *Build) CompileDrone(output *drone.Output) error {
	output.Step(drone.MakeStep(build.Name()).DependsOn(dag.GatherMatchingInputNames(build, dag.Implements[drone.Compiler]())...))

	return nil
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (build *Build) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	output.AddStep(
		"default",
		ghworkflow.Step(build.Name()).SetMakeStep(build.Name()),
	)

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (build *Build) CompileMakefile(output *makefile.Output) error {
	artifacts := build.getArtifacts()
	deps := make([]string, 0, len(artifacts))

	for _, artifact := range artifacts {
		output.Target("$(ARTIFACTS)/" + artifact.name).
			Script(fmt.Sprintf("@$(MAKE) local-%s DEST=$(ARTIFACTS)", artifact.name)).
			Phony()

		deps = append(deps, artifact.name)

		output.Target(artifact.name).
			Description(fmt.Sprintf("Builds executable for %s.", artifact.name)).
			Depends("$(ARTIFACTS)/" + artifact.name).
			Phony()
	}

	output.Target(build.Name()).
		Description(fmt.Sprintf("Builds executables for %s.", build.Name())).
		Depends(deps...).
		Phony()

	return nil
}

// Entrypoint implements dockerfile.CmdCompiler.
func (build *Build) Entrypoint() string {
	return build.entrypoint
}

func (build *Build) getArtifacts() []artifact {
	if build.artifacts != nil {
		return build.artifacts
	}

	build.configs = []CompileConfig{}

	if len(build.Outputs) == 0 {
		build.artifacts = []artifact{
			{
				name: build.Name() + "-linux-amd64",
			},
		}
	} else {
		build.artifacts = maps.ToSlice(build.Outputs, func(name string, config CompileConfig) artifact {
			return artifact{
				name:   strings.Join([]string{build.Name(), name}, "-"),
				config: config,
			}
		})

		slices.SortFunc(build.artifacts, func(a, b artifact) int {
			return cmp.Compare(a.name, b.name)
		})
	}

	return build.artifacts
}
