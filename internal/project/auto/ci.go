// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/project/common"
)

// DetectCI checks the ci settings.
func (builder *builder) DetectCI() (bool, error) {
	var ci CI

	if err := builder.meta.Config.Load(&ci); err != nil {
		return false, err
	}

	builder.meta.CompileGithubWorkflowsOnly = ci.CompileGHWorkflowsOnly
	builder.meta.BuildkitGithubActionsCache = ci.BuildkitGithubActionsCache

	return true, nil
}

// BuildCI builds the ci settings.
func (builder *builder) BuildCI() error {
	var targets []dag.Node

	ghw := common.NewGHWorkflow(builder.meta)
	targets = append(targets, ghw)

	if builder.meta.CompileGithubWorkflowsOnly {
		repo := common.NewRepository(builder.meta)
		repo.SetAutoContextsFunc(ghw.CollectEnforceContexts)
		repo.SetAutoLabelsFunc(ghw.CollectTriggerLabels)
		targets = append(targets, repo)
		targets = append(targets, common.NewSOPS(builder.meta))
		targets = append(targets, common.NewRenovate(builder.meta))
	}

	builder.proj.AddTarget(targets...)

	return nil
}
