// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/output/gitignore"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/project/meta"
)

// Build provides very common build environment settings.
type Build struct {
	dag.BaseNode

	meta *meta.Options

	ArtifactsPath string `yaml:"artifactsPath"`
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
		Variable(makefile.SimpleVariable("ARTIFACTS", build.ArtifactsPath))

	output.Target("clean").
		Script("@rm -rf $(ARTIFACTS)").
		Phony()

	return nil
}

// CompileGitignore implements gitignore.Compiler.
func (build *Build) CompileGitignore(output *gitignore.Output) error {
	output.
		IgnorePath(build.ArtifactsPath)

	return nil
}
