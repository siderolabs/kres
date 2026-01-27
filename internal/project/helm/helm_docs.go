// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package helm

import (
	"fmt"
	"path/filepath"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/project/meta"
)

// HelmDocs is a helm-docs node.
type HelmDocs struct {
	meta *meta.Options
	dag.BaseNode
}

// NewHelmDocs initializes HelmDocs.
func NewHelmDocs(meta *meta.Options) *HelmDocs {
	meta.BuildArgs.Add(
		"HELMDOCS_VERSION",
	)

	return &HelmDocs{
		meta:     meta,
		BaseNode: dag.NewBaseNode("helm-docs"),
	}
}

// ToolchainBuild implements common.ToolchainBuilder hook.
func (helm *HelmDocs) ToolchainBuild(stage *dockerfile.Stage) error {
	stage.
		Step(step.Arg("HELMDOCS_VERSION")).
		Step(step.Script(
			fmt.Sprintf(
				"go install github.com/norwoodj/helm-docs/cmd/helm-docs@${HELMDOCS_VERSION} \\\n"+
					"\t&& mv /go/bin/helm-docs %s/helm-docs", helm.meta.BinPath),
		).
			MountCache(filepath.Join(helm.meta.CachePath, "go-build"), helm.meta.GitHubRepository).
			MountCache(filepath.Join(helm.meta.GoPath, "pkg"), helm.meta.GitHubRepository),
		)

	return nil
}
