// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"fmt"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Gofumpt provides gofumpt linter.
type Gofumpt struct {
	dag.BaseNode

	meta *meta.Options

	GoVersion   string `yaml:"goVersion"`
	Version     string `yaml:"version"`
	projectPath string
}

// NewGofumpt builds Gofumpt node.
func NewGofumpt(meta *meta.Options, projectPath string) *Gofumpt {
	meta.BuildArgs.Add("GOFUMPT_VERSION")

	return &Gofumpt{
		BaseNode: dag.NewBaseNode(genName("lint-gofumpt", projectPath)),

		meta: meta,

		GoVersion:   config.GoVersion,
		Version:     config.GoFmtVersion,
		projectPath: projectPath,
	}
}

// CompileMakefile implements makefile.Compiler.
func (lint *Gofumpt) CompileMakefile(output *makefile.Output) error {
	output.Target(lint.Name()).Description("Runs gofumpt linter.").
		Script("@$(MAKE) target-$@")

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("GOFUMPT_VERSION", lint.Version)).
		Variable(makefile.OverridableVariable("GO_VERSION", lint.GoVersion))

	if !output.HasTarget("fmt") {
		output.Target("fmt").Description("Formats the source code").
			Phony().
			Script(
				`@docker run --rm -it -v $(PWD):/src -w /src golang:$(GO_VERSION) \
	bash -c "export GOEXPERIMENT=loopvar; export GOTOOLCHAIN=local; \
	export GO111MODULE=on; export GOPROXY=https://proxy.golang.org; \
	go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION) && \
	gofumpt -w ."`,
			)
	}

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (lint *Gofumpt) CompileDockerfile(output *dockerfile.Output) error {
	output.Stage(lint.Name()).
		Description("runs gofumpt").
		From("base").
		Step(step.Script(
			fmt.Sprintf(
				`FILES="$(gofumpt -l %s)" && test -z "${FILES}" || (echo -e "Source code is not formatted with 'gofumpt -w %s':\n${FILES}"; exit 1)`,
				lint.projectPath,
				lint.projectPath,
			),
		))

	return nil
}
