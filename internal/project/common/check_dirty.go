// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/ghworkflow"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/project/meta"
)

// CheckDirty builds Makefile `check-dirty` target.
type CheckDirty struct { //nolint:govet
	dag.BaseNode

	meta *meta.Options
}

// NewCheckDirty initializes CheckDirty.
func NewCheckDirty(meta *meta.Options) *CheckDirty {
	return &CheckDirty{
		BaseNode: dag.NewBaseNode("check-dirty"),

		meta: meta,
	}
}

// CompileMakefile implements makefile.Compiler.
func (c *CheckDirty) CompileMakefile(output *makefile.Output) error {
	if c.meta.ContainerImageFrontend != config.ContainerImageFrontendDockerfile {
		return nil
	}

	checkDirtyTarget := output.Target("check-dirty").
		Phony().
		Script("@if test -n \"`git status --porcelain`\"; then echo \"Source tree is dirty\"; git status; git diff; exit 1 ; fi")

	for _, parent := range c.Parents() {
		if dag.FindByName("protobuf", parent.Inputs()...) != nil {
			checkDirtyTarget.Depends("generate")

			break
		}
	}

	return nil
}

func (c *CheckDirty) CompileGitHubWorkflow(output *ghworkflow.Output) error {
	if c.meta.ContainerImageFrontend != config.ContainerImageFrontendDockerfile {
		return nil
	}

	ciTempReleaseTagStep := ghworkflow.Step("CI temp release tag").
		SetMakeStep("ci-temp-release-tag")

	if err := ciTempReleaseTagStep.SetConditions("on-pull-request"); err != nil {
		return err
	}

	output.AddStep("default", ciTempReleaseTagStep)

	checkDirtyStep := ghworkflow.Step("Check dirty").
		SetMakeStep("check-dirty")

	if err := checkDirtyStep.SetConditions("on-pull-request"); err != nil {
		return err
	}

	output.AddStep("default", checkDirtyStep)

	return nil
}
