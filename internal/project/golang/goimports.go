// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"fmt"
	"path/filepath"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// Goimports provides goimports linter.
type Goimports struct {
	dag.BaseNode

	meta *meta.Options

	Version string `yaml:"version"`
}

// NewGoimports builds Goimports node.
func NewGoimports(meta *meta.Options) *Goimports {
	meta.BuildArgs = append(meta.BuildArgs, "GOIMPORTS_VERSION")

	return &Goimports{
		BaseNode: dag.NewBaseNode("lint-goimports"),

		meta: meta,

		Version: config.GoImportsVersion,
	}
}

// CompileMakefile implements makefile.Compiler.
func (lint *Goimports) CompileMakefile(output *makefile.Output) error {
	output.Target("lint-goimports").Description("Runs goimports linter.").
		Script("@$(MAKE) target-$@")

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("GOIMPORTS_VERSION", lint.Version))

	return nil
}

// ToolchainBuild implements common.ToolchainBuilder hook.
func (lint *Goimports) ToolchainBuild(stage *dockerfile.Stage) error {
	stage.
		Step(step.Arg("GOIMPORTS_VERSION")).
		Step(step.Script(fmt.Sprintf(
			`go install golang.org/x/tools/cmd/goimports@${GOIMPORTS_VERSION} \
	&& mv /go/bin/goimports %s/goimports`, lint.meta.BinPath)).
			MountCache(filepath.Join(lint.meta.CachePath, "go-build")).
			MountCache(filepath.Join(lint.meta.GoPath, "pkg")),
		)

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (lint *Goimports) CompileDockerfile(output *dockerfile.Output) error {
	output.Stage("lint-goimports").
		Description("runs goimports").
		From("base").
		Step(step.Script(
			fmt.Sprintf(
				`FILES="$(goimports -l -local %s .)" && test -z "${FILES}" || (echo -e "Source code is not formatted with 'goimports -w -local %s .':\n${FILES}"; exit 1)`,
				lint.meta.CanonicalPath,
				lint.meta.CanonicalPath,
			),
		))

	return nil
}
