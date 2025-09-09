// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package js

import (
	"fmt"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// EsLint provides eslint.
type EsLint struct {
	meta *meta.Options
	dag.BaseNode
}

// NewEsLint builds eslint node.
func NewEsLint(meta *meta.Options) *EsLint {
	meta.SourceFiles = append(meta.SourceFiles, "frontend/eslint.config.ts")

	return &EsLint{
		BaseNode: dag.NewBaseNode("lint-eslint"),

		meta: meta,
	}
}

// CompileMakefile implements makefile.Compiler.
func (lint *EsLint) CompileMakefile(output *makefile.Output) error {
	output.Target(lint.Name()).Description("Runs eslint linter & prettier style check.").
		Script("@$(MAKE) target-$@")

	output.Target(lint.Name() + "-fmt").Description("Runs eslint & prettier and tries to fix issues automatically, updating the source tree.").
		Script("@$(MAKE) local-$@ DEST=.")

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (lint *EsLint) CompileDockerfile(output *dockerfile.Output) error {
	output.Stage(lint.Name()).
		Description("runs eslint & prettier").
		From("js").
		Step(step.Script("npm run lint"))

	output.Stage(lint.Name() + "-fmt-run").
		Description("runs eslint & prettier with autofix.").
		From("js").
		Step(step.Script("npm run lint:fix"))

	output.Stage(lint.Name() + "-fmt").
		Description(fmt.Sprintf("trim down %s output to contain only source files", lint.Name()+"-fmt-run")).
		From("scratch").
		Step(step.Copy("/src", "/frontend").
			From(lint.Name() + "-fmt-run").
			Exclude("node_modules"),
		)

	return nil
}

// LinterHasFmt is implemented by linters that have a formatting step.
func (lint *EsLint) LinterHasFmt() {}
