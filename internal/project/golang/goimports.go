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

// Goimports provides goimports linter.
type Goimports struct {
	dag.BaseNode

	meta *meta.Options

	Version       string `yaml:"version"`
	canonicalPath string
	projectPath   string
}

// NewGoimports builds Goimports node.
func NewGoimports(meta *meta.Options, projectPath, canonicalPath string) *Goimports {
	return &Goimports{
		BaseNode: dag.NewBaseNode(genName("lint-goimports", projectPath)),

		meta: meta,

		Version:       config.GoImportsVersion,
		canonicalPath: canonicalPath,
		projectPath:   projectPath,
	}
}

// CompileMakefile implements makefile.Compiler.
func (lint *Goimports) CompileMakefile(output *makefile.Output) error {
	output.Target(lint.Name()).Description("Runs goimports linter.").
		Script("@$(MAKE) target-$@")

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("GOIMPORTS_VERSION", lint.Version))

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (lint *Goimports) CompileDockerfile(output *dockerfile.Output) error {
	output.Stage(lint.Name()).
		Description("runs goimports").
		From("base").
		Step(step.Script(
			fmt.Sprintf(
				`FILES="$(goimports -l -local %s %s)" && test -z "${FILES}" || (echo -e "Source code is not formatted with 'goimports -w -local %s %s':\n${FILES}"; exit 1)`,
				lint.canonicalPath,
				lint.projectPath,
				lint.canonicalPath,
				lint.projectPath,
			),
		))

	return nil
}
