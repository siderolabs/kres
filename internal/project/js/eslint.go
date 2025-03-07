// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package js

import (
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
	meta.SourceFiles = append(meta.SourceFiles, "frontend/eslint.config.js")

	return &EsLint{
		BaseNode: dag.NewBaseNode("lint-eslint"),

		meta: meta,
	}
}

// CompileMakefile implements makefile.Compiler.
func (lint *EsLint) CompileMakefile(output *makefile.Output) error {
	output.Target("lint-eslint").Description("Runs eslint linter.").
		Script("@$(MAKE) target-$@")

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (lint *EsLint) CompileDockerfile(output *dockerfile.Output) error {
	output.Stage("lint-eslint").
		Description("runs eslint").
		From("js").
		Step(step.Script("bun run lint").
			MountCache(lint.meta.JSCachePath, lint.meta.GitHubRepository))

	return nil
}
