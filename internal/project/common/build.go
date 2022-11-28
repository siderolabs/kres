// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/gitignore"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/output/release"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Build provides very common build environment settings.
type Build struct {
	dag.BaseNode

	meta *meta.Options

	ArtifactsPath string   `yaml:"artifactsPath"`
	IgnoredPaths  []string `yaml:"ignoredPaths"`
}

// NewBuild initializes Build.
func NewBuild(meta *meta.Options) *Build {
	meta.BuildArgs = append(meta.BuildArgs, "ARTIFACTS", "SHA", "TAG")

	return &Build{
		BaseNode: dag.NewBaseNode("build"),

		meta: meta,

		ArtifactsPath: "_out",
	}
}

// CompileDockerfile implements dockerfile.Compiler.
func (build *Build) CompileDockerfile(output *dockerfile.Output) error {
	build.meta.ArtifactsPath = build.ArtifactsPath

	output.
		AllowLocalPath(build.meta.Directories...).
		AllowLocalPath(build.meta.SourceFiles...)

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (build *Build) CompileMakefile(output *makefile.Output) error {
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.SimpleVariable("SHA", "$(shell git describe --match=none --always --abbrev=8 --dirty)")).
		Variable(makefile.SimpleVariable("TAG", "$(shell git describe --tag --always --dirty)")).
		Variable(makefile.SimpleVariable("BRANCH", "$(shell git rev-parse --abbrev-ref HEAD)")).
		Variable(makefile.SimpleVariable("ARTIFACTS", build.ArtifactsPath)).
		Variable(makefile.OverridableVariable("WITH_DEBUG", "false")).
		Variable(makefile.OverridableVariable("WITH_RACE", "false"))

	output.Target("clean").
		Description("Cleans up all artifacts.").
		Script("@rm -rf $(ARTIFACTS)").
		Phony()

	return nil
}

// CompileGitignore implements gitignore.Compiler.
func (build *Build) CompileGitignore(output *gitignore.Output) error {
	output.IgnorePath(build.ArtifactsPath)

	for _, ignoredPath := range build.IgnoredPaths {
		output.IgnorePath(ignoredPath)
	}

	return nil
}

// CompileRelease implements release.Compiler.
func (build *Build) CompileRelease(output *release.Output) error {
	output.SetMeta(build.meta)

	return nil
}
