// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"path/filepath"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// GoVulnCheck provides GoVulnCheck linter.
type GoVulnCheck struct { //nolint:govet
	dag.BaseNode

	Disabled bool `yaml:"disabled"`

	meta        *meta.Options
	projectPath string
}

// NewGoVulnCheck builds GoVulnCheck node.
func NewGoVulnCheck(meta *meta.Options, projectPath string) *GoVulnCheck {
	return &GoVulnCheck{
		BaseNode: dag.NewBaseNode(genName("lint-govulncheck", projectPath)),

		meta:        meta,
		projectPath: projectPath,
	}
}

// CompileMakefile implements makefile.Compiler.
func (lint *GoVulnCheck) CompileMakefile(output *makefile.Output) error {
	if lint.Disabled {
		output.Target(lint.Name()).Description("Disabled govulncheck linter.")

		return nil
	}

	output.Target(lint.Name()).Description("Runs govulncheck linter.").
		Script("@$(MAKE) target-$@")

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (lint *GoVulnCheck) CompileDockerfile(output *dockerfile.Output) error {
	if lint.Disabled {
		return nil
	}

	output.Stage(lint.Name()).
		Description("runs govulncheck").
		From("base").
		Step(step.WorkDir(filepath.Join("/src", lint.projectPath))).
		Step(step.Script(
			`govulncheck ./...`,
		).
			MountCache(filepath.Join(lint.meta.CachePath, "go-build")).
			MountCache(filepath.Join(lint.meta.GoPath, "pkg")))

	return nil
}
