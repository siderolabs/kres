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
	"github.com/siderolabs/kres/internal/project/meta"
)

// Linters is the common node for all linters.
type Linters struct {
	meta *meta.Options

	dag.BaseNode
}

// NewLinters builds GoVulnCheck node.
func NewLinters(meta *meta.Options) *Linters {
	meta.BuildArgs.Add(
		"GOLANGCILINT_VERSION",
		"GOFUMPT_VERSION",
	)

	return &Linters{
		BaseNode: dag.NewBaseNode("go-linters"),

		meta: meta,
	}
}

// ToolchainBuild implements common.ToolchainBuilder hook.
func (linters *Linters) ToolchainBuild(stage *dockerfile.Stage) error {
	stage.
		Step(step.Arg("GOLANGCILINT_VERSION")).
		Step(step.Script(
			fmt.Sprintf(
				"go install github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCILINT_VERSION} \\\n"+
					"\t&& mv /go/bin/golangci-lint %s/golangci-lint", linters.meta.BinPath),
		).
			MountCache(filepath.Join(linters.meta.CachePath, "go-build"), linters.meta.GitHubRepository).
			MountCache(filepath.Join(linters.meta.GoPath, "pkg"), linters.meta.GitHubRepository),
		).
		Step(step.Script(fmt.Sprintf(
			`go install golang.org/x/vuln/cmd/govulncheck@latest \
	&& mv /go/bin/govulncheck %s/govulncheck`, linters.meta.BinPath)).
			MountCache(filepath.Join(linters.meta.CachePath, "go-build"), linters.meta.GitHubRepository).
			MountCache(filepath.Join(linters.meta.GoPath, "pkg"), linters.meta.GitHubRepository),
		).
		Step(step.Arg("GOFUMPT_VERSION")).
		Step(step.Script(fmt.Sprintf(
			`go install mvdan.cc/gofumpt@${GOFUMPT_VERSION} \
	&& mv /go/bin/gofumpt %s/gofumpt`, linters.meta.BinPath)))

	return nil
}

// SkipAsMakefileDependency implements makefile.SkipAsMakefileDependency.
func (linters *Linters) SkipAsMakefileDependency() {
}
