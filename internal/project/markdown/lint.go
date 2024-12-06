// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package markdown

import (
	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/output/markdownlint"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Lint provides lint-markdown target.
type Lint struct {
	dag.BaseNode

	meta *meta.Options

	BaseImage               string `yaml:"baseImage"`
	MardownLintCLIVersion   string `yaml:"markdownLintCLIVersion"`
	SentencesPerLineVersion string `yaml:"sentencesPerLineVersion"`
}

// NewLint initializes Lint.
func NewLint(meta *meta.Options) *Lint {
	meta.SourceFiles = append(meta.SourceFiles, ".markdownlint.json")

	return &Lint{
		BaseNode: dag.NewBaseNode("lint-markdown"),

		meta: meta,

		BaseImage:               config.BunContainerImageVersion,
		MardownLintCLIVersion:   config.MarkdownLintCLIVersion,
		SentencesPerLineVersion: config.SentencesPerLineVersion,
	}
}

// CompileDockerfile implements dockerfile.Compiler.
func (lint *Lint) CompileDockerfile(output *dockerfile.Output) error {
	stage := output.Stage(lint.Name()).Description("runs markdownlint").
		From("docker.io/oven/bun:" + lint.BaseImage).
		Step(step.WorkDir("/src")).
		Step(step.Run("bun", "i", "markdownlint-cli@"+lint.MardownLintCLIVersion, "sentences-per-line@"+lint.SentencesPerLineVersion)).
		Step(step.Copy(".markdownlint.json", "."))

	for _, directory := range lint.meta.MarkdownDirectories {
		stage.Step(step.Copy("./"+directory, "./"+directory))
	}

	for _, file := range lint.meta.MarkdownSourceFiles {
		stage.Step(step.Copy("./"+file, "./"+file))
	}

	stage.
		Step(step.Script(`bunx markdownlint --ignore "CHANGELOG.md" --ignore "**/node_modules/**" --ignore '**/hack/chglog/**' --rules node_modules/sentences-per-line/index.js .`))

	return nil
}

// CompileMarkdownLint implements markdown.Compiler.
func (lint *Lint) CompileMarkdownLint(output *markdownlint.Output) error {
	output.Enable()

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (lint *Lint) CompileMakefile(output *makefile.Output) error {
	output.Target(lint.Name()).Description("Runs markdownlint.").
		Script("@$(MAKE) target-$@").
		Phony()

	return nil
}
