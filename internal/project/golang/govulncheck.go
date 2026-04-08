// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"os"
	"path/filepath"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/dockerignore"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

const (
	govulncheckPath        = "hack/govulncheck.sh"
	disvulncheckConfigPath = ".disvulncheck.yaml"
)

// GoVulnCheck provides GoVulnCheck linter.
type GoVulnCheck struct { //nolint:govet
	dag.BaseNode

	Disabled bool   `yaml:"disabled"`
	Version  string `yaml:"version"`

	meta                     *meta.Options
	projectPath              string
	disvulncheckConfigExists bool
}

// NewGoVulnCheck builds GoVulnCheck node.
func NewGoVulnCheck(meta *meta.Options, rootPath, projectPath string) *GoVulnCheck {
	disvulncheckConfigExists := false

	if _, err := os.Stat(filepath.Join(rootPath, disvulncheckConfigPath)); err == nil {
		disvulncheckConfigExists = true

		meta.SourceFiles = append(meta.SourceFiles, disvulncheckConfigPath)
	}

	return &GoVulnCheck{
		BaseNode: dag.NewBaseNode(genName("lint-govulncheck", projectPath)),
		Version:  config.DisVulnCheckVersion,

		meta:                     meta,
		projectPath:              projectPath,
		disvulncheckConfigExists: disvulncheckConfigExists,
	}
}

// CompileDockerignore implements dockerignore.Compiler.
func (lint *GoVulnCheck) CompileDockerignore(output *dockerignore.Output) error {
	output.AllowLocalPath(govulncheckPath)
	output.AllowLocalPath(disvulncheckConfigPath)

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (lint *GoVulnCheck) CompileMakefile(output *makefile.Output) error {
	if lint.Disabled {
		output.Target(lint.Name()).Description("Disabled govulncheck linter.")

		return nil
	}

	output.Target(lint.Name()).Description("Runs govulncheck linter.").
		Script("@$(MAKE) target-$@")

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("DIS_VULNCHECK_VERSION", lint.Version))

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (lint *GoVulnCheck) CompileDockerfile(output *dockerfile.Output) error {
	if lint.Disabled {
		return nil
	}

	script := "dis-vulncheck -tool=false ./..."

	stage := output.Stage(lint.Name()).
		Description("runs govulncheck").
		From("base").
		Step(step.WorkDir(filepath.Join("/src", lint.projectPath)))

	if lint.disvulncheckConfigExists {
		stage.Step(step.Copy(disvulncheckConfigPath, "."))
	}

	stage.Step(step.Script(
		script,
	).
		MountCache(filepath.Join(lint.meta.CachePath, "go-build"), lint.meta.GitHubRepository).
		MountCache(filepath.Join(lint.meta.GoPath, "pkg"), lint.meta.GitHubRepository),
	)

	return nil
}
