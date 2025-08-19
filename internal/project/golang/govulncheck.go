// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/dockerignore"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

const govulncheckPath = "hack/govulncheck.sh"

// GoVulnCheck provides GoVulnCheck linter.
type GoVulnCheck struct { //nolint:govet
	dag.BaseNode

	Disabled bool     `yaml:"disabled"`
	Ignore   []string `yaml:"ignore,omitempty"`

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

// CompileDockerignore implements dockerignore.Compiler.
func (lint *GoVulnCheck) CompileDockerignore(output *dockerignore.Output) error {
	output.AllowLocalPath(govulncheckPath)

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

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (lint *GoVulnCheck) CompileDockerfile(output *dockerfile.Output) error {
	if lint.Disabled {
		return nil
	}

	script := "./hack/govulncheck.sh ./..."
	if len(lint.Ignore) != 0 {
		script = fmt.Sprintf("./hack/govulncheck.sh -exclude '%s' ./...", strings.Join(lint.Ignore, ","))
	}

	output.Stage(lint.Name()).
		Description("runs govulncheck").
		From("base").
		Step(step.WorkDir(filepath.Join("/src", lint.projectPath))).
		Step(step.Copy(govulncheckPath, "./hack/govulncheck.sh").Chmod(0o755)).
		Step(step.Script(
			script,
		).
			MountCache(filepath.Join(lint.meta.CachePath, "go-build"), lint.meta.GitHubRepository).
			MountCache(filepath.Join(lint.meta.GoPath, "pkg"), lint.meta.GitHubRepository),
		)

	return nil
}
