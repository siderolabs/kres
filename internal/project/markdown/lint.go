// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package markdown

import (
	"fmt"

	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/output/dockerfile/step"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/output/markdownlint"
	"github.com/talos-systems/kres/internal/project/meta"
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

		BaseImage:               "node:14.8.0-alpine",
		MardownLintCLIVersion:   "0.23.2",
		SentencesPerLineVersion: "0.2.1",
	}
}

// CompileDockerfile implements dockerfile.Compiler.
func (lint *Lint) CompileDockerfile(output *dockerfile.Output) error {
	stage := output.Stage(lint.Name()).Description("runs markdownlint").
		From(lint.BaseImage).
		Step(step.Run("npm", "i", "-g", fmt.Sprintf("markdownlint-cli@%s", lint.MardownLintCLIVersion))).
		Step(step.Run("npm", "i", fmt.Sprintf("sentences-per-line@%s", lint.SentencesPerLineVersion))).
		Step(step.WorkDir("/src")).
		Step(step.Copy(".markdownlint.json", "."))

	for _, directory := range lint.meta.MarkdownDirectories {
		stage.Step(step.Copy("./"+directory, "./"+directory))
	}

	for _, file := range lint.meta.MarkdownSourceFiles {
		stage.Step(step.Copy("./"+file, "./"+file))
	}

	stage.
		Step(step.Script(`markdownlint --ignore "**/node_modules/**" --ignore '**/hack/chglog/**' --rules /node_modules/sentences-per-line/index.js .`))

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
