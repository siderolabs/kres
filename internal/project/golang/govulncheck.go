// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"fmt"
	"path/filepath"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// GoVulnCheck provides GoVulnCheck linter.
//
//nolint:govet
type GoVulnCheck struct {
	dag.BaseNode

	meta *meta.Options
}

// NewGoVulnCheck builds GoVulnCheck node.
func NewGoVulnCheck(meta *meta.Options) *GoVulnCheck {
	return &GoVulnCheck{
		BaseNode: dag.NewBaseNode("lint-govulncheck"),

		meta: meta,
	}
}

// CompileMakefile implements makefile.Compiler.
func (lint *GoVulnCheck) CompileMakefile(output *makefile.Output) error {
	output.Target("lint-govulncheck").Description("Runs govulncheck linter.").
		Script("@$(MAKE) target-$@")

	return nil
}

// ToolchainBuild implements common.ToolchainBuilder hook.
func (lint *GoVulnCheck) ToolchainBuild(stage *dockerfile.Stage) error {
	stage.
		Step(step.Script(fmt.Sprintf(
			`go install golang.org/x/vuln/cmd/govulncheck@latest \
	&& mv /go/bin/govulncheck %s/govulncheck`, lint.meta.BinPath)))

	return nil
}

// CompileDockerfile implements dockerfile.Compiler.
func (lint *GoVulnCheck) CompileDockerfile(output *dockerfile.Output) error {
	output.Stage("lint-govulncheck").
		Description("runs govulncheck").
		From("base").
		Step(step.Script(
			`govulncheck ./...`,
		).
			MountCache(filepath.Join(lint.meta.CachePath, "go-build")).
			MountCache(filepath.Join(lint.meta.GoPath, "pkg")))

	return nil
}
