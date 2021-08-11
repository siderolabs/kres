// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"fmt"

	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/output/dockerfile/step"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/project/meta"
)

// Gofumpt provides gofumpt linter.
type Gofumpt struct {
	dag.BaseNode

	meta *meta.Options

	GoVersion string `yaml:"goVersion"`
	Version   string `yaml:"version"`
}

// NewGofumpt builds Gofumpt node.
func NewGofumpt(meta *meta.Options) *Gofumpt {
	meta.BuildArgs = append(meta.BuildArgs, "GOFUMPT_VERSION")

	return &Gofumpt{
		BaseNode: dag.NewBaseNode("lint-gofumpt"),

		meta: meta,

		GoVersion: "1.16",
		Version:   "abc0db2c416aca0f60ea33c23c76665f6e7ba0b6",
	}
}

// CompileMakefile implements makefile.Compiler.
func (lint *Gofumpt) CompileMakefile(output *makefile.Output) error {
	output.Target("lint-gofumpt").Description("Runs gofumpt linter.").
		Script("@$(MAKE) target-$@")

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("GOFUMPT_VERSION", lint.Version)).
		Variable(makefile.OverridableVariable("GO_VERSION", lint.GoVersion))

	output.Target("fmt").Description("Formats the source code").
		Phony().
		Script(fmt.Sprintf(
			`@docker run --rm -it -v $(PWD):/src -w /src golang:$(GO_VERSION) \
	bash -c "export GO111MODULE=on; export GOPROXY=https://proxy.golang.org; \
	go install mvdan.cc/gofumpt/gofumports@$(GOFUMPT_VERSION) && \
	gofumports -w -local %s ."`,
			lint.meta.CanonicalPath,
		))

	return nil
}

// ToolchainBuild implements common.ToolchainBuilder hook.
func (lint *Gofumpt) ToolchainBuild(stage *dockerfile.Stage) error {
	stage.
		Step(step.Arg("GOFUMPT_VERSION")).
		Step(step.Script(fmt.Sprintf(
			`go install mvdan.cc/gofumpt/gofumports@${GOFUMPT_VERSION} \
	&& mv /go/bin/gofumports %s/gofumports`, lint.meta.BinPath)))

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (lint *Gofumpt) CompileDockerfile(output *dockerfile.Output) error {
	output.Stage("lint-gofumpt").
		Description("runs gofumpt").
		From("base").
		Step(step.Script(`find . -name '*.pb.go' | xargs -r rm`)).
		Step(step.Script(`find . -name '*.pb.gw.go' | xargs -r rm`)).
		Step(step.Script(
			fmt.Sprintf(
				`FILES="$(gofumports -l -local %s .)" && test -z "${FILES}" || (echo -e "Source code is not formatted with 'gofumports -w -local %s .':\n${FILES}"; exit 1)`,
				lint.meta.CanonicalPath,
				lint.meta.CanonicalPath,
			),
		))

	return nil
}
