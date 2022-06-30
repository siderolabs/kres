// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package js

import (
	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/output/dockerfile/step"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/output/template"
	"github.com/talos-systems/kres/internal/project/js/templates"
	"github.com/talos-systems/kres/internal/project/meta"
)

// EsLint provides golangci-lint.
type EsLint struct {
	meta *meta.Options

	dag.BaseNode
}

// NewEsLint builds golangci-lint node.
func NewEsLint(meta *meta.Options) *EsLint {
	meta.SourceFiles = append(meta.SourceFiles, "frontend/.eslintrc.yaml")

	return &EsLint{
		BaseNode: dag.NewBaseNode("lint-eslint"),

		meta: meta,
	}
}

// CompileTemplates implements templates.Compiler.
func (lint *EsLint) CompileTemplates(output *template.Output) error {
	output.Define("frontend/.eslintrc.yaml", templates.Eslint).
		PreamblePrefix("# ")

	return nil
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
		Step(step.Script("npm run lint").
			MountCache(lint.meta.NpmCachePath))

	return nil
}
