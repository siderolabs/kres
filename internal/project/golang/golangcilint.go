// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"path/filepath"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/golangci"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// GolangciLint provides golangci-lint.
type GolangciLint struct {
	dag.BaseNode

	meta *meta.Options

	DepguardExtraRules map[string]any `yaml:"depguardExtraRules"`

	Version     string
	projectPath string

	BuildTags []string `yaml:"buildTags"`
}

// NewGolangciLint builds golangci-lint node.
func NewGolangciLint(meta *meta.Options, projectPath string) *GolangciLint {
	meta.SourceFiles = append(meta.SourceFiles, filepath.Join(projectPath, ".golangci.yml"))

	return &GolangciLint{
		BaseNode: dag.NewBaseNode(genName("lint-golangci-lint", projectPath)),

		meta: meta,

		Version:     config.GolangCIlintVersion,
		projectPath: projectPath,
	}
}

// CompileGolangci implements golangci.Compiler.
func (lint *GolangciLint) CompileGolangci(output *golangci.Output) error {
	output.Enable()
	output.SetDepguardExtraRules(lint.DepguardExtraRules)
	output.SetBuildTags(lint.BuildTags)
	output.NewFile(lint.projectPath)

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (lint *GolangciLint) CompileMakefile(output *makefile.Output) error {
	output.Target(lint.Name()).Description("Runs golangci-lint linter.").
		Script("@$(MAKE) target-$@")

	output.Target(lint.Name() + "-fmt").Description("Runs golangci-lint formatter and tries to fix issues automatically.").
		Script("@$(MAKE) local-$@ DEST=.")

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("GOLANGCILINT_VERSION", lint.Version))

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (lint *GolangciLint) CompileDockerfile(output *dockerfile.Output) error {
	output.Stage(lint.Name()).
		Description("runs golangci-lint").
		From("base").
		Step(step.WorkDir(filepath.Join("/src", lint.projectPath))).
		Step(step.Copy(filepath.Join(lint.projectPath, ".golangci.yml"), ".")).
		Step(step.Env("GOGC", "50")).
		Step(step.Run("golangci-lint", "run", "--config", ".golangci.yml").
			MountCache(filepath.Join(lint.meta.CachePath, "go-build"), lint.meta.GitHubRepository).
			MountCache(filepath.Join(lint.meta.CachePath, "golangci-lint"), lint.meta.GitHubRepository, step.CacheLocked).
			MountCache(filepath.Join(lint.meta.GoPath, "pkg"), lint.meta.GitHubRepository),
		)

	output.Stage(lint.Name() + "-fmt-run").
		Description("runs golangci-lint fmt").
		From("base").
		Step(step.WorkDir(filepath.Join("/src", lint.projectPath))).
		Step(step.Copy(filepath.Join(lint.projectPath, ".golangci.yml"), ".")).
		Step(step.Env("GOGC", "50")).
		Step(step.Run("golangci-lint", "fmt", "--config", ".golangci.yml").
			MountCache(filepath.Join(lint.meta.CachePath, "go-build"), lint.meta.GitHubRepository).
			MountCache(filepath.Join(lint.meta.CachePath, "golangci-lint"), lint.meta.GitHubRepository, step.CacheLocked).
			MountCache(filepath.Join(lint.meta.GoPath, "pkg"), lint.meta.GitHubRepository),
		).
		Step(step.Run("golangci-lint", "run", "--fix", "--issues-exit-code", "0", "--config", ".golangci.yml").
			MountCache(filepath.Join(lint.meta.CachePath, "go-build"), lint.meta.GitHubRepository).
			MountCache(filepath.Join(lint.meta.CachePath, "golangci-lint"), lint.meta.GitHubRepository, step.CacheLocked).
			MountCache(filepath.Join(lint.meta.GoPath, "pkg"), lint.meta.GitHubRepository),
		)

	output.Stage(lint.Name() + "-fmt").
		Description("clean golangci-lint fmt output").
		From("scratch").
		Step(step.Copy(filepath.Join("/src", lint.projectPath), lint.projectPath).From(lint.Name() + "-fmt-run"))

	return nil
}

// LinterHasFmt is implemented by linters that have a formatting step.
func (lint *GolangciLint) LinterHasFmt() {}
