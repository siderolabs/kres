// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package js

import (
	"fmt"
	"path/filepath"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/drone"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/output/template"
	"github.com/siderolabs/kres/internal/project/js/templates"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Build produces binaries for Go programs.
type Build struct {
	meta *meta.Options

	dag.BaseNode

	embedFile   string
	LicenseText string `yaml:"licenseText"`
	artifacts   []string
}

const nodeBuildArgsVarName = "NODE_BUILD_ARGS"

// NewBuild initializes Build.
func NewBuild(meta *meta.Options, name string) *Build {
	embedFile := fmt.Sprintf("internal/%s/%s.go", name, name)
	meta.SourceFiles = append(meta.SourceFiles, embedFile)
	meta.BuildArgs = append(meta.BuildArgs, nodeBuildArgsVarName)

	return &Build{
		BaseNode:  dag.NewBaseNode(name),
		meta:      meta,
		embedFile: embedFile,
	}
}

// CompileTemplates implements template.Compiler.
func (build *Build) CompileTemplates(output *template.Output) error {
	output.Define(build.embedFile, templates.GoEmbed).
		Params(map[string]string{
			"project": build.Name(),
		}).
		PreamblePrefix("// ").
		WithLicense().
		WithLicenseText(build.LicenseText)

	distDir := filepath.Join(
		filepath.Dir(build.embedFile),
		"dist",
		".gitkeep",
	)

	output.Define(distDir, "").NoPreamble()

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (build *Build) CompileDockerfile(output *dockerfile.Output) error {
	outputDir := fmt.Sprintf("/internal/%s/dist", build.Name())

	output.Stage(build.Name()).
		Description("builds " + build.Name()).
		From("--platform=${BUILDPLATFORM} js").
		Step(step.Arg(nodeBuildArgsVarName)).
		Step(step.Script("npm run build ${" + nodeBuildArgsVarName + "}").
			MountCache(build.meta.NpmCachePath)).
		Step(step.Script("mkdir -p " + outputDir)).
		Step(step.Script("cp -rf ./dist/* " + outputDir))

	build.artifacts = []string{outputDir}

	return nil
}

// CompileDrone implements drone.Compiler.
func (build *Build) CompileDrone(output *drone.Output) error {
	output.Step(drone.MakeStep(build.Name()).DependsOn(dag.GatherMatchingInputNames(build, dag.Implements[drone.Compiler]())...))

	return nil
}

// CompileGitHubWorkflow implements ghworkflow.Compiler.
func (build *Build) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	output.AddStep("default", ghworkflow.Step(build.Name()).SetMakeStep(build.Name()))

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (build *Build) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable(nodeBuildArgsVarName, ""))

	output.Target(fmt.Sprintf("$(ARTIFACTS)/%s-js", build.Name())).
		Script("@$(MAKE) target-" + build.Name()).
		Phony()

	output.Target(build.Name()).
		Description(fmt.Sprintf("Builds js release for %s.", build.Name())).
		Depends(fmt.Sprintf("$(ARTIFACTS)/%s-js", build.Name())).
		Phony()

	return nil
}

// GetArtifacts implements dockerfile.Generator.
func (build *Build) GetArtifacts() []string {
	return build.artifacts
}
