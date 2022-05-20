// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"fmt"
	"path/filepath"

	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/dockerfile"
	"github.com/talos-systems/kres/internal/output/dockerfile/step"
	"github.com/talos-systems/kres/internal/output/golangci"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/project/meta"
)

// GolangciLint provides golangci-lint.
type GolangciLint struct {
	dag.BaseNode

	meta *meta.Options

	Version string
}

// NewGolangciLint builds golangci-lint node.
func NewGolangciLint(meta *meta.Options) *GolangciLint {
	meta.SourceFiles = append(meta.SourceFiles, ".golangci.yml")
	meta.BuildArgs = append(meta.BuildArgs, "GOLANGCILINT_VERSION")

	return &GolangciLint{
		BaseNode: dag.NewBaseNode("lint-golangci-lint"),

		meta: meta,

		Version: "v1.46.2",
	}
}

// CompileGolangci implements golangci.Compiler.
func (lint *GolangciLint) CompileGolangci(output *golangci.Output) error {
	output.Enable()
	output.CanonicalPath(lint.meta.CanonicalPath)

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (lint *GolangciLint) CompileMakefile(output *makefile.Output) error {
	output.Target("lint-golangci-lint").Description("Runs golangci-lint linter.").
		Script("@$(MAKE) target-$@")

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("GOLANGCILINT_VERSION", lint.Version))

	return nil
}

// ToolchainBuild implements common.ToolchainBuilder hook.
func (lint *GolangciLint) ToolchainBuild(stage *dockerfile.Stage) error {
	stage.
		Step(step.Arg("GOLANGCILINT_VERSION")).
		Step(step.Script(
			fmt.Sprintf("curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/${GOLANGCILINT_VERSION}/install.sh | bash -s -- -b %s ${GOLANGCILINT_VERSION}", lint.meta.BinPath),
		))

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (lint *GolangciLint) CompileDockerfile(output *dockerfile.Output) error {
	output.Stage("lint-golangci-lint").
		Description("runs golangci-lint").
		From("base").
		Step(step.Copy(".golangci.yml", ".")).
		Step(step.Env("GOGC", "50")).
		Step(step.Run("golangci-lint", "run", "--config", ".golangci.yml").
			MountCache(filepath.Join(lint.meta.CachePath, "go-build")).
			MountCache(filepath.Join(lint.meta.CachePath, "golangci-lint")).
			MountCache(filepath.Join(lint.meta.GoPath, "pkg")),
		)

	return nil
}
